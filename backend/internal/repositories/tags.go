package repositories

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

type TagsRepository struct {
	db *sqlx.DB
}

func NewTagsRepository(db *sqlx.DB) *TagsRepository {
	return &TagsRepository{db: db}
}

func (r *TagsRepository) ListGroups() ([]domain.TagGroup, error) {
	var groups []domain.TagGroup
	if err := r.db.Select(&groups, `
		SELECT id, key, name, description, sort_order, allow_multiple, is_filterable, created_at, updated_at
		FROM tag_groups
		ORDER BY sort_order ASC, id ASC
	`); err != nil {
		return nil, fmt.Errorf("list tag groups: %w", err)
	}

	return groups, nil
}

func (r *TagsRepository) CreateGroup(input domain.TagGroupWriteInput, sortOrder int, allowMultiple bool, isFilterable bool) (*domain.TagGroup, error) {
	var group domain.TagGroup
	if err := r.db.Get(&group, `
		INSERT INTO tag_groups (key, name, description, sort_order, allow_multiple, is_filterable)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING id, key, name, description, sort_order, allow_multiple, is_filterable, created_at, updated_at
	`, input.Key, input.Name, input.Description, sortOrder, boolToInt(allowMultiple), boolToInt(isFilterable)); err != nil {
		return nil, fmt.Errorf("create tag group: %w", err)
	}

	return &group, nil
}

func (r *TagsRepository) ListTags(params domain.TagsListParams) ([]domain.Tag, error) {
	where := []string{"1 = 1"}
	args := map[string]any{}

	if params.GroupID > 0 {
		where = append(where, "t.group_id = :group_id")
		args["group_id"] = params.GroupID
	}
	if strings.TrimSpace(params.GroupKey) != "" {
		where = append(where, "g.key = :group_key")
		args["group_key"] = strings.TrimSpace(params.GroupKey)
	}
	if params.Active != nil {
		where = append(where, "t.is_active = :is_active")
		args["is_active"] = boolToInt(*params.Active)
	}

	query := fmt.Sprintf(`
		SELECT
			t.id,
			t.group_id,
			g.key AS group_key,
			g.name AS group_name,
			t.name,
			t.slug,
			t.parent_id,
			t.sort_order,
			t.is_active,
			t.created_at,
			t.updated_at
		FROM tags t
		INNER JOIN tag_groups g ON g.id = t.group_id
		WHERE %s
		ORDER BY g.sort_order ASC, g.id ASC, t.sort_order ASC, t.id ASC
	`, strings.Join(where, " AND "))

	stmt, queryArgs, err := sqlx.Named(query, args)
	if err != nil {
		return nil, fmt.Errorf("build tags list query: %w", err)
	}
	stmt = r.db.Rebind(stmt)

	var tags []domain.Tag
	if err := r.db.Select(&tags, stmt, queryArgs...); err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}

	return tags, nil
}

func (r *TagsRepository) CreateTag(input domain.TagWriteInput, slug string, sortOrder int, isActive bool) (*domain.Tag, error) {
	var tag domain.Tag
	if err := r.db.Get(&tag, `
		INSERT INTO tags (group_id, name, slug, parent_id, sort_order, is_active)
		VALUES (?, ?, ?, ?, ?, ?)
		RETURNING
			id,
			group_id,
			'' AS group_key,
			'' AS group_name,
			name,
			slug,
			parent_id,
			sort_order,
			is_active,
			created_at,
			updated_at
	`, input.GroupID, input.Name, slug, input.ParentID, sortOrder, boolToInt(isActive)); err != nil {
		return nil, fmt.Errorf("create tag: %w", err)
	}

	tags, err := r.ListTags(domain.TagsListParams{GroupID: tag.GroupID})
	if err != nil {
		return nil, err
	}
	for _, item := range tags {
		if item.ID == tag.ID {
			return &item, nil
		}
	}

	return &tag, nil
}

