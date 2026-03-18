package repositories

import (
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type ReviewIssueOverrideRepository struct {
	db *sqlx.DB
}

func NewReviewIssueOverrideRepository(db *sqlx.DB) *ReviewIssueOverrideRepository {
	return &ReviewIssueOverrideRepository{db: db}
}

func (r *ReviewIssueOverrideRepository) List(gameIDs []int64) ([]domain.ReviewIssueOverride, error) {
	query := `
		SELECT id, game_id, issue_key, status, reason, created_at, updated_at
		FROM game_review_issue_overrides
	`

	args := map[string]any{}
	if len(gameIDs) > 0 {
		query += ` WHERE game_id IN (:game_ids)`
		args["game_ids"] = gameIDs
	}
	query += ` ORDER BY game_id ASC, issue_key ASC`

	stmt, boundArgs, err := sqlx.Named(query, args)
	if err != nil {
		return nil, fmt.Errorf("build review override list query: %w", err)
	}
	stmt, boundArgs, err = sqlx.In(stmt, boundArgs...)
	if err != nil {
		return nil, fmt.Errorf("expand review override list query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var items []domain.ReviewIssueOverride
	if err := r.db.Select(&items, stmt, boundArgs...); err != nil {
		if strings.Contains(err.Error(), "not enough args") && len(gameIDs) == 0 {
			return []domain.ReviewIssueOverride{}, nil
		}
		return nil, fmt.Errorf("list review overrides: %w", err)
	}
	if items == nil {
		return []domain.ReviewIssueOverride{}, nil
	}
	return items, nil
}

func (r *ReviewIssueOverrideRepository) Upsert(gameID int64, issueKey, status string, reason *string) (*domain.ReviewIssueOverride, error) {
	const query = `
		INSERT INTO game_review_issue_overrides (
			game_id, issue_key, status, reason
		) VALUES (?, ?, ?, ?)
		ON CONFLICT(game_id, issue_key) DO UPDATE SET
			status = excluded.status,
			reason = excluded.reason,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, game_id, issue_key, status, reason, created_at, updated_at
	`

	var item domain.ReviewIssueOverride
	if err := r.db.Get(&item, query, gameID, issueKey, status, reason); err != nil {
		return nil, fmt.Errorf("upsert review override: %w", err)
	}
	return &item, nil
}

func (r *ReviewIssueOverrideRepository) Delete(gameID int64, issueKey string) error {
	if _, err := r.db.Exec(
		`DELETE FROM game_review_issue_overrides WHERE game_id = ? AND issue_key = ?`,
		gameID,
		issueKey,
	); err != nil {
		return fmt.Errorf("delete review override: %w", err)
	}
	return nil
}
