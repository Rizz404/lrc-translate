package romanize

import "testing"

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
