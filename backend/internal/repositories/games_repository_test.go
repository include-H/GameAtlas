package repositories

import (
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/jmoiron/sqlx"
)

func TestGamesRepositoryUpdateAggregatePreservesOmittedRelationsAndSeries(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	gameID := insertRepositoryGame(t, db, "repo-aggregate-preserve", "Repo Aggregate Preserve", "public")

	seriesID := insertRepositorySeries(t, db, "Repo Series", "repo-series")
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, seriesID, gameID); err != nil {
		t.Fatalf("set game series: %v", err)
	}

	platformID := insertRepositoryPlatform(t, db, "Repo Windows", "repo-windows")
	linkRepositoryGamePlatform(t, db, gameID, platformID, 0)

	developerID := insertRepositoryDeveloper(t, db, "Repo Dev", "repo-dev")
	linkRepositoryGameDeveloper(t, db, gameID, developerID, 0)

	publisherID := insertRepositoryPublisher(t, db, "Repo Pub", "repo-pub")
	linkRepositoryGamePublisher(t, db, gameID, publisherID, 0)

	tagGroupID := insertRepositoryTagGroup(t, db, "repo-aggregate-preserve", "Repo Aggregate Preserve", true, true)
	tagID := insertRepositoryTag(t, db, tagGroupID, "Repo Tag", "repo-tag", true)
	linkRepositoryGameTag(t, db, gameID, tagID, 0)

	if _, err := repo.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			Title: "Repo Aggregate Preserve Updated",
		},
	}); err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}

	series, err := repo.GetSeriesMetadata(gameID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata returned error: %v", err)
	}
	if series == nil || series.ID != seriesID {
		t.Fatalf("series = %#v, want existing series preserved", series)
	}

	platforms, err := repo.ListMetadata("platforms", "game_platforms", "platform_id", gameID)
	if err != nil {
		t.Fatalf("ListMetadata(platforms) returned error: %v", err)
	}
	if len(platforms) != 1 || platforms[0].ID != platformID {
		t.Fatalf("platforms = %#v, want existing platform preserved", platforms)
	}

	developers, err := repo.ListMetadata("developers", "game_developers", "developer_id", gameID)
	if err != nil {
		t.Fatalf("ListMetadata(developers) returned error: %v", err)
	}
	if len(developers) != 1 || developers[0].ID != developerID {
		t.Fatalf("developers = %#v, want existing developer preserved", developers)
	}

	publishers, err := repo.ListMetadata("publishers", "game_publishers", "publisher_id", gameID)
	if err != nil {
		t.Fatalf("ListMetadata(publishers) returned error: %v", err)
	}
	if len(publishers) != 1 || publishers[0].ID != publisherID {
		t.Fatalf("publishers = %#v, want existing publisher preserved", publishers)
	}

	tags, err := NewTagsRepository(db).ListByGameID(gameID)
	if err != nil {
		t.Fatalf("ListByGameID returned error: %v", err)
	}
	if len(tags) != 1 || tags[0].ID != tagID {
		t.Fatalf("tags = %#v, want existing tag preserved", tags)
	}
}

func TestGamesRepositoryUpdateAggregateClearsPresentRelationsAndSeries(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	gameID := insertRepositoryGame(t, db, "repo-aggregate-clear", "Repo Aggregate Clear", "public")

	seriesID := insertRepositorySeries(t, db, "Repo Clear Series", "repo-clear-series")
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, seriesID, gameID); err != nil {
		t.Fatalf("set game series: %v", err)
	}

	developerID := insertRepositoryDeveloper(t, db, "Repo Clear Dev", "repo-clear-dev")
	linkRepositoryGameDeveloper(t, db, gameID, developerID, 0)

	if _, err := repo.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregatePatchInput{
			Title:        "Repo Aggregate Clear Updated",
			SeriesID:     domain.OptionalInt64Patch{Present: true, Value: nil},
			DeveloperIDs: domain.Int64SlicePatch{Present: true, Values: []int64{}},
		},
	}); err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}

	series, err := repo.GetSeriesMetadata(gameID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata returned error: %v", err)
	}
	if series != nil {
		t.Fatalf("series = %#v, want nil", series)
	}

	developers, err := repo.ListMetadata("developers", "game_developers", "developer_id", gameID)
	if err != nil {
		t.Fatalf("ListMetadata(developers) returned error: %v", err)
	}
	if len(developers) != 0 {
		t.Fatalf("developers = %#v, want cleared developers", developers)
	}
}

