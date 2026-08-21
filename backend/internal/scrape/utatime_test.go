package scrape

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Kenshi Yonezu":     "kenshi-yonezu",
		"Compared Child":    "compared-child",
		"  Extra   Spaces ": "extra-spaces",
		"Don't Stop!":       "dont-stop",
	}
	for in, want := range cases {
		if got := slugify(in); got != want {
			t.Errorf("slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestGuessUtatimeURLs(t *testing.T) {
	got := guessUtatimeURLs("Kenshi Yonezu", "Lemon")
	want := "https://www.utatime.com/global/lyrics/kenshi-yonezu/lemon/"
	if len(got) == 0 || got[0] != want {
		t.Errorf("guessUtatimeURLs = %v, want first entry %q", got, want)
	}

	if got := guessUtatimeURLs("", "Lemon"); got != nil {
		t.Errorf("expected nil candidates for empty artist, got %v", got)
	}
}

// sample is a trimmed-down stand-in for the real page structure discovered
// during development (see utatime.go's doc comment): #Original and
// #Translations tabs, each a ".olyrictext" block of numbered
// ".line-number"/".line-text" span pairs, where a lone "<br/>" marks a blank
// (block-separator) line.
const sampleUtatimeHTML = `
<html><body>
<div class="contents" id="Romaji">
  <div class="olyrictext">
    <span class="line-number">1.</span><span class="line-text">Kurabe rarekko<br /></span>
  </div>
</div>
<div class="contents" id="Original">
  <div class="olyrictext">
    <span class="line-number">1.</span><span class="line-text">くらべられっ子<br /></span>
    <span class="line-number">2.</span><span class="line-text"><br /></span>
    <span class="line-number">3.</span><span class="line-text">とっくに知ってるよ</span>
  </div>
</div>
<div class="contents" id="Translations">
  <div id="langtabs">
    <div class="subcontentsblock">
      <div class="contents subcontents" id="German">
        <div class="olyrictext">
          <span class="line-number">1.</span><span class="line-text">Verglichenes Kind<br /></span>
        </div>
      </div>
      <div class="contents subcontents" id="English">
        <div class="olyrictext">
          <span class="line-number">1.</span><span class="line-text">Compared child<br /></span>
          <span class="line-number">2.</span><span class="line-text"><br /></span>
          <span class="line-number">3.</span><span class="line-text">I already know</span>
        </div>
      </div>
    </div>
  </div>
</div>
</body></html>`

func TestParseUtatimeHTML_PrefersEnglish(t *testing.T) {
	got, err := parseUtatimeHTML(sampleUtatimeHTML)
	if err != nil {
		t.Fatalf("parseUtatimeHTML returned error: %v", err)
	}

	want := "Compared child\n\nI already know"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestParseUtatimeHTML_FallsBackWhenNoEnglish(t *testing.T) {
	htmlNoEnglish := strings.Replace(sampleUtatimeHTML, `id="English"`, `id="Indonesian"`, 1)

	got, err := parseUtatimeHTML(htmlNoEnglish)
	if err != nil {
		t.Fatalf("parseUtatimeHTML returned error: %v", err)
	}
	// With no English subtab, the first subtab found (German, in this
	// fixture) should be used instead of erroring out.
	if !strings.Contains(got, "Verglichenes Kind") {
		t.Errorf("expected fallback to first available subtab, got %q", got)
	}
}

func TestParseUtatimeHTML_ErrorsWithoutTranslationsBlock(t *testing.T) {
	_, err := parseUtatimeHTML(`<html><body><div class="contents" id="Original"></div></body></html>`)
	if err == nil {
		t.Fatal("expected an error when there's no #Translations block")
	}
}
