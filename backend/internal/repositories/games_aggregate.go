package repositories

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

func (r *GamesRepository) Create(input domain.GameCreateInput) (*domain.Game, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("begin create game tx: %w", err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO games (
			public_id, title, title_alt, title_sort_key, visibility, summary, release_date, cover_image, banner_image, series_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING
			id, public_id, title, title_alt, visibility, summary, release_date, cover_image, banner_image,
			wiki_content, downloads, created_at, updated_at`

	var game domain.Game
	if err := tx.Get(
		&game,
		query,
		newGamePublicID(),
		input.Title,
		nil,
		buildTitleSortKey(input.Title, nil),
		input.Visibility,
		nil,
		nil,
		nil,
		nil,
		nil,
	); err != nil {
		return nil, fmt.Errorf("create game: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit create game tx: %w", err)
	}

	return r.GetByID(game.ID)
}

func (r *GamesRepository) UpdateAggregate(id int64, input domain.GameAggregateUpdateInput) ([]string, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("begin aggregate update tx: %w", err)
	}
	defer tx.Rollback()

	if err := r.updateGameRowTx(tx, id, input.Game); err != nil {
		return nil, err
	}
	if err := r.replaceRelationsTx(tx, id, input.Game); err != nil {
		return nil, err
	}
	if err := r.syncGameFilesTx(tx, id, input.Assets.Files); err != nil {
		return nil, err
	}

	// Auto-diff: delete assets in DB but not in the submitted form.
	deletedAssetPaths, err := r.diffAndDeleteAssetsTx(tx, id, input.Assets)
	if err != nil {
		return nil, err
	}

	// Auto-insert: create DB rows for new assets that don't exist yet.
	if err := r.ensureAssetsExistTx(tx, id, input.Assets.NewAssets); err != nil {
		return nil, err
	}

	if err := r.reorderAssetsTx(tx, id, "screenshot", input.Assets.ScreenshotOrderAssetUIDs); err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "video", input.Assets.VideoOrderAssetUIDs); err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "cover", input.Assets.CoverOrderAssetUIDs); err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "logo", input.Assets.LogoOrderAssetUIDs); err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "banner", input.Assets.BannerOrderAssetUIDs); err != nil {
		return nil, err
	}
	if err := r.updateLogoPositionsTx(tx, id, input.Assets.LogoPositions); err != nil {
		return nil, err
	}
	if err := r.syncPrimaryCoverTx(tx, id); err != nil {
		return nil, err
	}
	if err := r.syncPrimaryBannerTx(tx, id); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit aggregate update tx: %w", err)
	}

	return uniqueNonEmptyStrings(deletedAssetPaths), nil
}

func (r *GamesRepository) Delete(id int64) ([]string, bool, error) {
	tx, err := r.db.Beginx()
	if err != nil {
		return nil, false, fmt.Errorf("begin game delete tx: %w", err)
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
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("load game before delete: %w", err)
	}

	assetPaths := make([]string, 0, 4)
	if gameRow.CoverImage.Valid {
		assetPaths = append(assetPaths, gameRow.CoverImage.String)
	}
	if gameRow.BannerImage.Valid {
		assetPaths = append(assetPaths, gameRow.BannerImage.String)
	}

	var relatedAssetPaths []string
	if err := tx.Select(&relatedAssetPaths, `
		SELECT path
		FROM game_assets
		WHERE game_id = ? AND TRIM(path) != ''
	`, id); err != nil {
		return nil, false, fmt.Errorf("list game assets before delete: %w", err)
	}
	assetPaths = append(assetPaths, relatedAssetPaths...)

	result, err := tx.Exec("DELETE FROM games WHERE id = ?", id)
	if err != nil {
		return nil, false, fmt.Errorf("delete game: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, false, fmt.Errorf("read deleted rows: %w", err)
	}
	if rows == 0 {
		return nil, false, nil
	}

	if err := tx.Commit(); err != nil {
		return nil, false, fmt.Errorf("commit game delete tx: %w", err)
	}

	return uniqueNonEmptyStrings(assetPaths), true, nil
}

func (r *GamesRepository) updateGameRowTx(tx *sqlx.Tx, id int64, input domain.GameAggregateCoreUpdateInput) error {
	setClauses := []string{
		"title = ?",
		"title_alt = ?",
		"title_sort_key = ?",
		"visibility = ?",
		"summary = ?",
		"release_date = ?",
		"cover_image = ?",
		"banner_image = ?",
	}
	args := []any{
		input.Title,
		input.TitleAlt,
		buildTitleSortKey(input.Title, input.TitleAlt),
		input.Visibility,
		input.Summary,
		input.ReleaseDate,
		input.CoverImage,
		input.BannerImage,
	}
	setClauses = append(setClauses, "series_id = ?")
	args = append(args, input.SeriesID)
	setClauses = append(setClauses, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	query := fmt.Sprintf(`
		UPDATE games
		SET
			%s
		WHERE id = ?
	`, strings.Join(setClauses, ",\n\t\t\t"))

	result, err := tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("update game: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
	}

	return nil
}

func (r *GamesRepository) replaceRelationsTx(tx *sqlx.Tx, gameID int64, input domain.GameAggregateCoreUpdateInput) error {
	if err := replaceRelationRows(tx, "game_developers", "developer_id", gameID, input.DeveloperIDs); err != nil {
		return err
	}
	if err := replaceRelationRows(tx, "game_publishers", "publisher_id", gameID, input.PublisherIDs); err != nil {
		return err
	}
	return nil
}

func (r *GamesRepository) syncGameFilesTx(tx *sqlx.Tx, gameID int64, files []domain.GameFileUpsertInput) error {
	type existingGameFile struct {
		ID int64 `db:"id"`
	}

	var existingFiles []existingGameFile
	if err := tx.Select(&existingFiles, "SELECT id FROM game_files WHERE game_id = ?", gameID); err != nil {
		return fmt.Errorf("list game files before sync: %w", err)
	}

	keepFileIDs := make(map[int64]struct{}, len(files))
	for index, item := range files {
		sortOrder := item.SortOrder
		if sortOrder < 0 {
			sortOrder = index
		}

		if item.ID != nil && *item.ID > 0 {
			result, err := tx.Exec(`
				UPDATE game_files
				SET file_path = ?, label = ?, notes = ?, sort_order = ?, updated_at = CURRENT_TIMESTAMP
				WHERE game_id = ? AND id = ?
			`, item.FilePath, item.Label, item.Notes, sortOrder, gameID, *item.ID)
			if err != nil {
				return fmt.Errorf("update game file: %w", err)
			}
			rows, err := result.RowsAffected()
			if err != nil {
				return fmt.Errorf("read updated game file rows: %w", err)
			}
			if rows == 0 {
				return sql.ErrNoRows
			}
			keepFileIDs[*item.ID] = struct{}{}
			continue
		}

		if _, err := tx.Exec(`
			INSERT INTO game_files (game_id, file_path, label, notes, sort_order)
			VALUES (?, ?, ?, ?, ?)
		`, gameID, item.FilePath, item.Label, item.Notes, sortOrder); err != nil {
			return fmt.Errorf("create game file: %w", err)
		}
	}

	for _, file := range existingFiles {
		if _, keep := keepFileIDs[file.ID]; keep {
			continue
		}
		if _, err := tx.Exec("DELETE FROM game_files WHERE game_id = ? AND id = ?", gameID, file.ID); err != nil {
			return fmt.Errorf("delete game file: %w", err)
		}
	}

	return nil
}

// diffAndDeleteAssetsTx compares DB assets vs form-submitted UIDs for each asset type
// and deletes assets that are in DB but not in the form. Returns deleted file paths.
func (r *GamesRepository) diffAndDeleteAssetsTx(tx *sqlx.Tx, gameID int64, assets domain.GameAggregateAssetsInput) ([]string, error) {
	typeOrderMap := map[string][]string{
		"screenshot": assets.ScreenshotOrderAssetUIDs,
		"video":      assets.VideoOrderAssetUIDs,
		"cover":      assets.CoverOrderAssetUIDs,
		"logo":       assets.LogoOrderAssetUIDs,
		"banner":     assets.BannerOrderAssetUIDs,
	}

	var deletedPaths []string
	for assetType, orderUIDs := range typeOrderMap {
		// Build set of UIDs to keep.
		keepUIDs := make(map[string]struct{}, len(orderUIDs))
		for _, uid := range orderUIDs {
			trimmed := strings.TrimSpace(uid)
			if trimmed != "" {
				keepUIDs[trimmed] = struct{}{}
			}
		}

		// Query current DB assets of this type.
		var currentAssets []struct {
			Path     string         `db:"path"`
			AssetUID sql.NullString `db:"asset_uid"`
		}
		if err := tx.Select(&currentAssets, `
			SELECT path, asset_uid FROM game_assets
			WHERE game_id = ? AND asset_type = ?
		`, gameID, assetType); err != nil {
			return nil, fmt.Errorf("list current %s assets: %w", assetType, err)
		}

		// Delete assets not in the keep set.
		for _, current := range currentAssets {
			uid := strings.TrimSpace(current.AssetUID.String)
			if uid == "" {
				continue
			}
			if _, keep := keepUIDs[uid]; keep {
				continue
			}
			if _, err := tx.Exec(`
				DELETE FROM game_assets
				WHERE game_id = ? AND asset_type = ? AND asset_uid = ?
			`, gameID, assetType, uid); err != nil {
				return nil, fmt.Errorf("delete old %s asset %s: %w", assetType, uid, err)
			}
			if current.Path != "" {
				deletedPaths = append(deletedPaths, current.Path)
			}
		}
	}

	return deletedPaths, nil
}

// ensureAssetsExistTx inserts DB rows for new assets that don't exist yet.
// It matches by asset_uid — if a row with that UID already exists for this game, it's skipped.
func (r *GamesRepository) ensureAssetsExistTx(tx *sqlx.Tx, gameID int64, newAssets []domain.NewAssetEntry) error {
	for _, asset := range newAssets {
		trimmedUID := strings.TrimSpace(asset.AssetUID)
		trimmedPath := strings.TrimSpace(asset.Path)
		trimmedType := strings.TrimSpace(asset.AssetType)
		if trimmedUID == "" || trimmedPath == "" || trimmedType == "" {
			continue
		}

		// Check if this UID already exists for this game.
		var existing int
		if err := tx.Get(&existing, `
			SELECT COUNT(*) FROM game_assets
			WHERE game_id = ? AND asset_uid = ?
		`, gameID, trimmedUID); err != nil {
			return fmt.Errorf("check existing asset %s: %w", trimmedUID, err)
		}
		if existing > 0 {
			continue
		}

		if _, err := tx.Exec(`
			INSERT INTO game_assets (game_id, asset_uid, asset_type, path, sort_order)
			VALUES (?, ?, ?, ?, 0)
		`, gameID, trimmedUID, trimmedType, trimmedPath); err != nil {
			return fmt.Errorf("insert new %s asset %s: %w", trimmedType, trimmedUID, err)
		}
	}
	return nil
}

func (r *GamesRepository) reorderAssetsTx(tx *sqlx.Tx, gameID int64, assetType string, assetUIDs []string) error {
	if len(assetUIDs) == 0 {
		return nil
	}

	for index, assetUID := range assetUIDs {
		trimmedUID := strings.TrimSpace(assetUID)
		if trimmedUID == "" {
			return fmt.Errorf("empty %s asset uid", assetType)
		}

		result, err := tx.Exec(`
			UPDATE game_assets
			SET sort_order = ?
			WHERE game_id = ? AND asset_type = ? AND asset_uid = ?
		`, index, gameID, assetType, trimmedUID)
		if err != nil {
			return fmt.Errorf("update %s sort order: %w", assetType, err)
		}
		rows, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read %s reorder rows: %w", assetType, err)
		}
		if rows == 0 {
			return sql.ErrNoRows
		}
	}

	return nil
}

func (r *GamesRepository) updateLogoPositionsTx(tx *sqlx.Tx, gameID int64, positions []domain.LogoPositionInput) error {
	for _, lp := range positions {
		trimmedUID := strings.TrimSpace(lp.AssetUID)
		if trimmedUID == "" {
			continue
		}
		if _, err := tx.Exec(`
			UPDATE game_assets
			SET position_x = ?, position_y = ?, width_pct = ?
			WHERE game_id = ? AND asset_type = 'logo' AND asset_uid = ?
		`, lp.PositionX, lp.PositionY, lp.WidthPct, gameID, trimmedUID); err != nil {
			return fmt.Errorf("update logo position for %s: %w", trimmedUID, err)
		}
	}
	return nil
}

func (r *GamesRepository) syncPrimaryCoverTx(tx *sqlx.Tx, gameID int64) error {
	var primaryPath sql.NullString
	if err := tx.Get(&primaryPath, `
		SELECT path FROM game_assets
		WHERE game_id = ? AND asset_type = 'cover'
		ORDER BY sort_order ASC, id ASC
		LIMIT 1
	`, gameID); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("query primary cover: %w", err)
	}

	if _, err := tx.Exec(`
		UPDATE games SET cover_image = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, primaryPath, gameID); err != nil {
		return fmt.Errorf("sync primary cover: %w", err)
	}
	return nil
}

func (r *GamesRepository) syncPrimaryBannerTx(tx *sqlx.Tx, gameID int64) error {
	var primaryPath sql.NullString
	if err := tx.Get(&primaryPath, `
		SELECT path FROM game_assets
		WHERE game_id = ? AND asset_type = 'banner'
		ORDER BY sort_order ASC, id ASC
		LIMIT 1
	`, gameID); err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("query primary banner: %w", err)
	}

	if _, err := tx.Exec(`
		UPDATE games SET banner_image = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, primaryPath, gameID); err != nil {
		return fmt.Errorf("sync primary banner: %w", err)
	}
	return nil
}

func uniqueNonEmptyStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}

	return result
}
