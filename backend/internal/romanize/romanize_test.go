package romanize

import (
	"strings"
	"testing"
)

func TestToRomaji_Basic(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	got := r.ToRomaji("食べる")
	want := "taberu"
	if got != want {
		t.Errorf("ToRomaji(食べる) = %q, want %q", got, want)
	}
}

func TestToRomaji_AlreadyLatinPassesThrough(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	got := r.ToRomaji("hello")
	if got == "" {
		t.Errorf("ToRomaji(hello) returned empty string")
	}
}

// Regression test: a sokuon (っ) landing on a token boundary used to leave a
// garbled placeholder instead of doubling the next consonant (e.g. "知って"
// -> "shi<U+FFFD> te" instead of "shitte") because each token's reading was
// romanized in isolation. See the sokuonSuffix grouping in ToRomaji.
func TestToRomaji_SokuonAcrossTokenBoundary(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	cases := map[string]string{
		"知ってる":  "shitteru",
		"劣ってるのは": "ototteru no ha",
		"解ってるよ": "wakatteru yo",
	}
	for input, want := range cases {
		got := r.ToRomaji(input)
		if got != want {
			t.Errorf("ToRomaji(%q) = %q, want %q", input, got, want)
		}
		if strings.ContainsRune(got, '�') {
			t.Errorf("ToRomaji(%q) = %q contains a replacement character (U+FFFD)", input, got)
		}
	}
}
