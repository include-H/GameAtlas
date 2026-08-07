package repositories

import (
	"testing"
	"time"

	"github.com/hao/game/internal/domain"
	"github.com/jmoiron/sqlx"
)

func TestGamesRepositoryUpdateAggregateReplacesRelationsAndSeries(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	gameID := insertRepositoryGame(t, db, "repo-aggregate-preserve", "Repo Aggregate Preserve", "public")

	seriesID := insertRepositorySeries(t, db, "Repo Series", "repo-series")
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, seriesID, gameID); err != nil {
		t.Fatalf("set game series: %v", err)
	}

	developerID := insertRepositoryDeveloper(t, db, "Repo Dev", "repo-dev")
	linkRepositoryGameDeveloper(t, db, gameID, developerID, 0)

	publisherID := insertRepositoryPublisher(t, db, "Repo Pub", "repo-pub")
	linkRepositoryGamePublisher(t, db, gameID, publisherID, 0)

	if _, err := repo.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Repo Aggregate Preserve Updated"},
			SeriesID:      nil,
			DeveloperIDs:  []int64{},
			PublisherIDs:  []int64{},
		},
	}); err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}

	series, err := repo.GetSeriesMetadata(gameID)
	if err != nil {
		t.Fatalf("GetSeriesMetadata returned error: %v", err)
	}
	if series != nil {
		t.Fatalf("series = %#v, want cleared series", series)
	}

	developers, err := repo.ListMetadata(domain.MetadataDevelopers, gameID)
	if err != nil {
		t.Fatalf("ListMetadata(developers) returned error: %v", err)
	}
	if len(developers) != 0 {
		t.Fatalf("developers = %#v, want cleared developers", developers)
	}

	publishers, err := repo.ListMetadata(domain.MetadataPublishers, gameID)
	if err != nil {
		t.Fatalf("ListMetadata(publishers) returned error: %v", err)
	}
	if len(publishers) != 0 {
		t.Fatalf("publishers = %#v, want cleared publishers", publishers)
	}
}

func TestGamesRepositoryUpdateAggregateClearsPresentRelationsAndSeries(t *testing.T) {
	db := openRepositoryTestDB(t)
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
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Repo Aggregate Clear Updated"},
			SeriesID:      nil,
			DeveloperIDs:  []int64{},
			PublisherIDs:  []int64{},
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

	developers, err := repo.ListMetadata(domain.MetadataDevelopers, gameID)
	if err != nil {
		t.Fatalf("ListMetadata(developers) returned error: %v", err)
	}
	if len(developers) != 0 {
		t.Fatalf("developers = %#v, want cleared developers", developers)
	}
}

