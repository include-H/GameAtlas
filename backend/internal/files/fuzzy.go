package files

import (
	"strings"
	"unicode"
)

// JaroWinkler calculates the Jaro-Winkler similarity between two strings.
// Returns a value between 0 and 1, where 1 means exact match.
// Operates on runes (Unicode code points) for correct multi-byte character support.
func JaroWinkler(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	r1 := []rune(s1)
	r2 := []rune(s2)
	len1 := len(r1)
	len2 := len(r2)

	if len1 == 0 || len2 == 0 {
		return 0.0
	}

	// Calculate match distance
	matchDistance := (max(len1, len2) / 2) - 1
	if matchDistance < 0 {
		matchDistance = 0
	}

	s1Matches := make([]bool, len1)
	s2Matches := make([]bool, len2)

	matches := 0
	transpositions := 0

	// Find matches
	for i := 0; i < len1; i++ {
		start := max(0, i-matchDistance)
		end := min(i+matchDistance+1, len2)

		for j := start; j < end; j++ {
			if s2Matches[j] || r1[i] != r2[j] {
				continue
			}
			s1Matches[i] = true
			s2Matches[j] = true
			matches++
			break
		}
	}

	if matches == 0 {
		return 0.0
	}

	// Count transpositions
	k := 0
	for i := 0; i < len1; i++ {
		if !s1Matches[i] {
			continue
		}
		for !s2Matches[k] {
			k++
		}
		if r1[i] != r2[k] {
			transpositions++
		}
		k++
	}

	jaro := (float64(matches)/float64(len1) +
		float64(matches)/float64(len2) +
		(float64(matches)-float64(transpositions)/2.0)/float64(matches)) / 3.0

	// Calculate common prefix (up to 4 runes)
	prefix := 0
	for i := 0; i < min(4, min(len1, len2)); i++ {
		if r1[i] == r2[i] {
			prefix++
		} else {
			break
		}
	}

	// Winkler modification
	return jaro + float64(prefix)*0.1*(1.0-jaro)
}

// FuzzyMatch returns true if query matches target, combining substring matching
// with JaroWinkler similarity. Substring matches (e.g. "炼金" in "炼金工房") always pass.
func FuzzyMatch(query, target string) bool {
	if strings.Contains(target, query) {
		return true
	}
	return JaroWinkler(query, target) >= 0.8
}

// NormalizeForSearch removes special characters and converts to lowercase for fuzzy matching.
func NormalizeForSearch(s string) string {
	var builder strings.Builder
	builder.Grow(len(s))

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			builder.WriteRune(unicode.ToLower(r))
		}
		// Skip special characters like : - 　 etc.
	}

	return builder.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
