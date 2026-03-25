package markdown

import (
	"strings"
	"testing"
)

func TestRendererRenderSupportsBasicMarkdownFeatures(t *testing.T) {
	renderer := NewRenderer()

	html, err := renderer.Render("# Title\n\n- [x] done\n- [ ] todo\n\n|A|B|\n|-|-|\n|1|2|")
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	needles := []string{
		"<h1>Title</h1>",
		`<input checked="" disabled="" type="checkbox">`,
		`<input disabled="" type="checkbox">`,
		"<table>",
	}
	for _, needle := range needles {
		if !strings.Contains(html, needle) {
			t.Fatalf("rendered html does not contain %q: %s", needle, html)
		}
	}
}
