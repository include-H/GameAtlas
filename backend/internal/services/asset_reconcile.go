package services

import (
	"database/sql"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/files"
)

type AssetReconcileService struct {
	db                  *sqlx.DB
	store               *files.AssetStore
	mu                  sync.Mutex
	lastFullSweepAt     time.Time
	lastOrphanSweepAt   time.Time
	fullSweepCooldown   time.Duration
	orphanQuarantineDir string
}

const orphanAssetQuarantineRetention = 7 * 24 * time.Hour

// NewAssetReconcileService keeps DB asset references aligned with the current filesystem state.
// 2026-04-06: this boundary is startup-only.
// Impact: boot-time repair cleans stale asset references after manual file deletion;
// list/stats/detail reads must stay side-effect free and never mutate DB state.
func NewAssetReconcileService(cfg config.Config, db *sqlx.DB) *AssetReconcileService {
	return &AssetReconcileService{
		db:                  db,
		store:               files.NewAssetStore(cfg.AssetsDir),
		fullSweepCooldown:   2 * time.Second,
		orphanQuarantineDir: filepath.Join(filepath.Dir(cfg.AssetsDir), "orphaned-assets"),
	}
}

// ReconcileAllMissingAssets performs a bounded full sweep for startup repair only.
func (s *AssetReconcileService) ReconcileAllMissingAssets() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.lastFullSweepAt.IsZero() && time.Since(s.lastFullSweepAt) < s.fullSweepCooldown {
		return 0, nil
	}

	gameIDs, err := s.loadGameIDsWithAssetReferences()
	if err != nil {
		return 0, err
	}

	changed := 0
	for _, gameID := range gameIDs {
		gameChanged, err := s.reconcileGameMissingAssetsTx(gameID)
		if err != nil {
			return changed, err
		}
		if gameChanged {
			changed++
		}
	}

	s.lastFullSweepAt = time.Now()
	return changed, nil
}

// ReconcileGameMissingAssets reconciles a single game's asset references.
// NOTE: currently only exercised by tests; production startup uses ReconcileAllMissingAssets.
func (s *AssetReconcileService) ReconcileGameMissingAssets(gameID int64) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reconcileGameMissingAssetsTx(gameID)
}

func (s *AssetReconcileService) loadGameIDsWithAssetReferences() ([]int64, error) {
	var gameIDs []int64
	if err := s.db.Select(&gameIDs, `
		SELECT DISTINCT game_id
		FROM (
			SELECT id AS game_id
			FROM games
			WHERE COALESCE(TRIM(cover_image), '') != '' OR COALESCE(TRIM(banner_image), '') != ''
			UNION
			SELECT game_id
			FROM game_assets
			WHERE COALESCE(TRIM(path), '') != ''
		) refs
		ORDER BY game_id ASC
	`); err != nil {
		return nil, fmt.Errorf("list games with asset references: %w", err)
	}
	return gameIDs, nil
}

