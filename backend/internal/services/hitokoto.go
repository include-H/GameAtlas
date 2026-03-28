package services

import (
	_ "embed"
	"encoding/json"
	"math/rand/v2"
	"strings"
)

//go:embed data/hitokoto_game_sentences.json
var hitokotoGameSentencesJSON []byte

type HitokotoSentence struct {
	ID         int64   `json:"id"`
	UUID       string  `json:"uuid"`
	Hitokoto   string  `json:"hitokoto"`
	Type       string  `json:"type"`
	From       string  `json:"from"`
	FromWho    *string `json:"from_who"`
	Creator    string  `json:"creator"`
	CreatorUID int64   `json:"creator_uid"`
	Reviewer   int64   `json:"reviewer"`
	CommitFrom string  `json:"commit_from"`
	CreatedAt  string  `json:"created_at"`
	Length     int     `json:"length"`
}

type HitokotoQuery struct {
	Categories []string
	MinLength  int
	MaxLength  int
}

type HitokotoService struct {
	gameSentences []HitokotoSentence
}

func NewHitokotoService() *HitokotoService {
	sentences := make([]HitokotoSentence, 0)
	_ = json.Unmarshal(hitokotoGameSentencesJSON, &sentences)

	return &HitokotoService{
		gameSentences: sentences,
	}
}

func (s *HitokotoService) Random(query HitokotoQuery) (*HitokotoSentence, error) {
	minLength := max(query.MinLength, 0)
	maxLength := query.MaxLength
	if maxLength <= 0 {
		maxLength = 30
	}
	if maxLength < minLength {
		return nil, ErrValidation
	}

	categories := normalizeHitokotoCategories(query.Categories)
	if len(query.Categories) == 0 {
		categories = []string{"c"}
	} else if len(categories) == 0 {
		return nil, ErrNotFound
	}

	targets := make([]HitokotoSentence, 0, 64)
	for _, sentence := range s.gameSentences {
		if !containsString(categories, sentence.Type) {
			continue
		}
		if sentence.Length < minLength || sentence.Length > maxLength {
			continue
		}
		targets = append(targets, sentence)
	}

	if len(targets) == 0 {
		return nil, ErrNotFound
	}

	selected := targets[rand.IntN(len(targets))]
	return &selected, nil
}

func normalizeHitokotoCategories(categories []string) []string {
	if len(categories) == 0 {
		return nil
	}

	result := make([]string, 0, len(categories))
	seen := make(map[string]struct{}, len(categories))
	for _, category := range categories {
		value := strings.ToLower(strings.TrimSpace(category))
		if value == "" || value != "c" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
