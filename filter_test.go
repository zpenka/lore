package lore

import (
	"testing"
)

// TestFilter_EmptyQueryReturnsAll verifies that an empty query returns all candidates unchanged.
func TestFilter_EmptyQueryReturnsAll(t *testing.T) {
	candidates := []string{"foo", "bar", "baz"}
	got := fuzzyFilterCandidates("", candidates)
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}
	// Check all candidates are present
	if got[0] != "foo" || got[1] != "bar" || got[2] != "baz" {
		t.Errorf("want [foo bar baz], got %v", got)
	}
}

// TestFilter_FuzzySubsequenceMatches verifies fuzzy matching finds subsequences that substring would miss.
func TestFilter_FuzzySubsequenceMatches(t *testing.T) {
	candidates := []string{"api-server", "billing", "auth", "apple-pie"}
	got := fuzzyFilterCandidates("as", candidates)
	if len(got) == 0 {
		t.Fatalf("want matches for 'as' fuzzy subsequence, got 0 results")
	}
	// "api-server" should match via fuzzy (a...s)
	// and "apple-pie" should also match (a...s)
	if got[0] != "api-server" && got[0] != "apple-pie" {
		t.Errorf("want api-server or apple-pie first (best fuzzy score), got %s", got[0])
	}
}

// TestFilter_ExactPrefixRanksFirst verifies that exact prefix matches rank highest in fuzzy.
func TestFilter_ExactPrefixRanksFirst(t *testing.T) {
	candidates := []string{"xfoox", "foobar", "fxoxox"}
	got := fuzzyFilterCandidates("foo", candidates)
	if len(got) == 0 {
		t.Fatalf("want matches for 'foo', got 0 results")
	}
	if got[0] != "foobar" {
		t.Errorf("want foobar first (best fuzzy score), got %s", got[0])
	}
}

// TestFilter_CaseInsensitive verifies fuzzy matching is case-insensitive.
func TestFilter_CaseInsensitive(t *testing.T) {
	candidates := []string{"MyProject", "otherproject", "MYPROJ"}
	got := fuzzyFilterCandidates("myp", candidates)
	if len(got) == 0 {
		t.Fatalf("want matches for 'myp' case-insensitive, got 0 results")
	}
	// Both "MyProject" and "MYPROJ" should match
	if len(got) < 1 {
		t.Errorf("want at least 1 match, got %d", len(got))
	}
}

// TestFilter_NoMatch returns empty when nothing matches.
func TestFilter_NoMatch(t *testing.T) {
	candidates := []string{"foo", "bar", "baz"}
	got := fuzzyFilterCandidates("xyz", candidates)
	if len(got) != 0 {
		t.Errorf("want 0 matches for 'xyz', got %d: %v", len(got), got)
	}
}
