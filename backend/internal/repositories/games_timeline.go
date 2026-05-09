package repositories

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

// ListTimeline returns up to limit games ordered by release_date DESC, id DESC.
// When AfterDate/AfterID are set, returns only games older than that cursor.
func (r *GamesRepository) ListTimeline(params domain.GamesTimelineParams) ([]domain.TimelineGame, error) {
	where := []string{
		"g.release_date IS NOT NULL",
		"g.release_date != ''",
	}
	// Fetch one extra so the service can detect hasMore without a second query.
	args := map[string]any{
		"limit": params.Limit + 1,
	}

	if params.AfterDate != "" && params.AfterID > 0 {
		where = append(where, "(g.release_date < :after_date OR (g.release_date = :after_date AND g.id < :after_id))")
		args["after_date"] = params.AfterDate
		args["after_id"] = params.AfterID
	}

	if !params.IncludeAll {
		visibility := strings.TrimSpace(params.Visibility)
		if visibility == "" {
			visibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = visibility
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
		return nil, fmt.Errorf("build games timeline query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var games []domain.TimelineGame
	if err := r.db.Select(&games, stmt, stmtArgs...); err != nil {
		return nil, fmt.Errorf("list timeline games: %w", err)
	}

	return games, nil
}