func TestGamesRepositoryStatsExcludesPrivateGamesAndLoadsAssetCounts(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	firstGameID := insertRepositoryGame(t, db, "stats-a", "Stats A", "public")
	secondGameID := insertRepositoryGame(t, db, "stats-b", "Stats B", "public")
	privateGameID := insertRepositoryGame(t, db, "stats-private", "Stats Private", "private")

	updateRepositoryGameStats(t, db, firstGameID, 10, "2024-01-02 00:00:00")
	updateRepositoryGameStats(t, db, secondGameID, 30, "2024-01-03 00:00:00")
	updateRepositoryPrivateGameStats(t, db, "stats-private", 99, "2024-01-04 00:00:00")
	// 让第二个游戏成为“最近完善”的唯一候选：updated_at 明显晚于 created_at。
	if _, err := db.Exec(`UPDATE games SET updated_at = ? WHERE id = ?`, "2024-02-01 00:00:00", secondGameID); err != nil {
		t.Fatalf("set stats game updated_at: %v", err)
	}

	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?, banner_image = ?, summary = ?, wiki_content = ?
		WHERE id = ?
	`, "/assets/stats-b/cover.png", "/assets/stats-b/banner.png", "Ready", "# Ready", secondGameID); err != nil {
		t.Fatalf("seed resolved stats game: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?, banner_image = ?, summary = ?, wiki_content = ?
		WHERE id = ?
	`, "/assets/stats-private/cover.png", "/assets/stats-private/banner.png", "Private Ready", "# Private Ready", privateGameID); err != nil {
		t.Fatalf("seed private stats game: %v", err)
	}

	insertRepositoryAsset(t, db, secondGameID, "screen-b2", "screenshot", "/assets/stats-b/second.png", 1)
	insertRepositoryAsset(t, db, secondGameID, "screen-b1", "screenshot", "/assets/stats-b/first.png", 0)
	insertRepositoryAsset(t, db, secondGameID, "logo-b", "logo", "/assets/stats-b/logo.png", 0)
	insertRepositoryAsset(t, db, secondGameID, "video-b", "video", "/assets/stats-b/trailer.mp4", 0)
	insertRepositoryAsset(t, db, firstGameID, "screen-a1", "screenshot", "/assets/stats-a/only.png", 0)
	insertRepositoryAsset(t, db, privateGameID, "screen-private", "screenshot", "/assets/stats-private/only.png", 0)
	insertRepositoryGameFile(t, db, secondGameID, "/roms/stats-b.rom")
	insertRepositoryGameFile(t, db, privateGameID, "/roms/stats-private.rom")
	if _, err := db.Exec(`INSERT INTO favorite_games (game_id) VALUES (?)`, secondGameID); err != nil {
		t.Fatalf("insert favorite game: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO favorite_games (game_id) VALUES (?)`, privateGameID); err != nil {
		t.Fatalf("insert private favorite game: %v", err)
	}

	developerID := insertRepositoryDeveloper(t, db, "Stats Developer", "stats-developer")
	publisherID := insertRepositoryPublisher(t, db, "Stats Publisher", "stats-publisher")
	linkRepositoryGameDeveloper(t, db, secondGameID, developerID, 0)
	linkRepositoryGamePublisher(t, db, secondGameID, publisherID, 0)
	linkRepositoryGameDeveloper(t, db, privateGameID, developerID, 0)
	linkRepositoryGamePublisher(t, db, privateGameID, publisherID, 0)
	seriesID := insertRepositorySeries(t, db, "Stats Series", "stats-series")
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, seriesID, secondGameID); err != nil {
		t.Fatalf("set stats game series: %v", err)
	}

	catalogRepo := NewGameCatalogRepository(repo, NewFavoriteGamesRepository(db))

	stats, err := catalogRepo.Stats(domain.GamesListParams{})
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
		t.Fatalf("PendingReviews = %d, want 1 native pending public game", stats.PendingReviews)
	}
	if stats.FavoriteCount != 1 {
		t.Fatalf("FavoriteCount = %d, want 1 visible public favorite", stats.FavoriteCount)
	}

	if len(stats.RecentGames) != 2 || stats.RecentGames[0].ID != secondGameID {
		t.Fatalf("RecentGames = %+v, want second game first", stats.RecentGames)
	}
	if len(stats.RecentlyUpdatedGames) != 1 || stats.RecentlyUpdatedGames[0].ID != secondGameID {
		t.Fatalf("RecentlyUpdatedGames = %+v, want only second game", stats.RecentlyUpdatedGames)
	}
	if len(stats.PopularGames) != 2 || stats.PopularGames[0].ID != secondGameID {
		t.Fatalf("PopularGames = %+v, want second game first", stats.PopularGames)
	}
	if len(stats.FavoriteGames) != 1 || stats.FavoriteGames[0].ID != secondGameID {
		t.Fatalf("FavoriteGames = %+v, want only second game", stats.FavoriteGames)
	}
	if !stats.FavoriteGames[0].IsFavorite {
		t.Fatalf("favorite[0].IsFavorite = false, want true")
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
	if !stats.PopularGames[0].IsFavorite {
		t.Fatalf("popular[0].IsFavorite = false, want true")
	}
	if stats.PopularGames[0].SeriesID == nil || *stats.PopularGames[0].SeriesID != seriesID {
		t.Fatalf("popular[0].SeriesID = %v, want %d", stats.PopularGames[0].SeriesID, seriesID)
	}
	if stats.PopularGames[0].SeriesName == nil || *stats.PopularGames[0].SeriesName != "Stats Series" {
		t.Fatalf("popular[0].SeriesName = %v, want Stats Series", stats.PopularGames[0].SeriesName)
	}
}