func TestGamesRepositoryLatestTimelineReleaseDateRespectsVisibility(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	_ = insertRepositoryGameWithReleaseDate(t, db, "public-old", "Public Old", "public", "2023-06-01")
	_ = insertRepositoryGameWithReleaseDate(t, db, "public-new", "Public New", "public", "2024-08-01")
	_ = insertRepositoryGameWithReleaseDate(t, db, "private-newest", "Private Newest", "private", "2025-01-01")

	publicOnly, err := repo.LatestTimelineReleaseDate(false, "")
	if err != nil {
		t.Fatalf("LatestTimelineReleaseDate(false) returned error: %v", err)
	}
	if publicOnly == nil || *publicOnly != "2024-08-01" {
		t.Fatalf("publicOnly = %v, want 2024-08-01", publicOnly)
	}

	includeAll, err := repo.LatestTimelineReleaseDate(true, "")
	if err != nil {
		t.Fatalf("LatestTimelineReleaseDate(true) returned error: %v", err)
	}
	if includeAll == nil || *includeAll != "2025-01-01" {
		t.Fatalf("includeAll = %v, want 2025-01-01", includeAll)
	}
}

func TestGamesRepositoryStatsExcludesPrivateGamesAndLoadsAssetCounts(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	firstGameID := insertRepositoryGame(t, db, "stats-a", "Stats A", "public")
	secondGameID := insertRepositoryGame(t, db, "stats-b", "Stats B", "public")
	_ = insertRepositoryGame(t, db, "stats-private", "Stats Private", "private")

	updateRepositoryGameStats(t, db, firstGameID, 10, true, "2024-01-02 00:00:00")
	updateRepositoryGameStats(t, db, secondGameID, 30, false, "2024-01-03 00:00:00")
	updateRepositoryPrivateGameStats(t, db, "stats-private", 99, true, "2024-01-04 00:00:00")

	insertRepositoryAsset(t, db, secondGameID, "screen-b2", "screenshot", "/assets/stats-b/second.png", 1)
	insertRepositoryAsset(t, db, secondGameID, "screen-b1", "screenshot", "/assets/stats-b/first.png", 0)
	insertRepositoryAsset(t, db, firstGameID, "screen-a1", "screenshot", "/assets/stats-a/only.png", 0)
	insertRepositoryGameFile(t, db, secondGameID, "/roms/stats-b.rom")

	stats, err := repo.Stats(domain.GamesListParams{})
	if err != nil {
		t.Fatalf("Stats returned error: %v", err)
	}

	if stats.TotalGames != 2 {
		t.Fatalf("TotalGames = %d, want 2", stats.TotalGames)
	}
	if stats.TotalDownloads != 40 {
		t.Fatalf("TotalDownloads = %d, want 40", stats.TotalDownloads)
	}
	if stats.PendingReviews != 1 {
		t.Fatalf("PendingReviews = %d, want 1", stats.PendingReviews)
	}

	if len(stats.RecentGames) != 2 || stats.RecentGames[0].ID != secondGameID {
		t.Fatalf("RecentGames = %+v, want second game first", stats.RecentGames)
	}
	if len(stats.PopularGames) != 2 || stats.PopularGames[0].ID != secondGameID {
		t.Fatalf("PopularGames = %+v, want second game first", stats.PopularGames)
	}
	if stats.PopularGames[0].ScreenshotCount != 2 {
		t.Fatalf("popular[0].ScreenshotCount = %d, want 2", stats.PopularGames[0].ScreenshotCount)
	}
	if stats.PopularGames[0].PrimaryScreenshot == nil || *stats.PopularGames[0].PrimaryScreenshot != "/assets/stats-b/first.png" {
		t.Fatalf("popular[0].PrimaryScreenshot = %v, want first sorted screenshot", stats.PopularGames[0].PrimaryScreenshot)
	}
	if stats.PopularGames[0].FileCount != 1 {
		t.Fatalf("popular[0].FileCount = %d, want 1", stats.PopularGames[0].FileCount)
	}
}

