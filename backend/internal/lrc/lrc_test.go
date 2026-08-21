package lrc

import "testing"

func TestParse_Basic(t *testing.T) {
	raw := "[ti:Sample]\n[ar:Someone]\n[00:01.00]first line\n[00:12.34]second line\n\n[00:00.50]zero-ish line\n"

	lines, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %+v", len(lines), lines)
	}

	// Must come back sorted by time.
	want := []Line{
		{TimeMs: 500, Timestamp: "[00:00.50]", Text: "zero-ish line"},
		{TimeMs: 1000, Timestamp: "[00:01.00]", Text: "first line"},
		{TimeMs: 12340, Timestamp: "[00:12.34]", Text: "second line"},
	}
	for i, w := range want {
		if lines[i] != w {
			t.Errorf("line %d: got %+v, want %+v", i, lines[i], w)
		}
	}
}

func TestParse_MultipleTimestampsOnOneLine(t *testing.T) {
	raw := "[00:01.00][00:05.00]repeated chorus"

	lines, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(lines) != 2 {
		t.Fatalf("expected 2 expanded lines, got %d: %+v", len(lines), lines)
	}
	if lines[0].Text != "repeated chorus" || lines[1].Text != "repeated chorus" {
		t.Errorf("expanded lines should share the same text, got %+v", lines)
	}
}

func TestParse_IgnoresBlankAndMetadataOnlyLines(t *testing.T) {
	raw := "[length: 03:45]\n\n   \n[00:02.00]only real line\n"

	lines, err := Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d: %+v", len(lines), lines)
	}
	if lines[0].Text != "only real line" {
		t.Errorf("got text %q", lines[0].Text)
	}
}

func TestFormat_RoundTripsThroughParse(t *testing.T) {
	original := []Line{
		{TimeMs: 12340, Text: "second"},
		{TimeMs: 1000, Text: "first"},
	}

	formatted := Format(original)
	reparsed, err := Parse(formatted)
	if err != nil {
		t.Fatalf("Parse(Format(...)) returned error: %v", err)
	}

	if len(reparsed) != 2 {
		t.Fatalf("expected 2 lines after round-trip, got %d", len(reparsed))
	}
	if reparsed[0].Text != "first" || reparsed[0].TimeMs != 1000 {
		t.Errorf("round-trip line 0 mismatch: %+v", reparsed[0])
	}
	if reparsed[1].Text != "second" || reparsed[1].TimeMs != 12340 {
		t.Errorf("round-trip line 1 mismatch: %+v", reparsed[1])
	}
}

func TestFormatTimestamp(t *testing.T) {
	cases := map[int64]string{
		0:      "[00:00.00]",
		500:    "[00:00.50]",
		1000:   "[00:01.00]",
		61_230: "[01:01.23]",
	}
	for ms, want := range cases {
		if got := formatTimestamp(ms); got != want {
			t.Errorf("formatTimestamp(%d) = %q, want %q", ms, got, want)
		}
	}
}