func (r *TagsRepository) NormalizeActiveTagIDs(tagIDs []int64) ([]int64, error) {
	normalized := uniquePositiveIDs(tagIDs)
	if len(normalized) == 0 {
		return []int64{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT id
		FROM tags
		WHERE is_active = 1 AND id IN (?)
		ORDER BY id ASC
	`, normalized)
	if err != nil {
		return nil, fmt.Errorf("build active tags query: %w", err)
	}
	query = r.db.Rebind(query)

	var existing []int64
	if err := r.db.Select(&existing, query, args...); err != nil {
		return nil, fmt.Errorf("list active tags: %w", err)
	}
	if len(existing) != len(normalized) {
		return nil, fmt.Errorf("invalid tag selection")
	}

	return normalized, nil
}

func (r *TagsRepository) ValidateTagSelection(tagIDs []int64) ([]int64, error) {
	normalized, err := r.NormalizeActiveTagIDs(tagIDs)
	if err != nil {
		return nil, err
	}
	if len(normalized) == 0 {
		return []int64{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT t.id, t.group_id, g.allow_multiple
		FROM tags t
		INNER JOIN tag_groups g ON g.id = t.group_id
		WHERE t.id IN (?)
	`, normalized)
	if err != nil {
		return nil, fmt.Errorf("build tag validation query: %w", err)
	}
	query = r.db.Rebind(query)

	type row struct {
		ID            int64 `db:"id"`
		GroupID       int64 `db:"group_id"`
		AllowMultiple bool  `db:"allow_multiple"`
	}

	var rows []row
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, fmt.Errorf("validate tags: %w", err)
	}
	if len(rows) != len(normalized) {
		return nil, fmt.Errorf("invalid tag selection")
	}

	groupCounts := map[int64]int{}
	groupAllowMultiple := map[int64]bool{}
	for _, item := range rows {
		groupCounts[item.GroupID]++
		groupAllowMultiple[item.GroupID] = item.AllowMultiple
	}
	for groupID, count := range groupCounts {
		if !groupAllowMultiple[groupID] && count > 1 {
			return nil, fmt.Errorf("multiple tags selected in single-select group")
		}
	}

	return normalized, nil
}

func (r *TagsRepository) GroupTagIDs(tagIDs []int64) (map[int64][]int64, error) {
	normalized := uniquePositiveIDs(tagIDs)
	if len(normalized) == 0 {
		return map[int64][]int64{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT id, group_id
		FROM tags
		WHERE is_active = 1 AND id IN (?)
	`, normalized)
	if err != nil {
		return nil, fmt.Errorf("build grouped tags query: %w", err)
	}
	query = r.db.Rebind(query)

	type row struct {
		ID      int64 `db:"id"`
		GroupID int64 `db:"group_id"`
	}

	var rows []row
	if err := r.db.Select(&rows, query, args...); err != nil {
		return nil, fmt.Errorf("group tags: %w", err)
	}
	if len(rows) != len(normalized) {
		return nil, fmt.Errorf("missing tag ids")
	}

	grouped := make(map[int64][]int64, len(rows))
	for _, item := range rows {
		grouped[item.GroupID] = append(grouped[item.GroupID], item.ID)
	}

	for groupID := range grouped {
		sort.Slice(grouped[groupID], func(i, j int) bool {
			return grouped[groupID][i] < grouped[groupID][j]
		})
	}

	return grouped, nil
}

func (r *TagsRepository) ListByGameID(gameID int64) ([]domain.Tag, error) {
	var tags []domain.Tag
	if err := r.db.Select(&tags, `
		SELECT
			t.id,
			t.group_id,
			g.key AS group_key,
			g.name AS group_name,
			t.name,
			t.slug,
			t.parent_id,
			t.sort_order,
			t.is_active,
			t.created_at,
			t.updated_at
		FROM game_tags gt
		INNER JOIN tags t ON t.id = gt.tag_id
		INNER JOIN tag_groups g ON g.id = t.group_id
		WHERE gt.game_id = ?
		ORDER BY g.sort_order ASC, g.id ASC, gt.sort_order ASC, t.sort_order ASC, t.id ASC
	`, gameID); err != nil {
		return nil, fmt.Errorf("list game tags: %w", err)
	}

	return tags, nil
}

func uniquePositiveIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return []int64{}
	}

	seen := make(map[int64]struct{}, len(ids))
	result := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i] < result[j]
	})
	return result
}