func TestGamesRepositoryListAppliesGroupedTagAndPlatformFilters(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	platformID := insertRepositoryPlatform(t, db, "Windows", "windows")
	otherPlatformID := insertRepositoryPlatform(t, db, "Linux", "linux")
	genreGroupID := insertRepositoryTagGroup(t, db, "repo-genre", "Repo Genre", true, true)
	themeGroupID := insertRepositoryTagGroup(t, db, "repo-theme", "Repo Theme", true, true)
	actionID := insertRepositoryTag(t, db, genreGroupID, "Action", "action", true)
	rpgID := insertRepositoryTag(t, db, genreGroupID, "RPG", "rpg", true)
	scifiID := insertRepositoryTag(t, db, themeGroupID, "Sci-Fi", "scifi", true)

	matchingID := insertRepositoryGame(t, db, "list-match", "List Match", "public")
	nonMatchingPlatformID := insertRepositoryGame(t, db, "list-platform", "List Platform", "public")
	nonMatchingThemeID := insertRepositoryGame(t, db, "list-theme", "List Theme", "public")
	privateMatchingID := insertRepositoryGame(t, db, "list-private", "List Private", "private")

	linkRepositoryGamePlatform(t, db, matchingID, platformID, 0)
	linkRepositoryGamePlatform(t, db, nonMatchingPlatformID, otherPlatformID, 0)
	linkRepositoryGamePlatform(t, db, nonMatchingThemeID, platformID, 0)
	linkRepositoryGamePlatform(t, db, privateMatchingID, platformID, 0)

	linkRepositoryGameTag(t, db, matchingID, actionID, 0)
	linkRepositoryGameTag(t, db, matchingID, scifiID, 1)
	linkRepositoryGameTag(t, db, nonMatchingPlatformID, rpgID, 0)
	linkRepositoryGameTag(t, db, nonMatchingPlatformID, scifiID, 1)
	linkRepositoryGameTag(t, db, nonMatchingThemeID, actionID, 0)
	linkRepositoryGameTag(t, db, privateMatchingID, actionID, 0)
	linkRepositoryGameTag(t, db, privateMatchingID, scifiID, 1)

	games, total, err := repo.List(domain.GamesListParams{
		Page:       1,
		Limit:      10,
		PlatformID: platformID,
		TagIDs:     []int64{rpgID, scifiID, actionID},
		Sort:       "updated_at",
		Order:      "desc",
	})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(games) != 1 || games[0].ID != matchingID {
		t.Fatalf("games = %+v, want only matching public game", games)
	}
}

func TestGamesRepositoryListTimelineAppliesCursorAndVisibility(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	newestID := insertRepositoryGameWithReleaseDate(t, db, "timeline-newest", "Timeline Newest", "public", "2024-06-01")
	middleID := insertRepositoryGameWithReleaseDate(t, db, "timeline-middle", "Timeline Middle", "public", "2024-05-01")
	_ = insertRepositoryGameWithReleaseDate(t, db, "timeline-private", "Timeline Private", "private", "2024-04-15")
	oldestID := insertRepositoryGameWithReleaseDate(t, db, "timeline-oldest", "Timeline Oldest", "public", "2024-04-01")

	games, hasMore, err := repo.ListTimeline(domain.GamesTimelineParams{
		Limit:      2,
		FromDate:   "2024-01-01",
		ToDate:     "2024-12-31",
		Visibility: domain.GameVisibilityPublic,
	})
	if err != nil {
		t.Fatalf("ListTimeline first page returned error: %v", err)
	}
	if !hasMore {
		t.Fatalf("expected hasMore=true on first page")
	}
	if len(games) != 2 || games[0].ID != newestID || games[1].ID != middleID {
		t.Fatalf("first page games = %+v, want newest then middle public games", games)
	}

	games, hasMore, err = repo.ListTimeline(domain.GamesTimelineParams{
		Limit:             2,
		FromDate:          "2024-01-01",
		ToDate:            "2024-12-31",
		Visibility:        domain.GameVisibilityPublic,
		CursorReleaseDate: "2024-05-01",
		CursorID:          middleID,
	})
	if err != nil {
		t.Fatalf("ListTimeline second page returned error: %v", err)
	}
	if hasMore {
		t.Fatalf("expected hasMore=false on second page")
	}
	if len(games) != 1 || games[0].ID != oldestID {
		t.Fatalf("second page games = %+v, want only oldest visible public game", games)
	}
}

func TestGamesRepositoryHasOlderTimelineGameHonorsVisibility(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	currentID := insertRepositoryGameWithReleaseDate(t, db, "older-current", "Older Current", "public", "2024-05-01")
	_ = insertRepositoryGameWithReleaseDate(t, db, "older-private", "Older Private", "private", "2024-04-01")

	exists, err := repo.HasOlderTimelineGame(domain.GamesTimelineParams{
		ToDate:     "2024-12-31",
		Visibility: domain.GameVisibilityPublic,
	}, "2024-05-01", currentID)
	if err != nil {
		t.Fatalf("HasOlderTimelineGame public returned error: %v", err)
	}
	if exists {
		t.Fatalf("expected public visibility to ignore older private game")
	}

	exists, err = repo.HasOlderTimelineGame(domain.GamesTimelineParams{
		ToDate:     "2024-12-31",
		IncludeAll: true,
		Visibility: domain.GameVisibilityPublic,
	}, "2024-05-01", currentID)
	if err != nil {
		t.Fatalf("HasOlderTimelineGame includeAll returned error: %v", err)
	}
	if !exists {
		t.Fatalf("expected includeAll=true to see older private game")
	}
}

