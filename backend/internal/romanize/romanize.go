// Package romanize converts Japanese text (kanji/kana) to romaji.
//
// This is the Go equivalent of kuromoji+kuroshiro (see plan-extended.md):
// kagome tokenizes the text and reports each token's reading in katakana
// (its dictionary is embedded in the binary, no separate file to load), and
// gojp/kana converts that katakana reading to romaji.
package romanize

import (
	"strings"

	"github.com/gojp/kana"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

// Romanizer converts Japanese text to romaji. It holds the tokenizer's
// dictionary, so callers should build one Romanizer at startup and reuse it
// (see cmd/server/main.go) rather than constructing one per request.
type Romanizer struct {
	tok *tokenizer.Tokenizer
}

// New builds a Romanizer, loading the embedded IPA dictionary once.
func New() (*Romanizer, error) {
	tok, err := tokenizer.New(ipa.Dict(), tokenizer.OmitBosEos())
	if err != nil {
		return nil, err
	}
	kana.Initialize()
	return &Romanizer{tok: tok}, nil
}

// sokuonSuffix is the small tsu (っ/ッ) that marks a doubled consonant. When
// it's the last kana of a token's reading, the doubling only makes sense
// together with the *next* token's leading consonant (e.g. 知って = "shi"+
// small-tsu | "te" -> "shitte"). Romanizing each token in isolation loses
// that lookahead and gojp/kana emits a garbled placeholder instead of the
// doubled consonant — so such a token's reading is merged into the next
// one before conversion instead of being converted on its own.
const sokuonSuffix = "ッ"

// ToRomaji tokenizes text and joins each token's romanized reading with
// spaces. Tokens with no dictionary reading (e.g. punctuation, text that's
// already Latin) fall back to their surface form.
func (r *Romanizer) ToRomaji(text string) string {
	tokens := r.tok.Tokenize(text)

	// Group token readings so that a trailing sokuon stays joined with the
	// reading that follows it, then romanize once per group — this is what
	// lets gojp/kana see the consonant it needs to double.
	var groups []string
	for _, t := range tokens {
		if t.Class == tokenizer.DUMMY {
			continue // BOS/EOS marker, shouldn't appear with OmitBosEos but skip defensively
		}

		reading, ok := t.Reading()
		if !ok || reading == "" {
			reading = t.Surface
		}

		if n := len(groups); n > 0 && strings.HasSuffix(groups[n-1], sokuonSuffix) {
			groups[n-1] += reading
		} else {
			groups = append(groups, reading)
		}
	}

	parts := make([]string, len(groups))
	for i, g := range groups {
		parts[i] = kana.KanaToRomaji(g)
	}

	return strings.Join(parts, " ")
}
