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
			public_id, title, title_alt, title_sort_key, visibility, summary, release_date, engine, cover_image, banner_image, series_id
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING
			id, public_id, title, title_alt, visibility, summary, release_date, engine, cover_image, banner_image,
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

	deletedAssetPaths, err := r.deleteAssetsTx(tx, id, input.Assets.DeleteAssets)
	if err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "screenshot", input.Assets.ScreenshotOrderAssetUIDs); err != nil {
		return nil, err
	}
	if err := r.reorderAssetsTx(tx, id, "video", input.Assets.VideoOrderAssetUIDs); err != nil {
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
		"engine = ?",
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
		input.Engine,
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
	if err := replaceRelationRows(tx, "game_platforms", "platform_id", gameID, input.PlatformIDs); err != nil {
		return err
	}
	if err := replaceRelationRows(tx, "game_developers", "developer_id", gameID, input.DeveloperIDs); err != nil {
		return err
	}
	if err := replaceRelationRows(tx, "game_publishers", "publisher_id", gameID, input.PublisherIDs); err != nil {
		return err
	}
	if err := replaceRelationRows(tx, "game_tags", "tag_id", gameID, input.TagIDs); err != nil {
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

func (r *GamesRepository) deleteAssetsTx(tx *sqlx.Tx, gameID int64, deleteAssets []domain.GameAssetDeleteInput) ([]string, error) {
	assetPaths := make([]string, 0, len(deleteAssets))

	for _, item := range deleteAssets {
		switch strings.TrimSpace(item.AssetType) {
		case "cover":
			if _, err := tx.Exec("UPDATE games SET cover_image = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?", gameID); err != nil {
				return nil, fmt.Errorf("delete cover image: %w", err)
			}
			assetPaths = append(assetPaths, strings.TrimSpace(item.Path))
		case "banner":
			if _, err := tx.Exec("UPDATE games SET banner_image = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?", gameID); err != nil {
				return nil, fmt.Errorf("delete banner image: %w", err)
			}
			assetPaths = append(assetPaths, strings.TrimSpace(item.Path))
		case "screenshot":
			deletedPath, _, deleted, err := r.deleteSingleAssetTx(tx, gameID, "screenshot", item)
			if err != nil {
				return nil, err
			}
			if deleted {
				assetPaths = append(assetPaths, deletedPath)
			}
		case "video":
			deletedPath, _, deleted, err := r.deleteSingleAssetTx(tx, gameID, "video", item)
			if err != nil {
				return nil, err
			}
			if deleted {
				assetPaths = append(assetPaths, deletedPath)
			}
		default:
			return nil, fmt.Errorf("invalid asset type: %s", strings.TrimSpace(item.AssetType))
		}
	}

	return assetPaths, nil
}

func (r *GamesRepository) deleteSingleAssetTx(
	tx *sqlx.Tx,
	gameID int64,
	assetType string,
	item domain.GameAssetDeleteInput,
) (string, string, bool, error) {
	trimmedUID := strings.TrimSpace(item.AssetUID)
	if trimmedUID != "" {
		var deleted struct {
			Path     string         `db:"path"`
			AssetUID sql.NullString `db:"asset_uid"`
		}
		if err := tx.Get(&deleted, `
			DELETE FROM game_assets
			WHERE game_id = ? AND asset_type = ? AND asset_uid = ?
			RETURNING path, asset_uid
		`, gameID, assetType, trimmedUID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", "", false, nil
			}
			return "", "", false, fmt.Errorf("delete %s by uid: %w", assetType, err)
		}
		return deleted.Path, deleted.AssetUID.String, true, nil
	}

	if item.AssetID != nil && *item.AssetID > 0 {
		var deleted struct {
			Path     string         `db:"path"`
			AssetUID sql.NullString `db:"asset_uid"`
		}
		if err := tx.Get(&deleted, `
			DELETE FROM game_assets
			WHERE game_id = ? AND asset_type = ? AND id = ?
			RETURNING path, asset_uid
		`, gameID, assetType, *item.AssetID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return "", "", false, nil
			}
			return "", "", false, fmt.Errorf("delete %s by id: %w", assetType, err)
		}
		return deleted.Path, deleted.AssetUID.String, true, nil
	}

	trimmedPath := strings.TrimSpace(item.Path)
	if trimmedPath == "" {
		return "", "", false, nil
	}
	var deleted struct {
		Path     string         `db:"path"`
		AssetUID sql.NullString `db:"asset_uid"`
	}
	if err := tx.Get(&deleted, `
		DELETE FROM game_assets
		WHERE game_id = ? AND asset_type = ? AND path = ?
		RETURNING path, asset_uid
	`, gameID, assetType, trimmedPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", false, nil
		}
		return "", "", false, fmt.Errorf("delete %s by path: %w", assetType, err)
	}
	return deleted.Path, deleted.AssetUID.String, true, nil
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
