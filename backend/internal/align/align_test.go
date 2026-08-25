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

func TestAlign_BlockHeuristicHandlesMergeInsideBlock(t *testing.T) {
	// Regression test for the real utatime.com "kuraberarekko" bug report:
	// within one block, the scraped side merges a PAIR OF LINES IN THE
	// MIDDLE of the block (not at the very end) into a single translated
	// line, so the block is short by one. The old alignByBlock copied
	// scrapedBlock[i] directly for every i up to len(scrapedBlock)-1 and
	// only used proportionalIndex for the leftover tail position — which
	// is correct when the deficit is at the end, but produces an off-by-
	// one line shift for every position after a mid-block merge, plus an
	// exact duplicate at the very end of the block (scrapedBlock's last
	// item ends up doing double duty). Distributing every position in the
	// block proportionally (not just the excess ones) fixes both.
	original := []string{
		"o1", "o2", "o3", "o4", "o5", "o6", "o7", "o8",
		"", // instrumental gap -> block boundary
		"p1", "p2",
	}
	scraped := []string{
		"s1", "s2", "s3-merged-of-o3-o4", "s4", "s5", "s6-merged-of-o7-o8",
		"",
		"t1", "t2",
	}

	got := Align(original, scraped)

	// The merged translations should cover BOTH original lines they stand
	// in for, and nothing should duplicate onto a line that isn't part of
	// that merge.
	want := []string{
		"s1", "s2",
		"s3-merged-of-o3-o4", "s3-merged-of-o3-o4",
		"s4", "s5",
		"s6-merged-of-o7-o8", "s6-merged-of-o7-o8",
		"",
		"t1", "t2",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	// The bug's most visible symptom: content that belongs to the NEXT
	// block bleeding into this one. Make sure the second block's content
	// stayed in the second block.
	if got[9] != "t1" || got[10] != "t2" {
		t.Errorf("second block got corrupted: got[9]=%q got[10]=%q", got[9], got[10])
	}
}

func TestAlign_PositionalMatchBeatsBlockHeuristicWhenBlanksLineUp(t *testing.T) {
	// Regression test for the utatime.com "kuraberarekko" bug report: a
	// scraped source (like utatime.com's per-line ".line-text" spans) that
	// is already position-aligned with original, blanks included, used to
	// get routed through the block/proportional heuristics instead
	// (because the old exact-match check compared against the
	// blank-stripped scraped count, which almost never equals
	// len(original) once instrumental gaps are involved). That produced
	// duplicated/shifted lines even though the source data was already
	// perfectly aligned. It should now be detected and copied 1:1 instead.
	original := []string{
		"line 1", "line 2", "line 3", "line 4",
		"line 5", "line 6", "line 7", "line 8",
		"", // instrumental gap
		"line 9", "line 10",
	}
	scraped := []string{
		"t1", "t2", "t3", "t4",
		"t5", "t6", "t7", "t8",
		"",
		"t9", "t10",
	}

	got := Align(original, scraped)
	want := []string{
		"t1", "t2", "t3", "t4",
		"t5", "t6", "t7", "t8",
		"",
		"t9", "t10",
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAlign_PositionalMatchIgnoresAnnotationLines(t *testing.T) {
	original := []string{"line 1", "line 2"}
	scraped := []string{"[Chorus]", "t1", "t2"}

	got := Align(original, scraped)
	want := []string{"t1", "t2"}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestAlignWithContext_ProportionalGivesNeighborsForReview(t *testing.T) {
	// Mirrors the real utatime.com case: original longer than scraped, so
	// alignProportional fires. A UI reviewing a needs_review line should be
	// able to see the raw scraped line right before/after the one that got
	// picked, to spot a bad guess like the Kuraberarekko duplicate report.
	original := []string{"o1", "o2", "o3", "o4"}
	scraped := []string{"s1", "s2", "s3"} // one short -> proportional fallback

	result, contexts := AlignWithContext(original, scraped)

	if len(contexts) != len(original) {
		t.Fatalf("expected %d contexts, got %d", len(original), len(contexts))
	}
	for i, ctx := range contexts {
		if ctx.Matched != result[i] {
			t.Errorf("index %d: Context.Matched %q != result %q", i, ctx.Matched, result[i])
		}
	}
	// The first position's Matched should be scraped's first line, with no
	// Prev (nothing comes before it) but a Next.
	if contexts[0].Matched != "s1" || contexts[0].Prev != "" || contexts[0].Next != "s2" {
		t.Errorf("contexts[0] = %+v, want Matched=s1 Prev='' Next=s2", contexts[0])
	}
	// The last scraped line has no Next.
	last := contexts[len(contexts)-1]
	if last.Next != "" {
		t.Errorf("last context should have no Next, got %+v", last)
	}
}

func TestAlignWithContext_BlankPositionsGetZeroValueContext(t *testing.T) {
	original := []string{"a", "", "b"}
	scraped := []string{"x", "y"}

	_, contexts := AlignWithContext(original, scraped)

	if contexts[1] != (Context{}) {
		t.Errorf("expected zero-value Context for the blank gap, got %+v", contexts[1])
	}
}

func TestAlignWithContext_PositionalStrategyUsesAdjacentRawLines(t *testing.T) {
	original := []string{"o1", "o2", "o3"}
	scraped := []string{"s1", "s2", "s3"}

	_, contexts := AlignWithContext(original, scraped)

	if contexts[1] != (Context{Prev: "s1", Matched: "s2", Next: "s3"}) {
		t.Errorf("contexts[1] = %+v, want Prev=s1 Matched=s2 Next=s3", contexts[1])
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
