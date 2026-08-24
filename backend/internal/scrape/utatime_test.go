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
	translation, romanized, language, err := parseUtatimeHTML(sampleUtatimeHTML)
	if err != nil {
		t.Fatalf("parseUtatimeHTML returned error: %v", err)
	}

	wantTranslation := "Compared child\n\nI already know"
	if translation != wantTranslation {
		t.Errorf("translation = %q, want %q", translation, wantTranslation)
	}

	wantRomanized := "Kurabe rarekko"
	if romanized != wantRomanized {
		t.Errorf("romanized = %q, want %q", romanized, wantRomanized)
	}

	if language != "en" {
		t.Errorf("language = %q, want %q", language, "en")
	}
}

func TestParseUtatimeHTML_FallsBackWhenNoEnglish(t *testing.T) {
	htmlNoEnglish := strings.Replace(sampleUtatimeHTML, `id="English"`, `id="Indonesian"`, 1)

	translation, _, language, err := parseUtatimeHTML(htmlNoEnglish)
	if err != nil {
		t.Fatalf("parseUtatimeHTML returned error: %v", err)
	}
	// With no English subtab, the first subtab found (German, in this
	// fixture) should be used instead of erroring out.
	if !strings.Contains(translation, "Verglichenes Kind") {
		t.Errorf("expected fallback to first available subtab, got %q", translation)
	}
	if language != "de" {
		t.Errorf("language = %q, want %q", language, "de")
	}
}

func TestParseUtatimeHTML_ErrorsWithoutTranslationsBlock(t *testing.T) {
	_, _, _, err := parseUtatimeHTML(`<html><body><div class="contents" id="Original"></div></body></html>`)
	if err == nil {
		t.Fatal("expected an error when there's no #Translations block")
	}
}

func TestToUtatimeGlobalURL(t *testing.T) {
	cases := map[string]string{
		"https://www.utatime.com/lyrics/kessoku-band/seiza-ni-naretara/":        "https://www.utatime.com/global/lyrics/kessoku-band/seiza-ni-naretara/",
		"https://www.utatime.com/global/lyrics/kessoku-band/seiza-ni-naretara/": "https://www.utatime.com/global/lyrics/kessoku-band/seiza-ni-naretara/", // already global: unchanged
		"https://www.utatime.com/series/some-anime-theme-songs/":                "https://www.utatime.com/series/some-anime-theme-songs/",                // not a /lyrics/ URL: unchanged
	}
	for in, want := range cases {
		if got := toUtatimeGlobalURL(in); got != want {
			t.Errorf("toUtatimeGlobalURL(%q) = %q, want %q", in, got, want)
		}
	}
}

// sampleUtatimeSearchHTML is a trimmed-down stand-in for the "html" field of
// a real /music-api/site/v1/search response: a search-results-meta line
// followed by result groups ("曲" for songs, "タイアップ" for tie-ins,
// "作詞者" for lyricists), discovered live against
// https://www.utatime.com/music-api/site/v1/search?q=... during development.
// Only "曲" results carry the "search-result-lyric" class.
const sampleUtatimeSearchHTML = `<div class="search-page-results">
<div class="search-results-meta">7 results</div>
<section class="search-group"><h2 class="search-group-title">曲</h2>
  <div class="search-group-results">
    <a class="search-result search-result-lyric" href="https://www.utatime.com/lyrics/kessoku-band/seiza-ni-naretara/">
      <div class="search-result-text"><div class="search-title">星座になれたら</div></div>
    </a>
    <a class="search-result search-result-lyric" href="https://www.utatime.com/lyrics/van-de-shop/comedy-na-hero-ni-nareta-nara/">
      <div class="search-result-text"><div class="search-title">コメディなヒーローになれたなら</div></div>
    </a>
  </div>
</section>
<section class="search-group"><h2 class="search-group-title">タイアップ</h2>
  <div class="search-group-results">
    <a class="search-result search-result-tiein" href="https://www.utatime.com/series/some-anime-theme-songs/">
      <div class="search-result-text"><div class="search-title">アニメ「なんとか」</div></div>
    </a>
  </div>
</section>
</div>`

func TestParseUtatimeSearchHTML_PicksFirstSongResult(t *testing.T) {
	got, err := parseUtatimeSearchHTML(sampleUtatimeSearchHTML)
	if err != nil {
		t.Fatalf("parseUtatimeSearchHTML returned error: %v", err)
	}
	want := "https://www.utatime.com/global/lyrics/kessoku-band/seiza-ni-naretara/"
	if got != want {
		t.Errorf("parseUtatimeSearchHTML = %q, want %q", got, want)
	}
}

func TestParseUtatimeSearchHTML_NoSongResult(t *testing.T) {
	noSongResults := `<div class="search-page-results"><div class="search-results-meta">0 results</div></div>`
	if _, err := parseUtatimeSearchHTML(noSongResults); err == nil {
		t.Fatal("expected an error when there's no song result in the search page")
	}

	// A tie-in-only match (no "search-result-lyric" class) shouldn't be
	// mistaken for a lyric page either.
	tieinOnly := `<div class="search-page-results"><a class="search-result search-result-tiein" href="https://www.utatime.com/series/x/"></a></div>`
	if _, err := parseUtatimeSearchHTML(tieinOnly); err == nil {
		t.Fatal("expected an error when only a non-lyric result is present")
	}
}

func TestParseUtatimeHTML_MissingRomajiIsNotAnError(t *testing.T) {
	htmlNoRomaji := strings.Replace(sampleUtatimeHTML, `id="Romaji"`, `id="NotRomaji"`, 1)

	translation, romanized, _, err := parseUtatimeHTML(htmlNoRomaji)
	if err != nil {
		t.Fatalf("parseUtatimeHTML returned error: %v", err)
	}
	if translation == "" {
		t.Error("expected translation to still be extracted")
	}
	if romanized != "" {
		t.Errorf("expected empty romanized when #Romaji is absent, got %q", romanized)
	}
}
