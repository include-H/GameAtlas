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

func TestFilterMetadataItems(t *testing.T) {
	items := []domain.MetadataItem{
		{ID: 1, Name: "Zelda", GameCount: 2},
		{ID: 2, Name: "Mario", GameCount: 5},
		{ID: 3, Name: "Metroid", GameCount: 3},
	}

	got := filterMetadataItems(items, MetadataListOptions{
		Search: "m",
		Sort:   "popular",
		Limit:  2,
	})

	if len(got) != 2 {
		t.Fatalf("len(filtered) = %d, want 2", len(got))
	}
	if got[0].Name != "Mario" || got[1].Name != "Metroid" {
		t.Fatalf("filtered order = [%s, %s], want [Mario, Metroid]", got[0].Name, got[1].Name)
	}
}

func TestFilterMetadataItemsForSimpleResources(t *testing.T) {
	items := []domain.MetadataItem{
		{ID: 1, Name: "Switch"},
		{ID: 2, Name: "PC-98"},
		{ID: 3, Name: "PC"},
	}

	got := filterMetadataItems(items, MetadataListOptions{
		Search: "pc",
		Limit:  1,
	})

	if len(got) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(got))
	}
	if got[0].Name != "PC" {
		t.Fatalf("filtered[0].Name = %q, want PC", got[0].Name)
	}
}

func TestApplySeriesItemGamesPicksDistinctCoverCandidates(t *testing.T) {
	item := &domain.MetadataItem{}
	cover := "/assets/cover-a.jpg"
	banner := "/assets/banner-b.jpg"
	screenshot := "/assets/screenshot-c.jpg"

	games := []domain.SeriesGameSummary{
		{UpdatedAt: "2026-03-25T00:00:00Z", CoverImage: &cover},
		{UpdatedAt: "2026-03-24T00:00:00Z", CoverImage: &cover},
		{UpdatedAt: "2026-03-23T00:00:00Z", BannerImage: &banner},
		{UpdatedAt: "2026-03-22T00:00:00Z", PrimaryScreenshot: &screenshot},
	}

	applySeriesItemGames(item, games)

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

func TestPickSeriesCoverSourceFallsBackInOrder(t *testing.T) {
	cover := " cover "
	banner := " banner "
	screenshot := " screenshot "

	if got := pickSeriesCoverSource(domain.SeriesGameSummary{CoverImage: &cover, BannerImage: &banner, PrimaryScreenshot: &screenshot}); got != "cover" {
		t.Fatalf("pickSeriesCoverSource() = %q, want cover", got)
	}
	if got := pickSeriesCoverSource(domain.SeriesGameSummary{BannerImage: &banner, PrimaryScreenshot: &screenshot}); got != "banner" {
		t.Fatalf("pickSeriesCoverSource() = %q, want banner", got)
	}
	if got := pickSeriesCoverSource(domain.SeriesGameSummary{PrimaryScreenshot: &screenshot}); got != "screenshot" {
		t.Fatalf("pickSeriesCoverSource() = %q, want screenshot", got)
	}
}

func TestMetadataServiceListSeriesReturnsEnrichmentErrorInsteadOfSilentEmptyState(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	if _, err := db.Exec(`INSERT INTO series (name, slug, sort_order) VALUES ('Broken Series', 'broken-series', 0)`); err != nil {
		t.Fatalf("insert series: %v", err)
	}
	if _, err := db.Exec(`DROP TABLE games`); err != nil {
		t.Fatalf("drop games table: %v", err)
	}

	service := NewMetadataService(repositories.NewMetadataRepository(db))
	_, err := service.List(MetadataResource{Type: domain.MetadataSeries}, true, MetadataListOptions{})
	if err == nil {
		t.Fatal("List returned nil error, want enrichment failure")
	}
	if !strings.Contains(err.Error(), "list series games by ids") {
		t.Fatalf("List error = %v, want series enrichment context", err)
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

func TestMetadataServiceCleanupUnusedGameMetadataInvalidatesSeriesListCache(t *testing.T) {
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
	cached, err := service.List(MetadataResource{Type: domain.MetadataSeries}, true, MetadataListOptions{Sort: "name"})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(cached) != 2 {
		t.Fatalf("cached series = %+v, want both admin-visible rows before cleanup", cached)
	}

	if err := service.CleanupUnusedGameMetadata(); err != nil {
		t.Fatalf("CleanupUnusedGameMetadata returned error: %v", err)
	}

	refreshed, err := service.List(MetadataResource{Type: domain.MetadataSeries}, true, MetadataListOptions{Sort: "name"})
	if err != nil {
		t.Fatalf("List after cleanup returned error: %v", err)
	}
	if len(refreshed) != 1 || refreshed[0].ID != keepSeriesID {
		t.Fatalf("refreshed series = %+v, want only keep series after cache invalidation", refreshed)
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
	_, err = service.GetSeriesDetail(seriesID, true)
	if err == nil {
		t.Fatal("GetSeriesDetail returned nil error, want enrichment failure")
	}
	if !strings.Contains(err.Error(), "list series games") {
		t.Fatalf("GetSeriesDetail error = %v, want series enrichment context", err)
	}
}
