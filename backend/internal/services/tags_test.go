package services

import (
	"errors"
	"testing"

	"github.com/hao/game/internal/domain"
	"github.com/hao/game/internal/repositories"
)

func TestTagsServiceCreateGroupAppliesDefaultsAndSlugifiesKey(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewTagsService(repositories.NewTagsRepository(db))
	description := "  curated picks  "

	group, err := service.CreateGroup(domain.TagGroupWriteInput{
		Key:         "  Action Focus  ",
		Name:        " Action Focus ",
		Description: &description,
	})
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	if group.Key != "action-focus" {
		t.Fatalf("group.Key = %q, want action-focus", group.Key)
	}
	if group.Name != "Action Focus" {
		t.Fatalf("group.Name = %q, want Action Focus", group.Name)
	}
	if group.Description == nil || *group.Description != "curated picks" {
		t.Fatalf("group.Description = %v, want trimmed description", group.Description)
	}
	if group.SortOrder != 0 {
		t.Fatalf("group.SortOrder = %d, want 0", group.SortOrder)
	}
	if !group.AllowMultiple {
		t.Fatalf("expected AllowMultiple default true")
	}
	if !group.IsFilterable {
		t.Fatalf("expected IsFilterable default true")
	}
}

func TestTagsServiceCreateTagReusesExistingTagAndNormalizesDefaults(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewTagsService(repositories.NewTagsRepository(db))
	group, err := service.CreateGroup(domain.TagGroupWriteInput{
		Key:  "custom-genre",
		Name: "Genre",
	})
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	created, err := service.CreateTag(domain.TagWriteInput{
		GroupID: group.ID,
		Name:    "__new_tag__:genre:  Boss Rush  ",
	})
	if err != nil {
		t.Fatalf("CreateTag returned error: %v", err)
	}
	if created.Name != "Boss Rush" {
		t.Fatalf("created.Name = %q, want Boss Rush", created.Name)
	}
	if created.Slug != "boss-rush" {
		t.Fatalf("created.Slug = %q, want boss-rush", created.Slug)
	}
	if created.SortOrder != 0 {
		t.Fatalf("created.SortOrder = %d, want 0", created.SortOrder)
	}
	if !created.IsActive {
		t.Fatalf("expected IsActive default true")
	}

	reused, err := service.CreateTag(domain.TagWriteInput{
		GroupID: group.ID,
		Name:    " boss rush ",
	})
	if err != nil {
		t.Fatalf("CreateTag duplicate returned error: %v", err)
	}
	if reused.ID != created.ID {
		t.Fatalf("reused.ID = %d, want %d", reused.ID, created.ID)
	}

	tags, err := service.ListTags(domain.TagsListParams{GroupID: group.ID})
	if err != nil {
		t.Fatalf("ListTags returned error: %v", err)
	}
	if len(tags) != 1 {
		t.Fatalf("len(tags) = %d, want 1", len(tags))
	}
}

func TestTagsServiceCreateTagRejectsInputThatCannotProduceSlug(t *testing.T) {
	db := openServicesTestDB(t)
	defer func() { _ = db.Close() }()

	service := NewTagsService(repositories.NewTagsRepository(db))
	group, err := service.CreateGroup(domain.TagGroupWriteInput{
		Key:  "custom-theme",
		Name: "Theme",
	})
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	_, err = service.CreateTag(domain.TagWriteInput{
		GroupID: group.ID,
		Name:    "***",
	})
	if !errors.Is(err, ErrValidation) {
		t.Fatalf("CreateTag error = %v, want ErrValidation", err)
	}
}
