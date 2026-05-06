package lore

import (
	"testing"
	"time"
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

// ----- fuzzyFilterSessions helper (1B) -----

func TestFuzzyFilterSessions_EmptyTextReturnsAll(t *testing.T) {
	sessions := []Session{
		{Project: "alpha", Slug: "s1", Timestamp: time.Now()},
		{Project: "beta", Slug: "s2", Timestamp: time.Now()},
	}
	got := fuzzyFilterSessions("", func(s Session) string { return s.Project }, sessions)
	if len(got) != 2 {
		t.Fatalf("empty text: want 2 sessions, got %d", len(got))
	}
	if got[0].Slug != "s1" || got[1].Slug != "s2" {
		t.Errorf("empty text: want original order [s1 s2], got [%s %s]", got[0].Slug, got[1].Slug)
	}
}

func TestFuzzyFilterSessions_WhitespaceTextReturnsAll(t *testing.T) {
	sessions := []Session{
		{Project: "alpha", Slug: "s1"},
		{Project: "beta", Slug: "s2"},
	}
	got := fuzzyFilterSessions("   \t  ", func(s Session) string { return s.Project }, sessions)
	if len(got) != 2 {
		t.Errorf("whitespace text: want 2 sessions, got %d", len(got))
	}
}

func TestFuzzyFilterSessions_NoMatchReturnsEmpty(t *testing.T) {
	sessions := []Session{
		{Project: "alpha", Slug: "s1"},
		{Project: "beta", Slug: "s2"},
	}
	got := fuzzyFilterSessions("xyzzy", func(s Session) string { return s.Project }, sessions)
	if len(got) != 0 {
		t.Errorf("no match: want 0 sessions, got %d: %v", len(got), got)
	}
}

func TestFuzzyFilterSessions_MatchesPreserveOriginalOrder(t *testing.T) {
	sessions := []Session{
		{Project: "alpha", Slug: "s1"},
		{Project: "zeta", Slug: "s2"},
		{Project: "alpha-two", Slug: "s3"},
	}
	got := fuzzyFilterSessions("alpha", func(s Session) string { return s.Project }, sessions)
	if len(got) != 2 {
		t.Fatalf("want 2 matches for 'alpha', got %d: %v", len(got), got)
	}
	// Original input order: s1 then s3 (s2 doesn't match).
	if got[0].Slug != "s1" || got[1].Slug != "s3" {
		t.Errorf("want original order [s1 s3], got [%s %s]", got[0].Slug, got[1].Slug)
	}
}

func TestFuzzyFilterSessions_CustomCandidateFunc(t *testing.T) {
	sessions := []Session{
		{Project: "myapp", Branch: "main", Slug: "some-work"},
		{Project: "other", Branch: "feature-foo", Slug: "foo-session"},
		{Project: "third", Branch: "bugfix", Slug: "foo-work"},
	}
	candidate := func(s Session) string { return s.Slug + " " + s.Project + " " + s.Branch }
	got := fuzzyFilterSessions("foo", candidate, sessions)
	if len(got) == 0 {
		t.Fatal("composite candidate 'foo': want matches, got 0")
	}
	for _, s := range got {
		if s.Slug == "some-work" {
			t.Errorf("composite candidate 'foo': should not match myapp/main/some-work, got %v", s)
		}
	}
}

func TestFuzzyFilterSessions_DuplicateCandidates_AllSessionsKept(t *testing.T) {
	// Two sessions share the same project — both must be included if the project matches.
	sessions := []Session{
		{Project: "grit", Slug: "s1"},
		{Project: "other", Slug: "s2"},
		{Project: "grit", Slug: "s3"},
	}
	got := fuzzyFilterSessions("grit", func(s Session) string { return s.Project }, sessions)
	if len(got) != 2 {
		t.Fatalf("duplicate candidates: want 2 grit sessions, got %d: %v", len(got), got)
	}
	if got[0].Slug != "s1" || got[1].Slug != "s3" {
		t.Errorf("duplicate candidates: want [s1 s3] in original order, got [%s %s]", got[0].Slug, got[1].Slug)
	}
}
