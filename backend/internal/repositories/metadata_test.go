package repositories

import (
	"testing"
)

func TestMetadataRepositoryListSeriesGamesBySeriesIDsInitializesEmptyAndFiltersVisibility(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		INSERT INTO series (name, slug, sort_order)
		VALUES ('Series A', 'series-a', 0), ('Series B', 'series-b', 1)
	`); err != nil {
		t.Fatalf("insert series: %v", err)
	}

	var seriesAID int64
	if err := db.Get(&seriesAID, `SELECT id FROM series WHERE slug = 'series-a'`); err != nil {
		t.Fatalf("load series A id: %v", err)
	}
	var seriesBID int64
	if err := db.Get(&seriesBID, `SELECT id FROM series WHERE slug = 'series-b'`); err != nil {
		t.Fatalf("load series B id: %v", err)
	}

	publicID := insertRepositoryGame(t, db, "series-public", "Series Public", "public")
	privateID := insertRepositoryGame(t, db, "series-private", "Series Private", "private")
	if _, err := db.Exec(`UPDATE games SET series_id = ?, updated_at = '2024-02-02 00:00:00' WHERE id = ?`, seriesAID, publicID); err != nil {
		t.Fatalf("attach public game to series: %v", err)
	}
	if _, err := db.Exec(`UPDATE games SET series_id = ?, updated_at = '2024-02-03 00:00:00' WHERE id = ?`, seriesAID, privateID); err != nil {
		t.Fatalf("attach private game to series: %v", err)
	}
	insertRepositoryAsset(t, db, publicID, "series-shot", "screenshot", "/assets/series-public/cover.png", 0)

	repo := NewMetadataRepository(db)

	publicOnly, err := repo.ListSeriesGamesBySeriesIDs([]int64{seriesAID, seriesBID}, false)
	if err != nil {
		t.Fatalf("ListSeriesGamesBySeriesIDs(false) returned error: %v", err)
	}
	if len(publicOnly[seriesAID]) != 1 || publicOnly[seriesAID][0].ID != publicID {
		t.Fatalf("publicOnly[seriesA] = %+v, want only public game", publicOnly[seriesAID])
	}
	if publicOnly[seriesAID][0].PrimaryScreenshot == nil || *publicOnly[seriesAID][0].PrimaryScreenshot != "/assets/series-public/cover.png" {
		t.Fatalf("publicOnly primary screenshot = %v, want screenshot path", publicOnly[seriesAID][0].PrimaryScreenshot)
	}
	if games, ok := publicOnly[seriesBID]; !ok || len(games) != 0 {
		t.Fatalf("publicOnly[seriesB] = %+v, want initialized empty slice", games)
	}

	includeAll, err := repo.ListSeriesGamesBySeriesIDs([]int64{seriesAID, seriesBID}, true)
	if err != nil {
		t.Fatalf("ListSeriesGamesBySeriesIDs(true) returned error: %v", err)
	}
	if len(includeAll[seriesAID]) != 2 || includeAll[seriesAID][0].ID != privateID || includeAll[seriesAID][1].ID != publicID {
		t.Fatalf("includeAll[seriesA] = %+v, want private then public by updated_at desc", includeAll[seriesAID])
	}
}

func TestMetadataRepositoryDeleteUnusedRemovesOrphansOnly(t *testing.T) {
	db := openRepositoryTagsTestDB(t)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		INSERT INTO series (name, slug, sort_order)
		VALUES ('Keep Series', 'keep-series', 0), ('Drop Series', 'drop-series', 1)
	`); err != nil {
		t.Fatalf("insert series: %v", err)
	}
	var keepSeriesID int64
	if err := db.Get(&keepSeriesID, `SELECT id FROM series WHERE slug = 'keep-series'`); err != nil {
		t.Fatalf("load keep series id: %v", err)
	}
	gameID := insertRepositoryGame(t, db, "metadata-keep", "Metadata Keep", "public")
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, keepSeriesID, gameID); err != nil {
		t.Fatalf("attach keep series to game: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO platforms (name, slug, sort_order)
		VALUES ('Keep Platform', 'keep-platform', 0), ('Drop Platform', 'drop-platform', 1)
	`); err != nil {
		t.Fatalf("insert platforms: %v", err)
	}
	var keepPlatformID int64
	if err := db.Get(&keepPlatformID, `SELECT id FROM platforms WHERE slug = 'keep-platform'`); err != nil {
		t.Fatalf("load keep platform id: %v", err)
	}
	linkRepositoryGamePlatform(t, db, gameID, keepPlatformID, 0)

	repo := NewMetadataRepository(db)
	if err := repo.DeleteUnusedSeries(); err != nil {
		t.Fatalf("DeleteUnusedSeries returned error: %v", err)
	}
	if err := repo.DeleteUnused("platforms", "game_platforms", "platform_id"); err != nil {
		t.Fatalf("DeleteUnused platforms returned error: %v", err)
	}

	var seriesCount int
	if err := db.Get(&seriesCount, `SELECT COUNT(*) FROM series`); err != nil {
		t.Fatalf("count series: %v", err)
	}
	if seriesCount != 1 {
		t.Fatalf("series count = %d, want 1 after removing orphan", seriesCount)
	}

	var platformCount int
	if err := db.Get(&platformCount, `SELECT COUNT(*) FROM platforms`); err != nil {
		t.Fatalf("count platforms: %v", err)
	}
	if platformCount != 1 {
		t.Fatalf("platform count = %d, want 1 after removing orphan", platformCount)
	}
}
