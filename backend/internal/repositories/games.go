package repositories

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

var allowedGameSortFields = map[string]string{
	"title":        "g.title_sort_key",
	"created_at":   "g.created_at",
	"updated_at":   "g.updated_at",
	"release_date": "g.release_date",
	"downloads":    "g.downloads",
}

type pendingIssueConditionDefinition struct {
	Key              domain.PendingIssueDetailKey
	AnyCondition     string
	VisibleCondition string
}

var pendingIssueConditionDefinitions = []pendingIssueConditionDefinition{
	newPendingFieldIssue(domain.PendingIssueDetailMissingCover, "g.cover_image"),
	newPendingFieldIssue(domain.PendingIssueDetailMissingBanner, "g.banner_image"),
	newPendingRelationIssue(domain.PendingIssueDetailMissingScreenshots, "game_assets ga", "ga.game_id = g.id AND ga.asset_type = 'screenshot'"),
	newPendingWikiIssue(),
	newPendingRelationIssue(domain.PendingIssueDetailMissingFilesList, "game_files gf", "gf.game_id = g.id"),
	newPendingRelationIssue(domain.PendingIssueDetailMissingDeveloper, "game_developers gd", "gd.game_id = g.id"),
	newPendingRelationIssue(domain.PendingIssueDetailMissingPublisher, "game_publishers gp", "gp.game_id = g.id"),
	newPendingRelationIssue(domain.PendingIssueDetailMissingPlatform, "game_platforms gp", "gp.game_id = g.id"),
	newPendingFieldIssue(domain.PendingIssueDetailMissingSummary, "g.summary"),
}

type GamesRepository struct {
	db *sqlx.DB
}

var fallbackPublicIDCounter uint64

func NewGamesRepository(db *sqlx.DB) *GamesRepository {
	return &GamesRepository{db: db}
}

func (r *GamesRepository) DB() *sqlx.DB {
	return r.db
}

// gamesListItemSelectColumns defines the shared projection for catalog-oriented list rows.
// Keep read-model specific query entry points in the split repositories instead of adding new
// business methods back onto GamesRepository.
func gamesListItemSelectColumns() string {
	return `
			g.id,
			g.public_id,
			g.title,
			g.title_alt,
			g.visibility,
			g.summary,
			g.release_date,
			g.engine,
			g.cover_image,
			g.banner_image,
			g.wiki_content,
			g.downloads,
			ss.primary_screenshot,
			COALESCE(ss.screenshot_count, 0) AS screenshot_count,
			COALESCE(fs.file_count, 0) AS file_count,
			COALESCE(ds.developer_count, 0) AS developer_count,
			COALESCE(ps.publisher_count, 0) AS publisher_count,
			COALESCE(pls.platform_count, 0) AS platform_count,
			CASE WHEN fg.game_id IS NULL THEN 0 ELSE 1 END AS is_favorite,
			g.created_at,
			g.updated_at`
}

// gameListItemStatsCTEs centralizes the CTE fragments reused by catalog list and stats queries.
// Shared SQL helpers stay here to avoid duplication, but higher-level use cases should continue to
// live in catalog/detail/timeline/aggregate repositories.
func gameListItemStatsCTEs(sourceTable string) string {
	return fmt.Sprintf(`
		ranked_screenshots AS (
			SELECT
				ga.game_id,
				ga.path,
				ROW_NUMBER() OVER (
					PARTITION BY ga.game_id
					ORDER BY ga.sort_order ASC, ga.id ASC
				) AS row_num
			FROM game_assets ga
			INNER JOIN %s src ON src.id = ga.game_id
			WHERE ga.asset_type = 'screenshot'
		),
		screenshot_stats AS (
			SELECT
				rs.game_id,
				COUNT(*) AS screenshot_count,
				MAX(CASE WHEN rs.row_num = 1 THEN rs.path END) AS primary_screenshot
			FROM ranked_screenshots rs
			GROUP BY rs.game_id
		),
		file_stats AS (
			SELECT gf.game_id, COUNT(*) AS file_count
			FROM game_files gf
			INNER JOIN %s src ON src.id = gf.game_id
			GROUP BY gf.game_id
		),
		developer_stats AS (
			SELECT gd.game_id, COUNT(*) AS developer_count
			FROM game_developers gd
			INNER JOIN %s src ON src.id = gd.game_id
			GROUP BY gd.game_id
		),
		publisher_stats AS (
			SELECT gp.game_id, COUNT(*) AS publisher_count
			FROM game_publishers gp
			INNER JOIN %s src ON src.id = gp.game_id
			GROUP BY gp.game_id
		),
		platform_stats AS (
			SELECT gp.game_id, COUNT(*) AS platform_count
			FROM game_platforms gp
			INNER JOIN %s src ON src.id = gp.game_id
			GROUP BY gp.game_id
		)
	`, sourceTable, sourceTable, sourceTable, sourceTable, sourceTable)
}

