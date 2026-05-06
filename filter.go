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

// fuzzyFilterSessions returns the subset of sessions whose extracted
// candidate string fuzzy-matches text. Empty/whitespace text returns
// sessions unchanged. Results preserve the original input order; the
// fuzzy ranking is used only for inclusion, not ordering.
func fuzzyFilterSessions(text string, candidate func(Session) string, sessions []Session) []Session {
	if strings.TrimSpace(text) == "" {
		return sessions
	}
	candidates := make([]string, len(sessions))
	for i, s := range sessions {
		candidates[i] = candidate(s)
	}
	matched := fuzzyFilterCandidates(text, candidates)
	matchSet := make(map[string]bool, len(matched))
	for _, c := range matched {
		matchSet[c] = true
	}
	var out []Session
	for i, s := range sessions {
		if matchSet[candidates[i]] {
			out = append(out, s)
		}
	}
	return out
}
