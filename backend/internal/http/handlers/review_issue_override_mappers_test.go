package handlers

import (
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestToReviewIssueOverrideResponseMapsDomainShapeToTransportDTO(t *testing.T) {
	reason := "accepted"
	response := toReviewIssueOverrideResponse(domain.ReviewIssueOverride{
		ID:        8,
		GameID:    3,
		IssueKey:  "missing-cover",
		Status:    "ignored",
		Reason:    &reason,
		CreatedAt: "2026-04-04T00:00:00Z",
		UpdatedAt: "2026-04-04T01:00:00Z",
	})

	if response.ID != 8 || response.GameID != 3 || response.IssueKey != "missing-cover" {
		t.Fatalf("response = %+v, want mapped override fields", response)
	}
	if response.Reason == nil || *response.Reason != reason {
		t.Fatalf("reason = %v, want %q", response.Reason, reason)
	}
}