// buildGamesListWhere owns the common catalog filtering DSL used by list-like read models.
// If a new feature needs different semantics, add a dedicated read-model repository instead of
// stretching this helper into a cross-module catch-all.
func (r *GamesRepository) buildGamesListWhere(params domain.GamesListParams, excludePendingIssueFilter bool) ([]string, map[string]any, error) {
	where := []string{"1 = 1"}
	args := map[string]any{}

	if !params.IncludeAll {
		visibility := strings.TrimSpace(params.Visibility)
		if visibility == "" {
			visibility = domain.GameVisibilityPublic
		}
		where = append(where, "g.visibility = :visibility")
		args["visibility"] = visibility
	}

	if params.Search != "" {
		where = append(where, "(g.title LIKE :search OR COALESCE(g.title_alt, '') LIKE :search OR COALESCE(g.summary, '') LIKE :search)")
		args["search"] = "%" + params.Search + "%"
	}
	if params.PendingOnly {
		where = append(where, "("+pendingAnyIssueCondition(params.PendingIncludeIgnored)+")")
		if !excludePendingIssueFilter && params.PendingIssue != "" {
			pendingIssueConditions := pendingIssueConditionsForFilter(params.PendingIssue, params.PendingIncludeIgnored)
			if len(pendingIssueConditions) == 0 {
				where = append(where, "1 = 0")
			} else {
				where = append(where, "("+strings.Join(pendingIssueConditions, " OR ")+")")
			}
		}
		if params.PendingSevereOnly {
			where = append(where, "("+pendingSevereCondition()+")")
		}
		if params.PendingRecentDays > 0 {
			args["pending_recent_days"] = fmt.Sprintf("-%d days", params.PendingRecentDays)
			where = append(where, "datetime(g.created_at) >= datetime('now', :pending_recent_days)")
		}
	}
	if params.SeriesID > 0 {
		where = append(where, "g.series_id = :series_id")
		args["series_id"] = params.SeriesID
	}
	if params.PlatformID > 0 {
		where = append(where, "EXISTS (SELECT 1 FROM game_platforms gp WHERE gp.game_id = g.id AND gp.platform_id = :platform_id)")
		args["platform_id"] = params.PlatformID
	}
	if len(params.TagIDs) > 0 {
		tagFilters, tagArgs, err := r.buildTagFilters(params.TagIDs)
		if err != nil {
			return nil, nil, fmt.Errorf("build tag filters: %w", err)
		}
		where = append(where, tagFilters...)
		for key, value := range tagArgs {
			args[key] = value
		}
	}
	if params.FavoriteOnly {
		where = append(where, "EXISTS (SELECT 1 FROM favorite_games fg WHERE fg.game_id = g.id)")
	}

	return where, args, nil
}

