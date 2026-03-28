package repositories

import "testing"

func TestBuildTitleSortKey(t *testing.T) {
	alt := "塞尔达传说"

	if got := buildTitleSortKey(" Zelda ", nil); got == "" {
		t.Fatalf("buildTitleSortKey() returned empty key for title")
	}
	if got := buildTitleSortKey("", &alt); got == "" {
		t.Fatalf("buildTitleSortKey() returned empty key for titleAlt")
	}
	if got := buildTitleSortKey("   ", nil); got != "" {
		t.Fatalf("buildTitleSortKey() = %q, want empty string", got)
	}
}

func TestUniquePositiveIDs(t *testing.T) {
	got := uniquePositiveIDs([]int64{4, -1, 2, 4, 0, 3, 2})
	want := []int64{2, 3, 4}
	if len(got) != len(want) {
		t.Fatalf("len(result) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("result[%d] = %d, want %d", i, got[i], want[i])
		}
	}
}

func TestUniqueNonEmptyStrings(t *testing.T) {
	got := uniqueNonEmptyStrings([]string{"  /a  ", "", "/b", "/a", "  ", "/c", "/b "})
	want := []string{"/a", "/b", "/c"}
	if len(got) != len(want) {
		t.Fatalf("len(result) = %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("result[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
