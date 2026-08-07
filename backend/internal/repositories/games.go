package repositories

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/hao/game/internal/domain"
)

// 2026-05-09: allowedGameSortFields 将排序 key 映射到 SQL 列表达式。合法性校验在 domain 层的 allowedGamesListSorts 中完成，此处仅负责 SQL 映射。
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

// 2026-05-09: A-6 审查结论 — pending issue 判定规则在 domain 层（Go 代码）和此处（SQL
// 条件）双重定义，逻辑等价但表达不同。不合并原因：domain 层的 EvaluatePendingIssues
// 用于详情路径的运行时评估，此处的 SQL 条件用于列表查询的数据库过滤，两者服务于不同
// 的执行上下文。统一需要重构整个评估路径，收益不足以覆盖改动风险。
var pendingIssueConditionDefinitions = []pendingIssueConditionDefinition{
	newPendingFieldIssue(domain.PendingIssueDetailMissingCover, "g.cover_image"),
	newPendingFieldIssue(domain.PendingIssueDetailMissingBanner, "g.banner_image"),
	newPendingRelationIssue(domain.PendingIssueDetailMissingScreenshots, "game_assets ga", "ga.game_id = g.id AND ga.asset_type = 'screenshot'"),
	newPendingRelationIssue(domain.PendingIssueDetailMissingLogo, "game_assets gl", "gl.game_id = g.id AND gl.asset_type = 'logo'"),
	newPendingRelationIssue(domain.PendingIssueDetailMissingVideo, "game_assets gv", "gv.game_id = g.id AND gv.asset_type = 'video'"),
	newPendingWikiIssue(),
	newPendingRelationIssue(domain.PendingIssueDetailMissingFilesList, "game_files gf", "gf.game_id = g.id"),
	newPendingRelationIssue(domain.PendingIssueDetailMissingDeveloper, "game_developers gd", "gd.game_id = g.id"),
	newPendingRelationIssue(domain.PendingIssueDetailMissingPublisher, "game_publishers gp", "gp.game_id = g.id"),
	newPendingFieldIssue(domain.PendingIssueDetailMissingSummary, "g.summary"),
}

type GamesRepository struct {
	db *sqlx.DB
}

var fallbackPublicIDCounter uint64

func NewGamesRepository(db *sqlx.DB) *GamesRepository {
	return &GamesRepository{db: db}
}

// IsAssetPathReferenced reports whether an asset file path is still referenced
// by any game row, game asset row, or start-screen tile after the originating
// game_assets row has been detached. Physical deletion must be skipped when true.
func (r *GamesRepository) IsAssetPathReferenced(assetPath string) (bool, error) {
	trimmed := strings.TrimSpace(assetPath)
	if trimmed == "" {
		return false, nil
	}

	var count int
	if err := r.db.Get(&count, `
		SELECT COUNT(*) FROM (
			SELECT 1 FROM games
			WHERE cover_image = ? OR banner_image = ?
			UNION ALL
			SELECT 1 FROM game_assets WHERE path = ?
			UNION ALL
			SELECT 1 FROM start_screen_tiles
			WHERE image_small_path = ? OR image_wide_path = ? OR image_large_path = ?
		) refs
	`, trimmed, trimmed, trimmed, trimmed, trimmed, trimmed); err != nil {
		return false, fmt.Errorf("check asset path references: %w", err)
	}

	return count > 0, nil
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
			g.cover_image,
			g.banner_image,
			g.wiki_content,
			g.downloads,
			ss.primary_screenshot,
			COALESCE(ss.screenshot_count, 0) AS screenshot_count,
			COALESCE(ls.logo_count, 0) AS logo_count,
			g.logo_visible,
			COALESCE(vs.video_count, 0) AS video_count,
			COALESCE(fs.file_count, 0) AS file_count,
			COALESCE(ds.developer_count, 0) AS developer_count,
			COALESCE(ps.publisher_count, 0) AS publisher_count,
			CASE WHEN fg.game_id IS NULL THEN 0 ELSE 1 END AS is_favorite,
			s.id AS series_id,
			s.name AS series_name,
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
		logo_stats AS (
			SELECT gl.game_id, COUNT(*) AS logo_count
			FROM game_assets gl
			INNER JOIN %s src ON src.id = gl.game_id
			WHERE gl.asset_type = 'logo'
			GROUP BY gl.game_id
		),
		video_stats AS (
			SELECT gv.game_id, COUNT(*) AS video_count
			FROM game_assets gv
			INNER JOIN %s src ON src.id = gv.game_id
			WHERE gv.asset_type = 'video'
			GROUP BY gv.game_id
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
		)
	`, sourceTable, sourceTable, sourceTable, sourceTable, sourceTable, sourceTable)
}

// buildGamesListWhere owns the common catalog filtering DSL used by list-like read models.
// If a new feature needs different semantics, add a dedicated read-model repository instead of
// stretching this helper into a cross-module catch-all.
func (r *GamesRepository) buildGamesListWhere(params domain.GamesListParams, excludePendingIssueFilter bool) ([]string, map[string]any, error) {
	where := []string{"1 = 1"}
	args := map[string]any{}

	if !params.IncludeAll || strings.TrimSpace(params.Visibility) != "" {
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
	if params.FavoriteOnly {
		// 2026-05-01: FavoriteOnly is intentionally one-way. Transport/service code must
		// reject favorite=false before reaching the repository; do not reinterpret false
		// here as "exclude favorites" or add fallback behavior.
		where = append(where, "EXISTS (SELECT 1 FROM favorite_games fg WHERE fg.game_id = g.id)")
	}

	return where, args, nil
}

func replaceRelationRows(tx *sqlx.Tx, typ domain.MetadataType, gameID int64, ids []int64) error {
	table := metadataJoinTable(typ)
	column := metadataJoinColumn(typ)
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

// 2026-05-09: the 'ignored' status string matches domain.ReviewIssueStatusIgnored.
// Keep the literal here because SQL parameters cannot reference Go constants directly.
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