func (r *GamesRepository) buildTagFilters(tagIDs []int64) ([]string, map[string]any, error) {
	normalized := uniquePositiveIDs(tagIDs)
	if len(normalized) == 0 {
		return nil, map[string]any{}, nil
	}

	query, queryArgs, err := sqlx.In(`
		SELECT id, group_id
		FROM tags
		WHERE is_active = 1 AND id IN (?)
	`, normalized)
	if err != nil {
		return nil, nil, fmt.Errorf("build tag grouping query: %w", err)
	}
	query = r.db.Rebind(query)

	type row struct {
		ID      int64 `db:"id"`
		GroupID int64 `db:"group_id"`
	}

	var rows []row
	if err := r.db.Select(&rows, query, queryArgs...); err != nil {
		return nil, nil, fmt.Errorf("load tag groups: %w", err)
	}
	if len(rows) != len(normalized) {
		return []string{"1 = 0"}, map[string]any{}, nil
	}

	grouped := map[int64][]int64{}
	for _, item := range rows {
		grouped[item.GroupID] = append(grouped[item.GroupID], item.ID)
	}

	groupIDs := make([]int64, 0, len(grouped))
	for groupID := range grouped {
		groupIDs = append(groupIDs, groupID)
	}
	sort.Slice(groupIDs, func(i, j int) bool {
		return groupIDs[i] < groupIDs[j]
	})

	filters := make([]string, 0, len(groupIDs))
	args := map[string]any{}
	for groupIndex, groupID := range groupIDs {
		placeholders := make([]string, 0, len(grouped[groupID]))
		for tagIndex, tagID := range grouped[groupID] {
			argKey := fmt.Sprintf("tag_%d_%d", groupIndex, tagIndex)
			args[argKey] = tagID
			placeholders = append(placeholders, ":"+argKey)
		}
		filters = append(filters, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM game_tags gt WHERE gt.game_id = g.id AND gt.tag_id IN (%s))",
			strings.Join(placeholders, ", "),
		))
	}

	return filters, args, nil
}

func replaceRelationRows(tx *sqlx.Tx, table, column string, gameID int64, ids []int64) error {
	if _, err := tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE game_id = ?", table), gameID); err != nil {
		return fmt.Errorf("clear %s: %w", table, err)
	}

	for index, id := range ids {
		if _, err := tx.Exec(
			fmt.Sprintf("INSERT INTO %s (game_id, %s, sort_order) VALUES (?, ?, ?)", table, column),
			gameID,
			id,
			index,
		); err != nil {
			return fmt.Errorf("insert %s relation: %w", table, err)
		}
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func newGamePublicID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fallbackGamePublicID()
	}

	// UUIDv4 bits.
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	hexText := hex.EncodeToString(buf)
	return fmt.Sprintf(
		"%s-%s-%s-%s-%s",
		hexText[0:8],
		hexText[8:12],
		hexText[12:16],
		hexText[16:20],
		hexText[20:32],
	)
}

func fallbackGamePublicID() string {
	now := time.Now().UnixNano()
	sequence := atomic.AddUint64(&fallbackPublicIDCounter, 1)
	return fmt.Sprintf(
		"f%07x-%04x-4%03x-a%03x-%010x%02x",
		now&0x0fffffff,
		now&0xffff,
		now&0x0fff,
		now&0x0fff,
		now&0x0fffffffff,
		sequence&0xff,
	)
}

func newPendingFieldIssue(key domain.PendingIssueDetailKey, fieldExpr string) pendingIssueConditionDefinition {
	condition := pendingMissingFieldCondition(fieldExpr)
	return pendingIssueConditionDefinition{
		Key:              key,
		AnyCondition:     condition,
		VisibleCondition: pendingVisibleIssueCondition(condition, string(key)),
	}
}

func newPendingRelationIssue(key domain.PendingIssueDetailKey, table string, predicate string) pendingIssueConditionDefinition {
	condition := pendingMissingRelationCondition(table, predicate)
	return pendingIssueConditionDefinition{
		Key:              key,
		AnyCondition:     condition,
		VisibleCondition: pendingVisibleIssueCondition(condition, string(key)),
	}
}