func TestGamesRepositoryStatsIncludesPrivateFavoritesForAdmin(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	publicGameID := insertRepositoryGame(t, db, "stats-admin-public", "Stats Admin Public", "public")
	privateGameID := insertRepositoryGame(t, db, "stats-admin-private", "Stats Admin Private", "private")

	if _, err := db.Exec(`INSERT INTO favorite_games (game_id) VALUES (?)`, publicGameID); err != nil {
		t.Fatalf("insert public favorite game: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO favorite_games (game_id) VALUES (?)`, privateGameID); err != nil {
		t.Fatalf("insert private favorite game: %v", err)
	}

	catalogRepo := NewGameCatalogRepository(repo, NewFavoriteGamesRepository(db))

	stats, err := catalogRepo.Stats(domain.GamesListParams{IncludeAll: true})
	if err != nil {
		t.Fatalf("Stats returned error: %v", err)
	}

	if stats.FavoriteCount != 2 {
		t.Fatalf("FavoriteCount = %d, want 2 favorites for admin scope", stats.FavoriteCount)
	}
	if len(stats.FavoriteGames) != 2 {
		t.Fatalf("FavoriteGames = %d, want 2 favorites for admin scope", len(stats.FavoriteGames))
	}
}

func TestGameCatalogRepositoryListFiltersFavoritesAndExposesFavoriteState(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	catalogRepo := NewGameCatalogRepository(repo, NewFavoriteGamesRepository(db))
	favoriteID := insertRepositoryGame(t, db, "favorite-a", "Favorite A", "public")
	otherID := insertRepositoryGame(t, db, "favorite-b", "Favorite B", "public")
	privateFavoriteID := insertRepositoryGame(t, db, "favorite-private", "Favorite Private", "private")

	if _, err := db.Exec(`INSERT INTO favorite_games (game_id) VALUES (?)`, favoriteID); err != nil {
		t.Fatalf("insert public favorite: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO favorite_games (game_id) VALUES (?)`, privateFavoriteID); err != nil {
		t.Fatalf("insert private favorite: %v", err)
	}
	seriesID := insertRepositorySeries(t, db, "Favorite Series", "favorite-series")
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, seriesID, favoriteID); err != nil {
		t.Fatalf("set favorite series: %v", err)
	}

	games, total, err := catalogRepo.List(domain.GamesListParams{
		Page:         1,
		Limit:        10,
		FavoriteOnly: true,
		Sort:         "updated_at",
		Order:        "desc",
	})
	if err != nil {
		t.Fatalf("List favorite-only returned error: %v", err)
	}

	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(games) != 1 || games[0].ID != favoriteID {
		t.Fatalf("games = %+v, want only public favorite game", games)
	}
	if !games[0].IsFavorite {
		t.Fatalf("games[0].IsFavorite = false, want true")
	}
	if games[0].SeriesID == nil || *games[0].SeriesID != seriesID {
		t.Fatalf("games[0].SeriesID = %v, want %d", games[0].SeriesID, seriesID)
	}
	if games[0].SeriesName == nil || *games[0].SeriesName != "Favorite Series" {
		t.Fatalf("games[0].SeriesName = %v, want Favorite Series", games[0].SeriesName)
	}

	allGames, allTotal, err := catalogRepo.List(domain.GamesListParams{
		Page:         1,
		Limit:        10,
		IncludeAll:   true,
		FavoriteOnly: true,
		Sort:         "updated_at",
		Order:        "desc",
	})
	if err != nil {
		t.Fatalf("List favorite-only includeAll returned error: %v", err)
	}

	if allTotal != 2 {
		t.Fatalf("includeAll total = %d, want 2", allTotal)
	}
	if len(allGames) != 2 {
		t.Fatalf("len(includeAll games) = %d, want 2", len(allGames))
	}
	if allGames[0].ID != privateFavoriteID && allGames[1].ID != privateFavoriteID {
		t.Fatalf("includeAll games = %+v, want private favorite included", allGames)
	}
	for _, game := range allGames {
		if !game.IsFavorite {
			t.Fatalf("includeAll game %+v has IsFavorite=false, want true", game)
		}
		if game.ID == otherID {
			t.Fatalf("unexpected non-favorite game in results: %+v", game)
		}
	}
}

func TestGameCatalogRepositoryListPendingOnlyFiltersResolvedAndIgnoredIssues(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	catalogRepo := NewGameCatalogRepository(repo, NewFavoriteGamesRepository(db))

	visiblePendingID := insertRepositoryGame(t, db, "pending-visible", "Pending Visible", "public")
	resolvedID := insertRepositoryGame(t, db, "pending-resolved", "Pending Resolved", "public")
	ignoredID := insertRepositoryGame(t, db, "pending-ignored", "Pending Ignored", "public")
	privatePendingID := insertRepositoryGame(t, db, "pending-private", "Pending Private", "private")

	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?, banner_image = ?, summary = ?, wiki_content = ?
		WHERE id = ?
	`, "/assets/cover.png", "/assets/banner.png", "Ready", "# Ready", resolvedID); err != nil {
		t.Fatalf("seed resolved repository pending game: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE games
		SET banner_image = ?, summary = ?, wiki_content = ?
		WHERE id = ?
	`, "/assets/banner.png", "Ready", "# Ready", ignoredID); err != nil {
		t.Fatalf("seed ignored repository pending game: %v", err)
	}

	insertRepositoryAsset(t, db, resolvedID, "resolved-shot", "screenshot", "/assets/resolved/shot.png", 0)
	insertRepositoryAsset(t, db, resolvedID, "resolved-logo", "logo", "/assets/resolved/logo.png", 0)
	insertRepositoryAsset(t, db, resolvedID, "resolved-video", "video", "/assets/resolved/trailer.mp4", 0)
	insertRepositoryGameFile(t, db, resolvedID, "/roms/resolved.rom")
	insertRepositoryAsset(t, db, ignoredID, "ignored-shot", "screenshot", "/assets/ignored/shot.png", 0)
	insertRepositoryAsset(t, db, ignoredID, "ignored-logo", "logo", "/assets/ignored/logo.png", 0)
	insertRepositoryAsset(t, db, ignoredID, "ignored-video", "video", "/assets/ignored/trailer.mp4", 0)
	insertRepositoryGameFile(t, db, ignoredID, "/roms/ignored.rom")

	developerID := insertRepositoryDeveloper(t, db, "Pending Developer", "pending-developer")
	publisherID := insertRepositoryPublisher(t, db, "Pending Publisher", "pending-publisher")
	linkRepositoryGameDeveloper(t, db, resolvedID, developerID, 0)
	linkRepositoryGamePublisher(t, db, resolvedID, publisherID, 0)
	linkRepositoryGameDeveloper(t, db, ignoredID, developerID, 0)
	linkRepositoryGamePublisher(t, db, ignoredID, publisherID, 0)

	if _, err := db.Exec(`
		INSERT INTO game_review_issue_overrides (game_id, issue_key, status)
		VALUES (?, 'missing-cover', 'ignored')
	`, ignoredID); err != nil {
		t.Fatalf("insert ignored pending override: %v", err)
	}

	games, total, err := catalogRepo.List(domain.GamesListParams{
		Page:        1,
		Limit:       10,
		PendingOnly: true,
		Sort:        "updated_at",
		Order:       "desc",
	})
	if err != nil {
		t.Fatalf("List pending-only returned error: %v", err)
	}

	if total != 1 {
		t.Fatalf("total = %d, want 1", total)
	}
	if len(games) != 1 || games[0].ID != visiblePendingID {
		t.Fatalf("games = %+v, want only visible pending public game", games)
	}

	includeAllGames, includeAllTotal, err := catalogRepo.List(domain.GamesListParams{
		Page:        1,
		Limit:       10,
		PendingOnly: true,
		IncludeAll:  true,
		Sort:        "updated_at",
		Order:       "desc",
	})
	if err != nil {
		t.Fatalf("List pending-only includeAll returned error: %v", err)
	}

	if includeAllTotal != 2 {
		t.Fatalf("includeAll total = %d, want 2", includeAllTotal)
	}
	if len(includeAllGames) != 2 {
		t.Fatalf("len(includeAllGames) = %d, want 2", len(includeAllGames))
	}

	gotIDs := []int64{includeAllGames[0].ID, includeAllGames[1].ID}
	if !(containsRepositoryGameID(gotIDs, visiblePendingID) && containsRepositoryGameID(gotIDs, privatePendingID)) {
		t.Fatalf("includeAll games = %+v, want visible and private pending games", includeAllGames)
	}

	ignoredGames, ignoredTotal, err := catalogRepo.List(domain.GamesListParams{
		Page:                  1,
		Limit:                 10,
		PendingOnly:           true,
		PendingIncludeIgnored: true,
		Sort:                  "updated_at",
		Order:                 "desc",
	})
	if err != nil {
		t.Fatalf("List pending-only includeIgnored returned error: %v", err)
	}
	if ignoredTotal != 2 {
		t.Fatalf("includeIgnored total = %d, want 2", ignoredTotal)
	}
	if len(ignoredGames) != 2 {
		t.Fatalf("len(ignoredGames) = %d, want 2", len(ignoredGames))
	}
	if !containsRepositoryGameID([]int64{ignoredGames[0].ID, ignoredGames[1].ID}, ignoredID) {
		t.Fatalf("ignoredGames = %+v, want ignored-only game included", ignoredGames)
	}
}