func insertRepositoryGame(t *testing.T, db *sqlx.DB, publicID string, title string, visibility string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO games (public_id, title, visibility)
		VALUES (?, ?, ?)
	`, publicID, title, visibility)
	if err != nil {
		t.Fatalf("insert repository game: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertRepositoryGameWithReleaseDate(t *testing.T, db *sqlx.DB, publicID string, title string, visibility string, releaseDate string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO games (public_id, title, visibility, release_date)
		VALUES (?, ?, ?, ?)
	`, publicID, title, visibility, releaseDate)
	if err != nil {
		t.Fatalf("insert repository game with release date: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func updateRepositoryGameStats(t *testing.T, db *sqlx.DB, gameID int64, downloads int64, needsReview bool, createdAt string) {
	t.Helper()

	if _, err := db.Exec(`
		UPDATE games
		SET downloads = ?, needs_review = ?, created_at = ?, updated_at = ?
		WHERE id = ?
	`, downloads, boolToInt(needsReview), createdAt, createdAt, gameID); err != nil {
		t.Fatalf("update repository game stats: %v", err)
	}
}

func updateRepositoryPrivateGameStats(t *testing.T, db *sqlx.DB, publicID string, downloads int64, needsReview bool, createdAt string) {
	t.Helper()

	if _, err := db.Exec(`
		UPDATE games
		SET downloads = ?, needs_review = ?, created_at = ?, updated_at = ?
		WHERE public_id = ?
	`, downloads, boolToInt(needsReview), createdAt, createdAt, publicID); err != nil {
		t.Fatalf("update repository private game stats: %v", err)
	}
}

func insertRepositoryGameFile(t *testing.T, db *sqlx.DB, gameID int64, path string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO game_files (game_id, file_path)
		VALUES (?, ?)
	`, gameID, path)
	if err != nil {
		t.Fatalf("insert repository game file: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertRepositoryPlatform(t *testing.T, db *sqlx.DB, name string, slug string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO platforms (name, slug)
		VALUES (?, ?)
	`, name, slug)
	if err != nil {
		t.Fatalf("insert repository platform: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertRepositorySeries(t *testing.T, db *sqlx.DB, name string, slug string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO series (name, slug)
		VALUES (?, ?)
	`, name, slug)
	if err != nil {
		t.Fatalf("insert repository series: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertRepositoryDeveloper(t *testing.T, db *sqlx.DB, name string, slug string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO developers (name, slug)
		VALUES (?, ?)
	`, name, slug)
	if err != nil {
		t.Fatalf("insert repository developer: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func insertRepositoryPublisher(t *testing.T, db *sqlx.DB, name string, slug string) int64 {
	t.Helper()

	result, err := db.Exec(`
		INSERT INTO publishers (name, slug)
		VALUES (?, ?)
	`, name, slug)
	if err != nil {
		t.Fatalf("insert repository publisher: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId returned error: %v", err)
	}

	return id
}

func linkRepositoryGamePlatform(t *testing.T, db *sqlx.DB, gameID int64, platformID int64, sortOrder int) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO game_platforms (game_id, platform_id, sort_order)
		VALUES (?, ?, ?)
	`, gameID, platformID, sortOrder); err != nil {
		t.Fatalf("link repository game platform: %v", err)
	}
}

func linkRepositoryGameDeveloper(t *testing.T, db *sqlx.DB, gameID int64, developerID int64, sortOrder int) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO game_developers (game_id, developer_id, sort_order)
		VALUES (?, ?, ?)
	`, gameID, developerID, sortOrder); err != nil {
		t.Fatalf("link repository game developer: %v", err)
	}
}

func linkRepositoryGamePublisher(t *testing.T, db *sqlx.DB, gameID int64, publisherID int64, sortOrder int) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO game_publishers (game_id, publisher_id, sort_order)
		VALUES (?, ?, ?)
	`, gameID, publisherID, sortOrder); err != nil {
		t.Fatalf("link repository game publisher: %v", err)
	}
}

func linkRepositoryGameTag(t *testing.T, db *sqlx.DB, gameID int64, tagID int64, sortOrder int) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO game_tags (game_id, tag_id, sort_order)
		VALUES (?, ?, ?)
	`, gameID, tagID, sortOrder); err != nil {
		t.Fatalf("link repository game tag: %v", err)
	}
}