func newPendingWikiIssue() pendingIssueConditionDefinition {
	condition := pendingMissingWikiCondition()
	return pendingIssueConditionDefinition{
		Key:              domain.PendingIssueDetailMissingWikiContent,
		AnyCondition:     condition,
		VisibleCondition: pendingVisibleIssueCondition(condition, string(domain.PendingIssueDetailMissingWikiContent)),
	}
}

func pendingMissingFieldCondition(fieldExpr string) string {
	return fmt.Sprintf("COALESCE(TRIM(%s), '') = ''", fieldExpr)
}

func pendingMissingRelationCondition(table string, predicate string) string {
	return fmt.Sprintf("NOT EXISTS (SELECT 1 FROM %s WHERE %s)", table, predicate)
}

func pendingMissingWikiCondition() string {
	return "COALESCE(TRIM(g.wiki_content), '') = ''"
}

func pendingVisibleIssueCondition(condition string, issueKey string) string {
	return fmt.Sprintf("(%s AND %s)", condition, pendingIssueNotIgnoredCondition(issueKey))
}

func pendingIssueNotIgnoredCondition(issueKey string) string {
	return fmt.Sprintf(
		"NOT EXISTS (SELECT 1 FROM game_review_issue_overrides gio WHERE gio.game_id = g.id AND gio.issue_key = '%s' AND gio.status = 'ignored')",
		issueKey,
	)
}

func pendingAnyIssueCondition(includeIgnored bool) string {
	conditions := make([]string, 0, len(pendingIssueConditionDefinitions))
	for _, definition := range pendingIssueConditionDefinitions {
		if includeIgnored {
			conditions = append(conditions, definition.AnyCondition)
			continue
		}
		conditions = append(conditions, definition.VisibleCondition)
	}
	return strings.Join(conditions, " OR ")
}

func pendingIssueConditionsForFilter(filterKey string, includeIgnored bool) []string {
	conditions := make([]string, 0)
	for _, definition := range pendingIssueConditionDefinitions {
		if !domain.PendingIssueFilterMatches(filterKey, definition.Key) {
			continue
		}
		if includeIgnored {
			conditions = append(conditions, definition.AnyCondition)
		} else {
			conditions = append(conditions, definition.VisibleCondition)
		}
	}
	return conditions
}

func pendingGroupCondition(groupKey domain.PendingIssueKey, includeIgnored bool) string {
	conditions := pendingIssueConditionsForFilter(string(groupKey), includeIgnored)
	if len(conditions) == 0 {
		return "0 = 1"
	}
	return "(" + strings.Join(conditions, " OR ") + ")"
}

func pendingVisibleIssueCountExpression() string {
	parts := make([]string, 0, len(pendingIssueConditionDefinitions))
	for _, definition := range pendingIssueConditionDefinitions {
		parts = append(parts, fmt.Sprintf("CASE WHEN %s THEN 1 ELSE 0 END", definition.VisibleCondition))
	}
	return "(" + strings.Join(parts, " + ") + ")"
}

func pendingSevereCondition() string {
	policy := domain.PendingIssueSeverityRules()
	parts := make([]string, 0, 1+len(policy.SevereIfAnyGroup)+len(policy.SevereIfAllGroups))
	parts = append(parts, fmt.Sprintf("%s >= %d", pendingVisibleIssueCountExpression(), policy.MinVisibleDetails))

	for _, group := range policy.SevereIfAnyGroup {
		parts = append(parts, pendingGroupCondition(group, false))
	}

	for _, groupSet := range policy.SevereIfAllGroups {
		groupParts := make([]string, 0, len(groupSet))
		for _, group := range groupSet {
			groupParts = append(groupParts, pendingGroupCondition(group, false))
		}
		parts = append(parts, "("+strings.Join(groupParts, " AND ")+")")
	}

	return "(" + strings.Join(parts, " OR ") + ")"
}
