package services

import (
	"strings"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"  Mega Man X  ":   "mega-man-x",
		"动作 Game 2":        "动作-game-2",
		"Already__Slugged": "already-slugged",
		"***":              "",
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
	_, err := service.List(MetadataResource{Table: "series", ResourceName: "series"}, true, MetadataListOptions{})
	if err == nil {
		t.Fatal("List returned nil error, want enrichment failure")
	}
	if !strings.Contains(err.Error(), "list series games by ids") {
		t.Fatalf("List error = %v, want series enrichment context", err)
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