func TestGameCatalogRepositoryListPendingOnlySupportsNativeSortAndFilters(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	catalogRepo := NewGameCatalogRepository(repo, NewFavoriteGamesRepository(db))

	severeID := insertRepositoryGame(t, db, "pending-severe", "Pending Severe", "public")
	recentID := insertRepositoryGame(t, db, "pending-recent", "Pending Recent", "public")
	olderID := insertRepositoryGame(t, db, "pending-older", "Pending Older", "public")

	now := time.Now().UTC()
	updateRepositoryGameStats(t, db, severeID, 50, now.Format("2006-01-02 15:04:05"))
	updateRepositoryGameStats(t, db, recentID, 10, now.AddDate(0, 0, -1).Format("2006-01-02 15:04:05"))
	updateRepositoryGameStats(t, db, olderID, 5, now.AddDate(0, 0, -60).Format("2006-01-02 15:04:05"))

	if _, err := db.Exec(`
		UPDATE games
		SET banner_image = ?, summary = ?
		WHERE id = ?
	`, "/assets/recent-banner.png", "Ready", recentID); err != nil {
		t.Fatalf("seed recent pending game: %v", err)
	}

	games, total, err := catalogRepo.List(domain.GamesListParams{
		Page:              1,
		Limit:             10,
		PendingOnly:       true,
		PendingSevereOnly: true,
		PendingRecentDays: 30,
		PendingIssue:      "missing-assets",
		Sort:              "pending_issue_count",
		Order:             "desc",
	})
	if err != nil {
		t.Fatalf("List pending-only native filters returned error: %v", err)
	}

	if total != 2 {
		t.Fatalf("total = %d, want 2", total)
	}
	if len(games) != 2 {
		t.Fatalf("len(games) = %d, want 2", len(games))
	}
	if games[0].ID != severeID || games[1].ID != recentID {
		t.Fatalf("games = %+v, want severe game ordered before recent game", games)
	}
}

