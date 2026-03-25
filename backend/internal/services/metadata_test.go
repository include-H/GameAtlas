package services

import (
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"  Mega Man X  ":    "mega-man-x",
		"动作 Game 2":         "动作-game-2",
		"Already__Slugged":  "already-slugged",
		"***":               "",
	}

	for input, want := range cases {
		if got := slugify(input); got != want {
			t.Fatalf("slugify(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestFilterSeriesItems(t *testing.T) {
	items := []domain.MetadataItem{
		{ID: 1, Name: "Zelda", GameCount: 2},
		{ID: 2, Name: "Mario", GameCount: 5},
		{ID: 3, Name: "Metroid", GameCount: 3},
	}

	got := filterSeriesItems(items, MetadataListOptions{
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

func TestApplySeriesItemGamesPicksDistinctCoverCandidates(t *testing.T) {
	item := &domain.MetadataItem{}
	cover := "/assets/cover-a.jpg"
	banner := "/assets/banner-b.jpg"
	screenshot := "/assets/screenshot-c.jpg"

	games := []domain.Game{
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

	if got := pickSeriesCoverSource(domain.Game{CoverImage: &cover, BannerImage: &banner, PrimaryScreenshot: &screenshot}); got != "cover" {
		t.Fatalf("pickSeriesCoverSource() = %q, want cover", got)
	}
	if got := pickSeriesCoverSource(domain.Game{BannerImage: &banner, PrimaryScreenshot: &screenshot}); got != "banner" {
		t.Fatalf("pickSeriesCoverSource() = %q, want banner", got)
	}
	if got := pickSeriesCoverSource(domain.Game{PrimaryScreenshot: &screenshot}); got != "screenshot" {
		t.Fatalf("pickSeriesCoverSource() = %q, want screenshot", got)
	}
}
