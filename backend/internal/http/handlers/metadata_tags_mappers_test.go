package handlers

import (
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestToMetadataResponseKeepsTransportFieldsOutOfDomain(t *testing.T) {
	coverImage := "/assets/series-a/cover.png"
	updatedAt := "2026-04-04T00:00:00Z"

	response := toMetadataResponse(domain.MetadataItem{
		ID:              7,
		Name:            "Series A",
		Slug:            "series-a",
		SortOrder:       3,
		CreatedAt:       "2026-04-01T00:00:00Z",
		GameCount:       2,
		CoverImage:      &coverImage,
		CoverCandidates: []string{coverImage},
		LatestUpdatedAt: &updatedAt,
	})

	if response.ID != 7 || response.Name != "Series A" || response.GameCount != 2 {
		t.Fatalf("response = %+v, want mapped metadata fields", response)
	}
	if response.CoverImage == nil || *response.CoverImage != coverImage {
		t.Fatalf("cover_image = %v, want %q", response.CoverImage, coverImage)
	}
	if response.LatestUpdatedAt == nil || *response.LatestUpdatedAt != updatedAt {
		t.Fatalf("latest_updated_at = %v, want %q", response.LatestUpdatedAt, updatedAt)
	}
}

func TestToTagResponsesExposeOnlyTagResponseShape(t *testing.T) {
	parentID := int64(4)
	tag := domain.Tag{
		ID:                 9,
		GroupID:            3,
		GroupKey:           "genre",
		GroupName:          "Genre",
		GroupAllowMultiple: true,
		GroupIsFilterable:  true,
		Name:               "Action",
		Slug:               "action",
		ParentID:           &parentID,
		SortOrder:          5,
		IsActive:           true,
		CreatedAt:          "2026-04-01T00:00:00Z",
		UpdatedAt:          "2026-04-02T00:00:00Z",
	}

	response := toTagResponse(tag)
	if response.ID != 9 || response.GroupID != 3 || response.Name != "Action" {
		t.Fatalf("response = %+v, want mapped tag fields", response)
	}
	if response.ParentID == nil || *response.ParentID != parentID {
		t.Fatalf("parent_id = %v, want %d", response.ParentID, parentID)
	}
}

func TestToTagGroupResponsesExposeTransportDTO(t *testing.T) {
	description := "Primary genre picker"
	response := toTagGroupResponse(domain.TagGroup{
		ID:            6,
		Key:           "genre",
		Name:          "Genre",
		Description:   &description,
		SortOrder:     1,
		AllowMultiple: true,
		IsFilterable:  true,
		CreatedAt:     "2026-04-01T00:00:00Z",
		UpdatedAt:     "2026-04-02T00:00:00Z",
	})

	if response.ID != 6 || response.Key != "genre" || response.Name != "Genre" {
		t.Fatalf("response = %+v, want mapped tag group fields", response)
	}
	if response.Description == nil || *response.Description != description {
		t.Fatalf("description = %v, want %q", response.Description, description)
	}
}
