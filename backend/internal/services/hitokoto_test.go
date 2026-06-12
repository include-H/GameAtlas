package services

import (
	"testing"

	"github.com/hao/game/internal/domain"
)

func TestHitokotoServiceRandomReturnsGameSentenceWithinLengthRange(t *testing.T) {
	service := NewHitokotoService()

	sentence, err := service.Random(HitokotoQuery{
		Categories: []string{"c"},
		MinLength:  8,
		MaxLength:  20,
	})
	if err != nil {
		t.Fatalf("Random returned error: %v", err)
	}
	if sentence == nil {
		t.Fatal("Random returned nil sentence")
	}
	if sentence.Type != "c" {
		t.Fatalf("sentence.Type = %q, want c", sentence.Type)
	}
	if sentence.Length < 8 || sentence.Length > 20 {
		t.Fatalf("sentence.Length = %d, want within [8,20]", sentence.Length)
	}
	if sentence.Hitokoto == "" || sentence.From == "" || sentence.UUID == "" {
		t.Fatalf("unexpected empty sentence fields: %+v", sentence)
	}
}

func TestHitokotoServiceRandomRejectsInvalidLengthRange(t *testing.T) {
	service := NewHitokotoService()

	_, err := service.Random(HitokotoQuery{
		Categories: []string{"c"},
		MinLength:  30,
		MaxLength:  10,
	})
	if err != domain.ErrValidation {
		t.Fatalf("err = %v, want %v", err, domain.ErrValidation)
	}
}

func TestHitokotoServiceRandomReturnsNotFoundForUnsupportedCategory(t *testing.T) {
	service := NewHitokotoService()

	_, err := service.Random(HitokotoQuery{
		Categories: []string{"a"},
		MinLength:  0,
		MaxLength:  30,
	})
	if err != domain.ErrNotFound {
		t.Fatalf("err = %v, want %v", err, domain.ErrNotFound)
	}
}

func TestNewHitokotoServicePanicsOnInvalidEmbeddedJSON(t *testing.T) {
	original := hitokotoGameSentencesJSON
	hitokotoGameSentencesJSON = []byte("{")
	defer func() {
		hitokotoGameSentencesJSON = original
	}()

	defer func() {
		if recover() == nil {
			t.Fatal("expected NewHitokotoService to panic on invalid embedded json")
		}
	}()

	NewHitokotoService()
}
