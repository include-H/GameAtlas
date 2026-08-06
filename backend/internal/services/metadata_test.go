package services

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"

	dbpkg "github.com/hao/game/internal/db"
	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"  Mega Man X  ":     "mega-man-x",
		"动作 Game 2":          "动作-game-2",
		"Already__Slugged":   "already-slugged",
		"***":                "",
		"Io-Interactive A/S": "io-interactive-a-s",
		"IO Interactive A/S": "io-interactive-a-s",
	}

	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Fatalf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestApplyMetadataItemGamesPicksDistinctCoverCandidates(t *testing.T) {
	item := &domain.MetadataItem{}
	cover := "/assets/cover-a.jpg"
	banner := "/assets/banner-b.jpg"
	screenshot := "/assets/screenshot-c.jpg"

	games := []domain.MetadataGameSummary{
		{UpdatedAt: "2026-03-25T00:00:00Z", CoverImage: &cover},
		{UpdatedAt: "2026-03-24T00:00:00Z", CoverImage: &cover},
		{UpdatedAt: "2026-03-23T00:00:00Z", BannerImage: &banner},
		{UpdatedAt: "2026-03-22T00:00:00Z", PrimaryScreenshot: &screenshot},
	}

	applyMetadataItemGames(item, games)

	if item.GameCount != 4 {
		t.Fatalf("GameCount = %d, want 4", item.GameCount)
	}
	if item.LatestUpdatedAt == nil || *item.LatestUpdatedAt != "2026-03-25T00:00:00Z" {
		t.Fatalf("LatestUpdatedAt = %v, want first game's updated_at", item.LatestUpdatedAt)
	}
	want := []string{cover, banner, screenshot}
	if len(item.CoverCandidates) != len(want) {
		t.Fatalf("len(CoverCandidates) = %d, want %d", len(item.CoverCandidates), len(want))
	}
	for i := range want {
		if item.CoverCandidates[i] != want[i] {
			t.Fatalf("CoverCandidates[%d] = %q, want %q", i, item.CoverCandidates[i], want[i])
		}
	}
	if item.CoverImage == nil || *item.CoverImage != cover {
		t.Fatalf("CoverImage = %v, want %q", item.CoverImage, cover)
	}
}

func TestPickMetadataCoverSourceFallsBackInOrder(t *testing.T) {
	cover := " cover "
	banner := " banner "
	screenshot := " screenshot "

	if got := pickMetadataCoverSource(domain.MetadataGameSummary{CoverImage: &cover, BannerImage: &banner, PrimaryScreenshot: &screenshot}); got != "cover" {
		t.Fatalf("pickMetadataCoverSource() = %q, want cover", got)
	}
	if got := pickMetadataCoverSource(domain.MetadataGameSummary{BannerImage: &banner, PrimaryScreenshot: &screenshot}); got != "banner" {
		t.Fatalf("pickMetadataCoverSource() = %q, want banner", got)
	}
	if got := pickMetadataCoverSource(domain.MetadataGameSummary{PrimaryScreenshot: &screenshot}); got != "screenshot" {
		t.Fatalf("pickMetadataCoverSource() = %q, want screenshot", got)
	}
}

