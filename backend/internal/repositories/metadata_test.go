package repositories

import (
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestMetadataRepositoryListSeriesGamesBySeriesIDsInitializesEmptyAndFiltersVisibility(t *testing.T) {
	db := openRepositoryTestDB(t)
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

func TestMetadataRepositoryListPublisherGamesByIDsInitializesGroupsAndFiltersVisibility(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	publisherAID := insertRepositoryPublisher(t, db, "Publisher A", "publisher-a")
	publisherBID := insertRepositoryPublisher(t, db, "Publisher B", "publisher-b")
	publicID := insertRepositoryGame(t, db, "publisher-public", "Publisher Public", "public")
	privateID := insertRepositoryGame(t, db, "publisher-private", "Publisher Private", "private")
	otherID := insertRepositoryGame(t, db, "publisher-other", "Publisher Other", "public")
	linkRepositoryGamePublisher(t, db, publicID, publisherAID, 0)
	linkRepositoryGamePublisher(t, db, privateID, publisherAID, 1)
	linkRepositoryGamePublisher(t, db, otherID, publisherBID, 0)

	repo := NewMetadataRepository(db)

	publicOnly, err := repo.ListMetadataGamesByIDs(domain.MetadataPublishers, []int64{publisherAID, publisherBID}, false)
	if err != nil {
		t.Fatalf("ListMetadataGamesByIDs(false) returned error: %v", err)
	}
	if len(publicOnly[publisherAID]) != 1 || publicOnly[publisherAID][0].ID != publicID {
		t.Fatalf("publicOnly[publisherA] = %+v, want only public game", publicOnly[publisherAID])
	}
	if len(publicOnly[publisherBID]) != 1 || publicOnly[publisherBID][0].ID != otherID {
		t.Fatalf("publicOnly[publisherB] = %+v, want other public game", publicOnly[publisherBID])
	}

	includeAll, err := repo.ListMetadataGames(domain.MetadataPublishers, publisherAID, true)
	if err != nil {
		t.Fatalf("ListMetadataGames(true) returned error: %v", err)
	}
	if len(includeAll) != 2 {
		t.Fatalf("includeAll publisher games = %+v, want public and private games", includeAll)
	}
}

func TestMetadataRepositoryFindSimpleBySlugReturnsExistingItem(t *testing.T) {
	db := openRepositoryTestDB(t)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		INSERT INTO developers (name, slug, sort_order)
		VALUES ('IO Interactive A/S', 'io-interactive-a-s', 0)
	`); err != nil {
		t.Fatalf("insert developer: %v", err)
	}

	repo := NewMetadataRepository(db)

	found, err := repo.FindSimpleBySlug(domain.MetadataDevelopers, "io-interactive-a-s")
	if err != nil {
		t.Fatalf("FindSimpleBySlug returned error: %v", err)
	}
	if found == nil {
		t.Fatal("FindSimpleBySlug returned nil, want existing item")
	}
	if found.Name != "IO Interactive A/S" {
		t.Fatalf("FindSimpleBySlug name = %q, want %q", found.Name, "IO Interactive A/S")
	}

	notFound, err := repo.FindSimpleBySlug(domain.MetadataDevelopers, "nonexistent-slug")
	if err != nil {
		t.Fatalf("FindSimpleBySlug nonexistent returned error: %v", err)
	}
	if notFound != nil {
		t.Fatalf("FindSimpleBySlug nonexistent returned %+v, want nil", notFound)
	}
}

func TestMetadataRepositoryDeleteUnusedRemovesOrphansOnly(t *testing.T) {
	db := openRepositoryTestDB(t)
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
		INSERT INTO developers (name, slug, sort_order)
		VALUES ('Keep Developer', 'keep-developer', 0), ('Drop Developer', 'drop-developer', 1)
	`); err != nil {
		t.Fatalf("insert developers: %v", err)
	}
	var keepDeveloperID int64
	if err := db.Get(&keepDeveloperID, `SELECT id FROM developers WHERE slug = 'keep-developer'`); err != nil {
		t.Fatalf("load keep developer id: %v", err)
	}
	linkRepositoryGameDeveloper(t, db, gameID, keepDeveloperID, 0)

	repo := NewMetadataRepository(db)
	if err := repo.DeleteUnusedSeries(); err != nil {
		t.Fatalf("DeleteUnusedSeries returned error: %v", err)
	}
	if err := repo.DeleteUnused(domain.MetadataDevelopers); err != nil {
		t.Fatalf("DeleteUnused developers returned error: %v", err)
	}

	var seriesCount int
	if err := db.Get(&seriesCount, `SELECT COUNT(*) FROM series`); err != nil {
		t.Fatalf("count series: %v", err)
	}
	if seriesCount != 1 {
		t.Fatalf("series count = %d, want 1 after removing orphan", seriesCount)
	}

	var developerCount int
	if err := db.Get(&developerCount, `SELECT COUNT(*) FROM developers`); err != nil {
		t.Fatalf("count developers: %v", err)
	}
	if developerCount != 1 {
		t.Fatalf("developer count = %d, want 1 after removing orphan", developerCount)
	}
}
