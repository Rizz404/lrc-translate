package align

import (
	"fmt"
	"reflect"
	"testing"
)

func TestAlign_ExactPositionalMatch(t *testing.T) {
	original := []string{"line one", "line two", "line three"}
	scraped := []string{"baris satu", "baris dua", "baris tiga"}

	got := Align(original, scraped)
	want := []string{"baris satu", "baris dua", "baris tiga"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAlign_DropsBlankLinesAndAnnotations(t *testing.T) {
	original := []string{"a", "b"}
	scraped := []string{"[Chorus]", "", "terjemahan a", "terjemahan b", ""}

	got := Align(original, scraped)
	want := []string{"terjemahan a", "terjemahan b"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAlign_BlockMatchWhenCountsDiffer(t *testing.T) {
	// original has 2 blocks (separated by an instrumental gap), scraped has
	// 2 blocks too, but the lines-per-block counts differ overall so a pure
	// positional match (case 1) won't apply — block alignment should kick in.
	original := []string{
		"verse line 1", "verse line 2", "verse line 3",
		"", // instrumental gap -> block boundary
		"chorus line 1", "chorus line 2",
	}
	scraped := []string{
		"v1", "v2", "v3",
		"",
		"c1", "c2",
	}

	got := Align(original, scraped)
	want := []string{"v1", "v2", "v3", "", "c1", "c2"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAlign_ProportionalFallbackOnMismatch(t *testing.T) {
	original := []string{"a", "b", "c", "d"}
	scraped := []string{"x", "y"} // fewer scraped lines than original, single block

	got := Align(original, scraped)

	if len(got) != len(original) {
		t.Fatalf("expected result len %d, got %d", len(original), len(got))
	}
	for i, v := range got {
		if v == "" {
			t.Errorf("index %d: expected some fallback text, got empty", i)
		}
	}
	// First original line should map to the first scraped line.
	if got[0] != "x" {
		t.Errorf("got[0] = %q, want %q", got[0], "x")
	}
}

func TestAlign_ProportionalFallbackSpreadsDuplicatesEvenly(t *testing.T) {
	// 61 original lines, 56 scraped lines (mirrors the real-world utatime.com
	// case that motivated proportionalIndex's rounding — see its doc
	// comment). A naive floor-division mapping bunches both position 0 and
	// 1 onto scraped index 0 before ever advancing; rounding should not
	// produce a duplicate in the first few positions.
	original := make([]string, 61)
	for i := range original {
		original[i] = fmt.Sprintf("orig-%d", i)
	}
	scraped := make([]string, 56)
	for i := range scraped {
		scraped[i] = fmt.Sprintf("scraped-%d", i)
	}

	got := Align(original, scraped)

	if got[0] == got[1] {
		t.Errorf("expected no duplicate at the very start of the mapping, got %q twice", got[0])
	}
	// Every original line should still get some translation (56 < 61, so
	// a handful of adjacent duplicates are unavoidable, just not clustered).
	for i, v := range got {
		if v == "" {
			t.Errorf("index %d: expected some fallback text, got empty", i)
		}
	}
}

func TestAlign_EmptyInputsDoNotPanic(t *testing.T) {
	if got := Align(nil, nil); len(got) != 0 {
		t.Errorf("expected empty result for nil original, got %v", got)
	}
	if got := Align([]string{"a", "b"}, nil); len(got) != 2 || got[0] != "" || got[1] != "" {
		t.Errorf("expected all-empty result for nil scraped, got %v", got)
	}
}

func TestAlign_PreservesInstrumentalGaps(t *testing.T) {
	original := []string{"a", "", "b"}
	scraped := []string{"x", "y"}

	got := Align(original, scraped)
	if got[1] != "" {
		t.Errorf("instrumental gap at index 1 should stay empty, got %q", got[1])
	}
}
