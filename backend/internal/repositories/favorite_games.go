package repositories

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/hao/game/internal/domain"
)

type FavoriteGamesRepository struct {
	db favoriteGamesExecutor
}

type favoriteGamesExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Get(dest any, query string, args ...any) error
	Select(dest any, query string, args ...any) error
}

func NewFavoriteGamesRepository(db favoriteGamesExecutor) *FavoriteGamesRepository {
	return &FavoriteGamesRepository{db: db}
}

func (r *FavoriteGamesRepository) Set(gameID int64, isFavorite bool) error {
	if isFavorite {
		if _, err := r.db.Exec(`
			INSERT INTO favorite_games (game_id)
			VALUES (?)
			ON CONFLICT(game_id) DO NOTHING
		`, gameID); err != nil {
			return fmt.Errorf("set favorite game: %w", err)
		}
		return nil
	}

	if _, err := r.db.Exec(`
		DELETE FROM favorite_games
		WHERE game_id = ?
	`, gameID); err != nil {
		return fmt.Errorf("delete favorite game: %w", err)
	}

	return nil
}

func (r *FavoriteGamesRepository) IsFavorite(gameID int64) (bool, error) {
	var count int
	if err := r.db.Get(&count, `
		SELECT COUNT(*)
		FROM favorite_games
		WHERE game_id = ?
	`, gameID); err != nil {
		return false, fmt.Errorf("check favorite game: %w", err)
	}
	return count > 0, nil
}

func (r *FavoriteGamesRepository) Count(includeAll bool, visibility string) (int, error) {
	var count int
	query := `
		SELECT COUNT(*)
		FROM favorite_games fg
		INNER JOIN games g ON g.id = fg.game_id
	`
	args := []any{}
	if !includeAll {
		resolvedVisibility := strings.TrimSpace(visibility)
		if resolvedVisibility == "" {
			resolvedVisibility = domain.GameVisibilityPublic
		}
		query += `
		WHERE g.visibility = ?
	`
		args = append(args, resolvedVisibility)
	}

	if err := r.db.Get(&count, query, args...); err != nil {
		return 0, fmt.Errorf("count favorite games: %w", err)
	}
	return count, nil
}
