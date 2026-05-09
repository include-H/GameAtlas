package handlers

import (
	"encoding/json"
	"testing"
)

func TestMetadataWriteRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{"name":" Series ","slug":" custom-slug ","sort_order":7}`)

	var request metadataWriteRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()
	if input.Name != " Series " {
		t.Fatalf("name = %q, want original transport value", input.Name)
	}
	if input.Slug == nil || *input.Slug != " custom-slug " {
		t.Fatalf("slug = %v, want original transport value", input.Slug)
	}
	if input.SortOrder == nil || *input.SortOrder != 7 {
		t.Fatalf("sort_order = %v, want 7", input.SortOrder)
	}
}

func TestWikiWriteRequestToInputPreservesTransportSemantics(t *testing.T) {
	raw := []byte(`{"content":"# Demo","change_summary":"  summary  "}`)

	var request wikiWriteRequest
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatalf("unmarshal request: %v", err)
	}

	input := request.toInput()
	if input.Content != "# Demo" {
		t.Fatalf("content = %q, want original transport value", input.Content)
	}
	if input.ChangeSummary == nil || *input.ChangeSummary != "  summary  " {
		t.Fatalf("change_summary = %v, want original transport value", input.ChangeSummary)
	}
}
