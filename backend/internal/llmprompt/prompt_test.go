package llmprompt

import (
	"strings"
	"testing"
)

func TestBuildBatch_IncludesEveryLineInOrder(t *testing.T) {
	lines := []string{"hello", "world"}
	prompt := BuildBatch(lines, "en", "id")

	if want := "1. hello"; !strings.Contains(prompt, want) {
		t.Errorf("prompt missing %q:\n%s", want, prompt)
	}
	if want := "2. world"; !strings.Contains(prompt, want) {
		t.Errorf("prompt missing %q:\n%s", want, prompt)
	}
	if want := "exactly 2 lines"; !strings.Contains(prompt, want) {
		t.Errorf("prompt missing line-count instruction %q:\n%s", want, prompt)
	}
}

func TestParseBatch_DecodesPlainJSONArray(t *testing.T) {
	got, err := ParseBatch(`["halo", "dunia"]`, 2)
	if err != nil {
		t.Fatalf("ParseBatch returned error: %v", err)
	}
	if len(got) != 2 || got[0] != "halo" || got[1] != "dunia" {
		t.Errorf("got %v, want [halo dunia]", got)
	}
}

func TestParseBatch_StripsMarkdownFence(t *testing.T) {
	got, err := ParseBatch("```json\n[\"halo\", \"dunia\"]\n```", 2)
	if err != nil {
		t.Fatalf("ParseBatch returned error: %v", err)
	}
	if len(got) != 2 || got[0] != "halo" || got[1] != "dunia" {
		t.Errorf("got %v, want [halo dunia]", got)
	}
}

func TestParseBatch_ErrorsOnCountMismatch(t *testing.T) {
	_, err := ParseBatch(`["halo"]`, 2)
	if err == nil {
		t.Fatal("expected an error for a count mismatch, got nil")
	}
}

func TestParseBatch_ErrorsOnInvalidJSON(t *testing.T) {
	_, err := ParseBatch("not json at all", 1)
	if err == nil {
		t.Fatal("expected an error for invalid JSON, got nil")
	}
}
