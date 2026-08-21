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

// ToRomaji tokenizes text and joins each token's romanized reading with
// spaces. Tokens with no dictionary reading (e.g. punctuation, text that's
// already Latin) fall back to their surface form.
func (r *Romanizer) ToRomaji(text string) string {
	tokens := r.tok.Tokenize(text)
	parts := make([]string, 0, len(tokens))

	for _, t := range tokens {
		if t.Class == tokenizer.DUMMY {
			continue // BOS/EOS marker, shouldn't appear with OmitBosEos but skip defensively
		}

		reading, ok := t.Reading()
		if !ok || reading == "" {
			reading = t.Surface
		}
		parts = append(parts, kana.KanaToRomaji(reading))
	}

	return strings.Join(parts, " ")
}
