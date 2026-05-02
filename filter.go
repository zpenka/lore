package lore

import (
	"strings"

	"github.com/sahilm/fuzzy"
)

// fuzzyFilterCandidates takes a query and a list of candidate strings,
// and returns them ranked by fuzzy match score (best first).
// If query is empty, all candidates are returned in original order.
func fuzzyFilterCandidates(query string, candidates []string) []string {
	query = strings.TrimSpace(query)
	if query == "" {
		return candidates
	}

	// Use fuzzy.Find to rank candidates by match score
	matches := fuzzy.Find(query, candidates)
	result := make([]string, len(matches))
	for i, match := range matches {
		result[i] = match.Str
	}
	return result
}
