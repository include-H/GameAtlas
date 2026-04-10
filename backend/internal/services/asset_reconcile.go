package services

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/config"
	"github.com/hao/game/internal/files"
)

type AssetReconcileService struct {
	db                *sqlx.DB
	store             *files.AssetStore
	mu                sync.Mutex
	lastFullSweepAt   time.Time
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

// ReconcileGameMissingAssets remains available for explicit non-request repair callers.
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

	if !changed {
		return false, nil
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit asset reconcile tx: %w", err)
	}
	return true, nil
}
