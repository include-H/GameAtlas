package services

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/files"
	"github.com/hao/game/internal/repositories"
)

type AssetReconcileService struct {
	db                *sqlx.DB
	store             *files.AssetStore
	tasksRepo         *repositories.AssetCleanupTasksRepository
	mu                sync.Mutex
	lastFullSweepAt   time.Time
	lastOrphanSweepAt time.Time
	fullSweepCooldown time.Duration
}

// NewAssetReconcileService keeps DB asset references aligned with the current filesystem state.
// 2026-04-06: this boundary is startup-only.
// Impact: boot-time repair cleans stale asset references after manual file deletion;
// list/stats/detail reads must stay side-effect free and never mutate DB state.
func NewAssetReconcileService(cfg config.Config, db *sqlx.DB) *AssetReconcileService {
	return &AssetReconcileService{
		db:                db,
		store:             files.NewAssetStore(cfg.AssetsDir, cfg.Proxy, 30*time.Second),
		tasksRepo:         repositories.NewAssetCleanupTasksRepository(db),
		fullSweepCooldown: 2 * time.Second,
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

// CleanOrphanedAssetFiles scans the assets filesystem and deletes files not
// referenced by any game in the database. It runs at startup after
// ReconcileAllMissingAssets has already pruned stale DB references.
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

	deleted, err := s.deleteUnreferencedFiles(referenced)
	if err != nil {
		return deleted, err
	}

	s.lastOrphanSweepAt = time.Now()
	return deleted, nil
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

func (s *AssetReconcileService) deleteUnreferencedFiles(referenced map[string]struct{}) (int, error) {
	baseDir := s.store.BaseDir()

	if _, statErr := os.Stat(baseDir); os.IsNotExist(statErr) {
		return 0, nil
	}

	var dirs []string
	deleted := 0

	err := filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
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

		if _, ok := referenced[assetPath]; ok {
			return nil
		}

		if _, cleanupErr := cleanupAssetPath(s.store, s.tasksRepo, assetPath, "asset_reconcile.orphan"); cleanupErr != nil {
			log.Printf("asset_reconcile.orphan: failed to delete %s: %v", assetPath, cleanupErr)
			return nil
		}
		deleted++
		return nil
	})
	if err != nil {
		return deleted, fmt.Errorf("walk assets directory: %w", err)
	}

	// Remove empty game directories (reverse order = deepest first).
	for i := len(dirs) - 1; i >= 0; i-- {
		entries, readErr := os.ReadDir(dirs[i])
		if readErr == nil && len(entries) == 0 {
			_ = os.Remove(dirs[i])
		}
	}

	return deleted, nil
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
