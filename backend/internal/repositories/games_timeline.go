package repositories

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

func (r *GamesRepository) ListTimeline(params domain.GamesTimelineParams) ([]domain.TimelineGame, bool, error) {
	where := []string{
		"g.release_date IS NOT NULL",
		"g.release_date != ''",
		"g.release_date >= :from_date",
		"g.release_date <= :to_date",
	}
	args := map[string]any{
		"from_date": params.FromDate,
		"to_date":   params.ToDate,
		"limit":     params.Limit + 1,
	}

	if !params.IncludeAll {
		visibility := strings.TrimSpace(params.Visibility)
		if visibility == "" {
			visibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = visibility
	}

	if params.CursorReleaseDate != "" && params.CursorID > 0 {
		where = append(where, "(g.release_date < :cursor_release_date OR (g.release_date = :cursor_release_date AND g.id < :cursor_id))")
		args["cursor_release_date"] = params.CursorReleaseDate
		args["cursor_id"] = params.CursorID
	}

	query := fmt.Sprintf(`
		SELECT
			g.id,
			g.public_id,
			g.title,
			g.release_date,
			g.cover_image,
			g.banner_image
		FROM games g
		WHERE %s
		ORDER BY g.release_date DESC, g.id DESC
		LIMIT :limit
	`, strings.Join(where, " AND "))

	stmt, stmtArgs, err := sqlx.Named(query, args)
	if err != nil {
		return nil, false, fmt.Errorf("build games timeline query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var games []domain.TimelineGame
	if err := r.db.Select(&games, stmt, stmtArgs...); err != nil {
		return nil, false, fmt.Errorf("list timeline games: %w", err)
	}

	hasMore := len(games) > params.Limit
	if hasMore {
		games = games[:params.Limit]
	}

	return games, hasMore, nil
}

func (r *GamesRepository) LatestTimelineReleaseDate(includeAll bool, visibility string) (*string, error) {
	where := []string{
		"g.release_date IS NOT NULL",
		"g.release_date != ''",
	}
	args := map[string]any{}

	if !includeAll {
		targetVisibility := strings.TrimSpace(visibility)
		if targetVisibility == "" {
			targetVisibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = targetVisibility
	}

	query := fmt.Sprintf(`
		SELECT g.release_date
		FROM games g
		WHERE %s
		ORDER BY g.release_date DESC, g.id DESC
		LIMIT 1
	`, strings.Join(where, " AND "))

	stmt, stmtArgs, err := sqlx.Named(query, args)
	if err != nil {
		return nil, fmt.Errorf("build latest timeline release date query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var releaseDate string
	if err := r.db.Get(&releaseDate, stmt, stmtArgs...); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest timeline release date: %w", err)
	}

	trimmed := strings.TrimSpace(releaseDate)
	if trimmed == "" {
		return nil, nil
	}

	return &trimmed, nil
}

func (r *GamesRepository) HasOlderTimelineGame(params domain.GamesTimelineParams, cursorReleaseDate string, cursorID int64) (bool, error) {
	where := []string{
		"g.release_date IS NOT NULL",
		"g.release_date != ''",
		"g.release_date <= :to_date",
		"(g.release_date < :cursor_release_date OR (g.release_date = :cursor_release_date AND g.id < :cursor_id))",
	}
	args := map[string]any{
		"to_date":             params.ToDate,
		"cursor_release_date": cursorReleaseDate,
		"cursor_id":           cursorID,
	}

	if !params.IncludeAll {
		targetVisibility := strings.TrimSpace(params.Visibility)
		if targetVisibility == "" {
			targetVisibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = targetVisibility
	}

	query := fmt.Sprintf(`
		SELECT 1
		FROM games g
		WHERE %s
		LIMIT 1
	`, strings.Join(where, " AND "))

	stmt, stmtArgs, err := sqlx.Named(query, args)
	if err != nil {
		return false, fmt.Errorf("build older timeline exists query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var value int
	if err := r.db.Get(&value, stmt, stmtArgs...); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("check older timeline game exists: %w", err)
	}

	return true, nil
}