func TestMetadataServiceListPublishersEnrichesImagesAndRespectsVisibility(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`
		INSERT INTO publishers (name, slug, sort_order)
		VALUES ('Atlus', 'atlus', 0), ('Unused Publisher', 'unused-publisher', 1)
	`); err != nil {
		t.Fatalf("insert publishers: %v", err)
	}
	var publisherID int64
	if err := db.Get(&publisherID, `SELECT id FROM publishers WHERE slug = 'atlus'`); err != nil {
		t.Fatalf("load publisher id: %v", err)
	}

	publicID := insertServicesTestGame(t, db, "publisher-public", "Publisher Public", domain.GameVisibilityPublic)
	privateID := insertServicesTestGame(t, db, "publisher-private", "Publisher Private", domain.GameVisibilityPrivate)
	if _, err := db.Exec(`
		UPDATE games
		SET cover_image = ?, updated_at = '2026-03-25 00:00:00'
		WHERE id = ?
	`, "/assets/publisher-public/cover.png", publicID); err != nil {
		t.Fatalf("update public publisher game: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO game_publishers (game_id, publisher_id, sort_order)
		VALUES (?, ?, 0), (?, ?, 1)
	`, publicID, publisherID, privateID, publisherID); err != nil {
		t.Fatalf("link publisher games: %v", err)
	}

	service := NewMetadataService(repositories.NewMetadataRepository(db))
	publicPage, err := service.ListPage(
		MetadataResource{Type: domain.MetadataPublishers},
		false,
		MetadataListOptions{Page: 1, Limit: 100, Sort: "name"},
	)
	if err != nil {
		t.Fatalf("ListPage public publishers returned error: %v", err)
	}
	publicItems := publicPage.Items
	if len(publicItems) != 1 || publicItems[0].ID != publisherID {
		t.Fatalf("public publishers = %+v, want Atlus only", publicItems)
	}
	if publicItems[0].GameCount != 1 || publicItems[0].CoverImage == nil || *publicItems[0].CoverImage != "/assets/publisher-public/cover.png" {
		t.Fatalf("public publisher enrichment = %+v, want one game and cover", publicItems[0])
	}

	allPage, err := service.ListPage(
		MetadataResource{Type: domain.MetadataPublishers},
		true,
		MetadataListOptions{Page: 1, Limit: 100, Sort: "name"},
	)
	if err != nil {
		t.Fatalf("ListPage all publishers returned error: %v", err)
	}
	allItems := allPage.Items
	if len(allItems) != 2 || allItems[0].GameCount != 2 {
		t.Fatalf("all publishers = %+v, want Atlus with two games and unused row", allItems)
	}
}

func TestMetadataServiceListPagePaginatesSeriesAndDetailGames(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	insertSeries := func(name string, slug string) int64 {
		t.Helper()
		result, err := db.Exec(`INSERT INTO series (name, slug, sort_order) VALUES (?, ?, 0)`, name, slug)
		if err != nil {
			t.Fatalf("insert series %s: %v", name, err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatalf("LastInsertId for %s: %v", name, err)
		}
		return id
	}

	seriesAID := insertSeries("Alpha", "alpha")
	seriesBID := insertSeries("Beta", "beta")
	insertSeries("Gamma", "gamma")

	gameAOne := insertServicesTestGame(t, db, "alpha-one", "Alpha One", domain.GameVisibilityPublic)
	gameATwo := insertServicesTestGame(t, db, "alpha-two", "Alpha Two", domain.GameVisibilityPublic)
	gameBeta := insertServicesTestGame(t, db, "beta-one", "Beta One", domain.GameVisibilityPublic)
	gamePrivate := insertServicesTestGame(t, db, "alpha-private", "Alpha Private", domain.GameVisibilityPrivate)
	for _, gameID := range []int64{gameAOne, gameATwo, gamePrivate} {
		if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, seriesAID, gameID); err != nil {
			t.Fatalf("attach game to Alpha: %v", err)
		}
	}
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, seriesBID, gameBeta); err != nil {
		t.Fatalf("attach game to Beta: %v", err)
	}

	service := NewMetadataService(repositories.NewMetadataRepository(db))
	firstPage, err := service.ListPage(
		MetadataResource{Type: domain.MetadataSeries},
		false,
		MetadataListOptions{Page: 1, Limit: 2, Sort: "name"},
	)
	if err != nil {
		t.Fatalf("ListPage returned error: %v", err)
	}
	if firstPage.Total != 2 || firstPage.TotalPages != 1 || len(firstPage.Items) != 2 {
		t.Fatalf("ListPage first page = total %d, pages %d, items %d, want 2, 1, 2", firstPage.Total, firstPage.TotalPages, len(firstPage.Items))
	}
	if firstPage.Items[0].Name != "Alpha" || firstPage.Items[0].GameCount != 2 {
		t.Fatalf("ListPage first item = %+v, want Alpha with 2 public games", firstPage.Items[0])
	}

	secondPage, err := service.ListPage(
		MetadataResource{Type: domain.MetadataSeries},
		true,
		MetadataListOptions{Page: 2, Limit: 2, Sort: "name"},
	)
	if err != nil {
		t.Fatalf("ListPage admin second page returned error: %v", err)
	}
	if secondPage.Total != 3 || secondPage.TotalPages != 2 || len(secondPage.Items) != 1 || secondPage.Items[0].Name != "Gamma" {
		t.Fatalf("ListPage admin second page = %+v, want Gamma only", secondPage)
	}

	detail, err := service.GetSeriesDetail(seriesAID, false, MetadataDetailOptions{Page: 1, Limit: 1})
	if err != nil {
		t.Fatalf("GetSeriesDetail returned error: %v", err)
	}
	if detail.Total != 2 || detail.TotalPages != 2 || len(detail.Games) != 1 {
		t.Fatalf("GetSeriesDetail = total %d, pages %d, games %d, want 2, 2, 1", detail.Total, detail.TotalPages, len(detail.Games))
	}
}

func TestMetadataServiceListSeriesReturnsEnrichmentErrorInsteadOfSilentEmptyState(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`INSERT INTO series (name, slug, sort_order) VALUES ('Broken Series', 'broken-series', 0)`); err != nil {
		t.Fatalf("insert series: %v", err)
	}
	if _, err := db.Exec(`DROP TABLE game_assets`); err != nil {
		t.Fatalf("drop game_assets table: %v", err)
	}

	service := NewMetadataService(repositories.NewMetadataRepository(db))
	_, err := service.ListPage(MetadataResource{Type: domain.MetadataSeries}, true, MetadataListOptions{Page: 1, Limit: 24})
	if err == nil {
		t.Fatal("ListPage returned nil error, want enrichment failure")
	}
	if !strings.Contains(err.Error(), "list series games by ids") {
		t.Fatalf("ListPage error = %v, want series enrichment context", err)
	}
}

func TestMetadataServiceCreateDeveloperReturnsExistingBySlugWhenNameDiffers(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	// Insert "IO Interactive A/S" with slug "io-interactive-a-s".
	if _, err := db.Exec(`INSERT INTO developers (name, slug, sort_order) VALUES ('IO Interactive A/S', 'io-interactive-a-s', 0)`); err != nil {
		t.Fatalf("insert developer: %v", err)
	}

	service := NewMetadataService(repositories.NewMetadataRepository(db))

	// Try to create "Io-Interactive A/S" — name differs due to hyphen, but slug collides.
	created, err := service.Create(
		MetadataResource{Type: domain.MetadataDevelopers},
		domain.MetadataWriteInput{Name: "Io-Interactive A/S"},
	)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.Name != "IO Interactive A/S" {
		t.Fatalf("Create name = %q, want existing %q", created.Name, "IO Interactive A/S")
	}
}

func TestMetadataServiceCreateSeriesReturnsExistingBySlugWhenNameDiffers(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`INSERT INTO series (name, slug, sort_order) VALUES ('Far Cry', 'far-cry', 0)`); err != nil {
		t.Fatalf("insert series: %v", err)
	}

	service := NewMetadataService(repositories.NewMetadataRepository(db))

	created, err := service.Create(
		MetadataResource{Type: domain.MetadataSeries},
		domain.MetadataWriteInput{Name: "Far-Cry"},
	)
	if err != nil {
		t.Fatalf("Create returned error: %v", err)
	}
	if created.Name != "Far Cry" {
		t.Fatalf("Create name = %q, want existing %q", created.Name, "Far Cry")
	}

	var count int
	if err := db.Get(&count, `SELECT COUNT(*) FROM series`); err != nil {
		t.Fatalf("count series: %v", err)
	}
	if count != 1 {
		t.Fatalf("series count = %d, want 1", count)
	}
}

func TestMetadataServiceCleanupUnusedGameMetadataRefreshesSeriesList(t *testing.T) {
	db := openServicesTestDB(t)
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
	gameID := insertServicesTestGame(t, db, "metadata-cache-keep", "Metadata Cache Keep", domain.GameVisibilityPublic)
	if _, err := db.Exec(`UPDATE games SET series_id = ? WHERE id = ?`, keepSeriesID, gameID); err != nil {
		t.Fatalf("attach keep series: %v", err)
	}

	service := NewMetadataService(repositories.NewMetadataRepository(db))
	before, err := service.ListPage(MetadataResource{Type: domain.MetadataSeries}, true, MetadataListOptions{Page: 1, Limit: 100, Sort: "name"})
	if err != nil {
		t.Fatalf("ListPage before cleanup returned error: %v", err)
	}
	if len(before.Items) != 2 {
		t.Fatalf("series before cleanup = %+v, want both admin-visible rows", before.Items)
	}

	if err := service.CleanupUnusedGameMetadata(); err != nil {
		t.Fatalf("CleanupUnusedGameMetadata returned error: %v", err)
	}

	refreshed, err := service.ListPage(MetadataResource{Type: domain.MetadataSeries}, true, MetadataListOptions{Page: 1, Limit: 100, Sort: "name"})
	if err != nil {
		t.Fatalf("ListPage after cleanup returned error: %v", err)
	}
	if len(refreshed.Items) != 1 || refreshed.Items[0].ID != keepSeriesID {
		t.Fatalf("refreshed series = %+v, want only keep series after cleanup", refreshed.Items)
	}
}

func TestMetadataServiceCreateIsConcurrentAndIdempotent(t *testing.T) {
	testCases := []struct {
		name     string
		resource MetadataResource
		table    string
	}{
		{
			name:     "series",
			resource: MetadataResource{Type: domain.MetadataSeries},
			table:    "series",
		},
		{
			name:     "developer",
			resource: MetadataResource{Type: domain.MetadataDevelopers},
			table:    "developers",
		},
		{
			name:     "publisher",
			resource: MetadataResource{Type: domain.MetadataPublishers},
			table:    "publishers",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "app.db")
			firstDB, err := dbpkg.OpenSQLite(path)
			if err != nil {
				t.Fatalf("open first database: %v", err)
			}
			defer func() { _ = firstDB.Close() }()
			if err := dbpkg.RunMigrations(firstDB); err != nil {
				t.Fatalf("run migrations: %v", err)
			}

			secondDB, err := dbpkg.OpenSQLite(path)
			if err != nil {
				t.Fatalf("open second database: %v", err)
			}
			defer func() { _ = secondDB.Close() }()

			services := []*MetadataService{
				NewMetadataService(repositories.NewMetadataRepository(firstDB)),
				NewMetadataService(repositories.NewMetadataRepository(secondDB)),
			}
			start := make(chan struct{})
			results := make(chan *domain.MetadataItem, len(services))
			errs := make(chan error, len(services))
			var waitGroup sync.WaitGroup

			for _, service := range services {
				waitGroup.Add(1)
				go func(service *MetadataService) {
					defer waitGroup.Done()
					<-start
					item, createErr := service.Create(testCase.resource, domain.MetadataWriteInput{
						Name: "Concurrent Metadata",
					})
					if createErr != nil {
						errs <- createErr
						return
					}
					results <- item
				}(service)
			}

			close(start)
			waitGroup.Wait()
			close(results)
			close(errs)

			for createErr := range errs {
				t.Fatalf("concurrent Create returned error: %v", createErr)
			}

			var created []*domain.MetadataItem
			for item := range results {
				created = append(created, item)
			}
			if len(created) != 2 {
				t.Fatalf("created result count = %d, want 2", len(created))
			}
			if created[0].ID != created[1].ID {
				t.Fatalf("created ids = %d and %d, want the same id", created[0].ID, created[1].ID)
			}

			var count int
			if err := firstDB.Get(&count, "SELECT COUNT(*) FROM "+testCase.table); err != nil {
				t.Fatalf("count metadata rows: %v", err)
			}
			if count != 1 {
				t.Fatalf("metadata row count = %d, want 1", count)
			}
		})
	}
}

func TestMetadataServiceGetSeriesDetailReturnsEnrichmentErrorInsteadOfSilentEmptyState(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	result, err := db.Exec(`INSERT INTO series (name, slug, sort_order) VALUES ('Broken Detail', 'broken-detail', 0)`)
	if err != nil {
		t.Fatalf("insert series: %v", err)
	}
	seriesID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId: %v", err)
	}
	if _, err := db.Exec(`DROP TABLE games`); err != nil {
		t.Fatalf("drop games table: %v", err)
	}

	service := NewMetadataService(repositories.NewMetadataRepository(db))
	_, err = service.GetSeriesDetail(seriesID, true, MetadataDetailOptions{})
	if err == nil {
		t.Fatal("GetSeriesDetail returned nil error, want enrichment failure")
	}
	if !strings.Contains(err.Error(), "list series games") {
		t.Fatalf("GetSeriesDetail error = %v, want series enrichment context", err)
	}
}