func (s *AssetReconcileService) reconcileGameMissingAssetsTx(gameID int64) (bool, error) {
	tx, err := s.db.Beginx()
	if err != nil {
		return false, fmt.Errorf("begin asset reconcile tx: %w", err)
	}
	defer tx.Rollback()

	var gameRow struct {
		CoverImage  sql.NullString `db:"cover_image"`
		BannerImage sql.NullString `db:"banner_image"`
	}
	if err := tx.Get(&gameRow, `
		SELECT cover_image, banner_image
		FROM games
		WHERE id = ?
	`, gameID); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("load game asset references: %w", err)
	}

	changed := false
	if gameRow.CoverImage.Valid && !s.store.AssetExists(gameRow.CoverImage.String) {
		if _, err := tx.Exec(`
			UPDATE games
			SET cover_image = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, gameID); err != nil {
			return false, fmt.Errorf("clear missing cover image: %w", err)
		}
		changed = true
	}
	if gameRow.BannerImage.Valid && !s.store.AssetExists(gameRow.BannerImage.String) {
		if _, err := tx.Exec(`
			UPDATE games
			SET banner_image = NULL, updated_at = CURRENT_TIMESTAMP
			WHERE id = ?
		`, gameID); err != nil {
			return false, fmt.Errorf("clear missing banner image: %w", err)
		}
		changed = true
	}

	var assetRows []struct {
		ID   int64  `db:"id"`
		Path string `db:"path"`
	}
	if err := tx.Select(&assetRows, `
		SELECT id, path
		FROM game_assets
		WHERE game_id = ? AND COALESCE(TRIM(path), '') != ''
	`, gameID); err != nil {
		return false, fmt.Errorf("list game assets for reconcile: %w", err)
	}

	for _, row := range assetRows {
		if s.store.AssetExists(row.Path) {
			continue
		}
		if _, err := tx.Exec(`DELETE FROM game_assets WHERE id = ?`, row.ID); err != nil {
			return false, fmt.Errorf("delete missing asset row: %w", err)
		}
		changed = true
	}

	// Sync cover_image column with the primary cover from game_assets after cleanup.
	if changed {
		var primaryCover sql.NullString
		if err := tx.Get(&primaryCover, `
			SELECT path FROM game_assets
			WHERE game_id = ? AND asset_type = 'cover'
			ORDER BY sort_order ASC, id ASC
			LIMIT 1
		`, gameID); err != nil && err != sql.ErrNoRows {
			return false, fmt.Errorf("query primary cover for sync: %w", err)
		}
		if _, err := tx.Exec(`
			UPDATE games SET cover_image = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
		`, primaryCover, gameID); err != nil {
			return false, fmt.Errorf("sync cover_image after reconcile: %w", err)
		}
	}

	if !changed {
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit asset reconcile tx: %w", err)
	}
	return true, nil
}

// CleanStaging deletes files in the staging directory older than 1 hour.
// Staging files are ephemeral upload artifacts; if they survive past the
// upload session they are orphans from abandoned edits.
func (s *AssetReconcileService) CleanStaging() (int, error) {
	return s.store.CleanStaging(1 * time.Hour)
}

// CleanOrphanedAssetFiles moves unreferenced assets to a fixed filesystem
// quarantine for seven days. It intentionally has no restore workflow: the
// directory is a short-lived investigation buffer for NAS file browsing.
func (s *AssetReconcileService) CleanOrphanedAssetFiles() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.lastOrphanSweepAt.IsZero() && time.Since(s.lastOrphanSweepAt) < s.fullSweepCooldown {
		return 0, nil
	}
	referenced, err := s.loadAllReferencedAssetPaths()
	if err != nil {
		return 0, err
	}

	processed, err := s.deleteUnreferencedFiles(referenced)
	if err != nil {
		return processed, err
	}
	if _, err := s.cleanOldQuarantinedAssets(); err != nil {
		return processed, err
	}

	s.lastOrphanSweepAt = time.Now()
	return processed, nil
}

func (s *AssetReconcileService) loadAllReferencedAssetPaths() (map[string]struct{}, error) {
	var paths []string
	if err := s.db.Select(&paths, `
		SELECT path FROM (
			SELECT cover_image AS path FROM games WHERE COALESCE(TRIM(cover_image), '') != ''
			UNION
			SELECT banner_image AS path FROM games WHERE COALESCE(TRIM(banner_image), '') != ''
			UNION
			SELECT path FROM game_assets WHERE COALESCE(TRIM(path), '') != ''
			UNION
			SELECT poster_path AS path FROM game_assets WHERE COALESCE(TRIM(poster_path), '') != ''
			UNION
			SELECT image_path AS path FROM start_screen_tiles WHERE COALESCE(TRIM(image_path), '') != ''
			UNION
			SELECT value AS path FROM start_screen_tiles, json_each(start_screen_tiles.flip_images)
			WHERE COALESCE(TRIM(value), '') != ''
			UNION
			SELECT image_small_path AS path FROM start_screen_tiles WHERE COALESCE(TRIM(image_small_path), '') != ''
			UNION
			SELECT image_wide_path AS path FROM start_screen_tiles WHERE COALESCE(TRIM(image_wide_path), '') != ''
			UNION
			SELECT image_large_path AS path FROM start_screen_tiles WHERE COALESCE(TRIM(image_large_path), '') != ''
		) refs
	`); err != nil {
		return nil, fmt.Errorf("load all referenced asset paths: %w", err)
	}

	set := make(map[string]struct{}, len(paths))
	for _, p := range paths {
		set[strings.TrimSpace(p)] = struct{}{}
	}
	return set, nil
}

var knownAssetExtensions = map[string]struct{}{
	".jpg": {}, ".jpeg": {}, ".png": {}, ".webp": {}, ".gif": {},
	".mp4": {}, ".webm": {},
}

func isKnownAssetFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	_, ok := knownAssetExtensions[ext]
	return ok
}

// 变体缓存文件名形如 {uuid}.w480.webp（EnsureVariant 按需生成）。它们不是用户数据，
// 也不在任何 DB 引用表里。清扫规则：变体跟随原图——原图仍被引用则变体保留（按需再生成
// 的缓存）；原图已无人引用则变体与原图一起移入隔离区，不残留死缓存。
var variantAssetFilePattern = regexp.MustCompile(`(?i)^(.+)\.w\d+\.(jpe?g|png|webp|gif)$`)

// variantBaseCandidates 还原变体可能的原图路径。变体名丢弃了原图扩展名
// （cover.jpg → cover.w480.webp），因此按已知图片扩展名逐一生成候选，
// 与 DB 引用表比对时命中任意一个即视为原图仍被引用。
func variantBaseCandidates(assetPath string) []string {
	match := variantAssetFilePattern.FindStringSubmatch(filepath.Base(assetPath))
	if match == nil {
		return nil
	}
	stem := match[1]
	if ext := filepath.Ext(stem); ext != "" {
		stem = strings.TrimSuffix(stem, ext)
	}
	baseDir := filepath.ToSlash(filepath.Dir(assetPath))
	candidates := make([]string, 0, len(knownAssetExtensions))
	for ext := range knownAssetExtensions {
		candidates = append(candidates, baseDir+"/"+stem+ext)
	}
	return candidates
}

func (s *AssetReconcileService) deleteUnreferencedFiles(referenced map[string]struct{}) (int, error) {
	baseDir := s.store.BaseDir()

	if _, statErr := os.Stat(baseDir); os.IsNotExist(statErr) {
		return 0, nil
	}

	var dirs []string
	processed := 0
	sweepQuarantineDir := filepath.Join(
		s.orphanQuarantineDir,
		time.Now().UTC().Format("20060102-150405"),
	)

	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if filepath.Clean(path) == filepath.Join(baseDir, "_staging") {
				return filepath.SkipDir
			}
			if path != baseDir {
				dirs = append(dirs, path)
			}
			return nil
		}

		if !isKnownAssetFile(d.Name()) {
			return nil
		}
		assetPath, pathErr := fsPathToAssetPath(baseDir, path)
		if pathErr != nil {
			return nil
		}

		// 变体跟随原图：原图仍被引用则保留缓存，否则与原图一起被隔离。
		if bases := variantBaseCandidates(assetPath); bases != nil {
			kept := false
			for _, basePath := range bases {
				if _, ok := referenced[basePath]; ok {
					kept = true
					break
				}
			}
			if kept {
				return nil
			}
		}

		if _, ok := referenced[assetPath]; ok {
			return nil
		}

		if processErr := quarantineAssetFile(baseDir, path, sweepQuarantineDir); processErr != nil {
			log.Printf("asset_reconcile.orphan: failed to process %s: %v", assetPath, processErr)
			return nil
		}
		processed++
		return nil
	})
	if err != nil {
		return processed, fmt.Errorf("walk assets directory: %w", err)
	}

	// Remove empty game directories (reverse order = deepest first).
	for i := len(dirs) - 1; i >= 0; i-- {
		entries, readErr := os.ReadDir(dirs[i])
		if readErr == nil && len(entries) == 0 {
			_ = os.Remove(dirs[i])
		}
	}

	return processed, nil
}

func quarantineAssetFile(baseDir, sourcePath, sweepQuarantineDir string) error {
	rel, err := filepath.Rel(baseDir, sourcePath)
	if err != nil {
		return err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("path %q is outside base dir", sourcePath)
	}

	targetPath := filepath.Join(sweepQuarantineDir, rel)
	targetPath = uniqueFilePath(targetPath)
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create orphan quarantine directory: %w", err)
	}
	if err := moveFile(sourcePath, targetPath); err != nil {
		return fmt.Errorf("move orphan to quarantine: %w", err)
	}
	return nil
}

func moveFile(sourcePath, targetPath string) error {
	if err := os.Rename(sourcePath, targetPath); err == nil {
		return nil
	}

	input, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(output, input)
	closeErr := output.Close()
	if copyErr != nil {
		_ = os.Remove(targetPath)
		return copyErr
	}
	if closeErr != nil {
		_ = os.Remove(targetPath)
		return closeErr
	}
	return os.Remove(sourcePath)
}

func uniqueFilePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d%s", base, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

func (s *AssetReconcileService) cleanOldQuarantinedAssets() (int, error) {
	if strings.TrimSpace(s.orphanQuarantineDir) == "" {
		return 0, nil
	}
	if _, err := os.Stat(s.orphanQuarantineDir); os.IsNotExist(err) {
		return 0, nil
	} else if err != nil {
		return 0, fmt.Errorf("stat orphan quarantine directory: %w", err)
	}

	cutoff := time.Now().Add(-orphanAssetQuarantineRetention)
	removed := 0
	err := filepath.WalkDir(s.orphanQuarantineDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, statErr := d.Info()
		if statErr != nil {
			return statErr
		}
		if info.ModTime().After(cutoff) {
			return nil
		}
		if removeErr := os.Remove(path); removeErr != nil {
			return removeErr
		}
		removed++
		return nil
	})
	if err != nil {
		return removed, fmt.Errorf("clean old quarantined assets: %w", err)
	}

	removeEmptyDirs(s.orphanQuarantineDir)
	return removed, nil
}

func removeEmptyDirs(baseDir string) {
	var dirs []string
	_ = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && d.IsDir() && path != baseDir {
			dirs = append(dirs, path)
		}
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		entries, readErr := os.ReadDir(dirs[i])
		if readErr == nil && len(entries) == 0 {
			_ = os.Remove(dirs[i])
		}
	}
}

// fsPathToAssetPath converts an absolute filesystem path under baseDir to the
// URL-style asset path used by AssetStore (e.g. "/assets/game-id/file.jpg").
func fsPathToAssetPath(baseDir string, fsPath string) (string, error) {
	rel, err := filepath.Rel(baseDir, fsPath)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q is outside base dir", fsPath)
	}
	return "/assets/" + filepath.ToSlash(rel), nil
}