func TestGameCatalogRepositoryCountPendingGroupsUsesQueueFiltersButIgnoresIssueSelector(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	catalogRepo := NewGameCatalogRepository(repo, NewFavoriteGamesRepository(db))

	_ = insertRepositoryGame(t, db, "pending-asset", "Pending Asset", "public")
	wikiID := insertRepositoryGame(t, db, "pending-wiki", "Pending Wiki", "public")

	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?, banner_image = ?, summary = ?, wiki_content = ?
		WHERE id = ?
	`, "/assets/wiki-cover.png", "/assets/wiki-banner.png", "Ready", "# Ready", wikiID); err != nil {
		t.Fatalf("seed wiki game: %v", err)
	}
	insertRepositoryAsset(t, db, wikiID, "wiki-shot", "screenshot", "/assets/wiki/shot.png", 0)
	insertRepositoryAsset(t, db, wikiID, "wiki-logo", "logo", "/assets/wiki/logo.png", 0)
	insertRepositoryAsset(t, db, wikiID, "wiki-video", "video", "/assets/wiki/trailer.mp4", 0)
	insertRepositoryGameFile(t, db, wikiID, "/roms/wiki.rom")
	developerID := insertRepositoryDeveloper(t, db, "Wiki Developer", "wiki-developer")
	publisherID := insertRepositoryPublisher(t, db, "Wiki Publisher", "wiki-publisher")
	linkRepositoryGameDeveloper(t, db, wikiID, developerID, 0)
	linkRepositoryGamePublisher(t, db, wikiID, publisherID, 0)

	counts, err := catalogRepo.CountPendingGroups(domain.GamesListParams{
		Page:         1,
		Limit:        10,
		PendingOnly:  true,
		PendingIssue: "missing-assets",
		Search:       "Pending",
	})
	if err != nil {
		t.Fatalf("CountPendingGroups returned error: %v", err)
	}

	if counts.MissingAssets != 1 || counts.MissingWiki != 1 || counts.MissingFiles != 1 || counts.MissingMetadata != 1 {
		t.Fatalf("counts = %+v, want one matching game contributing to all visible pending groups", counts)
	}
	if counts.IgnoredTotal != 0 {
		t.Fatalf("counts.ignored_total = %d, want 0", counts.IgnoredTotal)
	}
}

func TestGamesRepositoryListTimelineAppliesVisibilityAndCursor(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	newestID := insertRepositoryGameWithReleaseDate(t, db, "timeline-newest", "Timeline Newest", "public", "2024-06-01")
	middleID := insertRepositoryGameWithReleaseDate(t, db, "timeline-middle", "Timeline Middle", "public", "2024-05-01")
	_ = insertRepositoryGameWithReleaseDate(t, db, "timeline-private", "Timeline Private", "private", "2024-04-15")
	oldestID := insertRepositoryGameWithReleaseDate(t, db, "timeline-oldest", "Timeline Oldest", "public", "2024-04-01")

	// First page: limit=2, should get newest and middle (private excluded).
	games, err := repo.ListTimeline(domain.GamesTimelineParams{
		Limit:      2,
		Visibility: domain.GameVisibilityPublic,
	})
	if err != nil {
		t.Fatalf("ListTimeline first page returned error: %v", err)
	}
	if len(games) != 3 {
		// Repo fetches limit+1=3 rows for hasMore detection.
		t.Fatalf("len(games) = %d, want 3 (limit+1)", len(games))
	}
	if games[0].ID != newestID || games[1].ID != middleID {
		t.Fatalf("first two games = %+v, want newest then middle", games[:2])
	}

	// Second page: cursor after middle.
	games, err = repo.ListTimeline(domain.GamesTimelineParams{
		Limit:      2,
		AfterDate:  "2024-05-01",
		AfterID:    middleID,
		Visibility: domain.GameVisibilityPublic,
	})
	if err != nil {
		t.Fatalf("ListTimeline second page returned error: %v", err)
	}
	if len(games) != 1 {
		t.Fatalf("len(games) = %d, want 1 (only oldest)", len(games))
	}
	if games[0].ID != oldestID {
		t.Fatalf("games[0] = %+v, want oldest", games[0])
	}
}

func TestGamesRepositoryUpdateAggregateReplaceCoverPreservesNewPath(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	gameID := insertRepositoryGame(t, db, "replace-cover", "Replace Cover", "public")

	// Seed an existing cover in game_assets and sync to games.cover_image.
	insertRepositoryAsset(t, db, gameID, "old-cover", "cover", "/assets/replace-cover/old-cover.jpg", 0)

	// Simulate the frontend "replace cover" flow: add a new cover asset, delete the old one.
	insertRepositoryAsset(t, db, gameID, "new-cover", "cover", "/assets/replace-cover/new-cover.jpg", 1)
	if _, err := repo.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{
				Title:      "Replace Cover",
				Visibility: "public",
			},
		},
		Assets: domain.GameAggregateAssetsInput{
			CoverOrderAssetUIDs: []string{"new-cover"},
		},
	}); err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}

	// The new cover path must be preserved via syncPrimaryCoverTx.
	game, err := repo.GetByID(gameID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if game.CoverImage == nil || *game.CoverImage != "/assets/replace-cover/new-cover.jpg" {
		t.Fatalf("cover_image = %v, want /assets/replace-cover/new-cover.jpg", game.CoverImage)
	}
}

func TestGamesRepositoryUpdateAggregateRemoveCoverSetsNull(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	gameID := insertRepositoryGame(t, db, "remove-cover", "Remove Cover", "public")

	// Seed an existing cover in game_assets and sync to games.cover_image.
	insertRepositoryAsset(t, db, gameID, "old-cover", "cover", "/assets/remove-cover/old-cover.jpg", 0)

	// Simulate the frontend "remove cover" flow: delete the only cover asset.
	if _, err := repo.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{
				Title:      "Remove Cover",
				Visibility: "public",
			},
		},
		Assets: domain.GameAggregateAssetsInput{},
	}); err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}

	game, err := repo.GetByID(gameID)
	if err != nil {
		t.Fatalf("GetByID returned error: %v", err)
	}
	if game.CoverImage != nil {
		t.Fatalf("cover_image = %v, want nil", game.CoverImage)
	}
}

func strPtr(s string) *string {
	return &s
}

func TestGamesRepositoryUpdateAggregatePersistsLogoPositionsForMultipleLogos(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	gameID := insertRepositoryGame(t, db, "repo-logo-positions", "Repo Logo Positions", "public")
	insertRepositoryAsset(t, db, gameID, "logo-a", "logo", "/assets/repo-logo-positions/logo-a.png", 0)
	insertRepositoryAsset(t, db, gameID, "logo-b", "logo", "/assets/repo-logo-positions/logo-b.png", 1)

	posX, posY, width := 21.0, 50.0, 10.0
	otherX, otherY, otherWidth := 11.0, 22.0, 33.0
	if _, err := repo.UpdateAggregate(gameID, domain.GameAggregateUpdateInput{
		Game: domain.GameAggregateCoreUpdateInput{
			GameCoreInput: domain.GameCoreInput{Title: "Repo Logo Positions", Visibility: "public"},
		},
		Assets: domain.GameAggregateAssetsInput{
			LogoOrderAssetUIDs: []string{"logo-a", "logo-b"},
			LogoPositions: []domain.LogoPositionInput{
				{AssetUID: "logo-a", PositionX: &posX, PositionY: &posY, WidthPct: &width},
				{AssetUID: "logo-b", PositionX: &otherX, PositionY: &otherY, WidthPct: &otherWidth},
			},
		},
	}); err != nil {
		t.Fatalf("UpdateAggregate returned error: %v", err)
	}

	var rows []struct {
		AssetUID  string   `db:"asset_uid"`
		PositionX *float64 `db:"position_x"`
		PositionY *float64 `db:"position_y"`
		WidthPct  *float64 `db:"width_pct"`
	}
	if err := db.Select(&rows, `
		SELECT asset_uid, position_x, position_y, width_pct
		FROM game_assets
		WHERE game_id = ? AND asset_type = 'logo'
		ORDER BY asset_uid
	`, gameID); err != nil {
		t.Fatalf("select logo positions: %v", err)
	}

	assertLogoPositionRow(t, rows, "logo-a", posX, posY, width)
	assertLogoPositionRow(t, rows, "logo-b", otherX, otherY, otherWidth)
}

func assertLogoPositionRow(t *testing.T, rows []struct {
	AssetUID  string   `db:"asset_uid"`
	PositionX *float64 `db:"position_x"`
	PositionY *float64 `db:"position_y"`
	WidthPct  *float64 `db:"width_pct"`
}, assetUID string, wantX, wantY, wantW float64) {
	t.Helper()
	for _, row := range rows {
		if row.AssetUID != assetUID {
			continue
		}
		if row.PositionX == nil || *row.PositionX != wantX {
			t.Fatalf("%s position_x = %v, want %v", assetUID, row.PositionX, wantX)
		}
		if row.PositionY == nil || *row.PositionY != wantY {
			t.Fatalf("%s position_y = %v, want %v", assetUID, row.PositionY, wantY)
		}
		if row.WidthPct == nil || *row.WidthPct != wantW {
			t.Fatalf("%s width_pct = %v, want %v", assetUID, row.WidthPct, wantW)
		}
		return
	}
	t.Fatalf("logo asset %s not found in %#v", assetUID, rows)
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

func updateRepositoryGameStats(t *testing.T, db *sqlx.DB, gameID int64, downloads int64, createdAt string) {
	t.Helper()

	if _, err := db.Exec(`
		UPDATE games
		SET downloads = ?, created_at = ?, updated_at = ?
		WHERE id = ?
	`, downloads, createdAt, createdAt, gameID); err != nil {
		t.Fatalf("update repository game stats: %v", err)
	}
}

func updateRepositoryPrivateGameStats(t *testing.T, db *sqlx.DB, publicID string, downloads int64, createdAt string) {
	t.Helper()

	if _, err := db.Exec(`
		UPDATE games
		SET downloads = ?, created_at = ?, updated_at = ?
		WHERE public_id = ?
	`, downloads, createdAt, createdAt, publicID); err != nil {
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

func containsRepositoryGameID(ids []int64, want int64) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

func TestGamesRepositoryListVideosByPublicIDs(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)

	withVideoID := insertRepositoryGame(t, db, "with-video", "With Video", "public")
	withoutVideoID := insertRepositoryGame(t, db, "without-video", "Without Video", "public")
	privateWithVideoID := insertRepositoryGame(t, db, "private-video", "Private Video", "private")
	_ = withoutVideoID

	insertRepositoryAsset(t, db, withVideoID, "v1", "video", "/assets/with-video/v1.mp4", 1)
	insertRepositoryAsset(t, db, withVideoID, "v0", "video", "/assets/with-video/v0.mp4", 0)
	insertRepositoryAsset(t, db, withVideoID, "cover", "cover", "/assets/with-video/cover.jpg", 0)
	insertRepositoryAsset(t, db, privateWithVideoID, "pv", "video", "/assets/private-video/pv.mp4", 0)

	rows, err := repo.ListVideosByPublicIDs([]string{"with-video", "without-video", "private-video", "missing"})
	if err != nil {
		t.Fatalf("ListVideosByPublicIDs returned error: %v", err)
	}

	// with-video has two video assets sorted by sort_order; without-video and missing have none.
	if len(rows) != 3 {
		t.Fatalf("rows = %d, want 3 (2 videos for with-video + 1 for private-video)", len(rows))
	}

	// Path already starts with "/", so the key is a plain concatenation.
	byID := map[string]videoAssetRow{}
	for _, row := range rows {
		byID[row.PublicID+row.Path] = row
	}
	first := byID["with-video/assets/with-video/v0.mp4"]
	second := byID["with-video/assets/with-video/v1.mp4"]
	if first.PublicID != "with-video" || second.PublicID != "with-video" {
		t.Fatalf("expected with-video rows, got %q and %q", first.PublicID, second.PublicID)
	}
	if first.SortOrder != 0 || second.SortOrder != 1 {
		t.Fatalf("sort_order = %d and %d, want 0 and 1", first.SortOrder, second.SortOrder)
	}
	private := byID["private-video/assets/private-video/pv.mp4"]
	if private.Visibility != "private" {
		t.Fatalf("private row visibility = %q, want private", private.Visibility)
	}
}

func TestGamesRepositoryListVideosByPublicIDsEmpty(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	repo := NewGamesRepository(db)
	rows, err := repo.ListVideosByPublicIDs(nil)
	if err != nil {
		t.Fatalf("ListVideosByPublicIDs returned error: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows = %d, want 0", len(rows))
	}
}
