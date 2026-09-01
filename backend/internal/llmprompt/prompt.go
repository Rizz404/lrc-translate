// Package llmprompt builds the song-lyric translation prompt shared by every
// LLM-based MT backend (internal/gemini, internal/localllm). Unlike plain
// NMT (internal/libretranslate), an LLM can be steered by instructions to
// translate like a song lyric — natural/idiomatic, informal register, full
// coverage of the line — instead of a stiff word-for-word mapping. The
// instructions themselves don't depend on which LLM API is calling them, so
// they live here once instead of being duplicated per backend.
package llmprompt

import "fmt"

// langNames maps common ISO codes to a display name for the prompt — most
// LLMs understand bare codes fine too, but a name reads more naturally in an
// instruction and avoids any ambiguity (e.g. "id" vs "auto").
var langNames = map[string]string{
	"ja": "Japanese",
	"en": "English",
	"id": "Indonesian",
	"ko": "Korean",
	"zh": "Chinese",
	"es": "Spanish",
	"fr": "French",
	"de": "German",
	"pt": "Portuguese",
	"ru": "Russian",
	"ar": "Arabic",
	"hi": "Hindi",
	"th": "Thai",
	"vi": "Vietnamese",
}

func langName(code string) string {
	if name, ok := langNames[code]; ok {
		return name
	}
	return code
}

// Build returns the prompt text for translating one line of song lyrics from
// sourceLang to targetLang.
func Build(text, sourceLang, targetLang string) string {
	var from string
	if sourceLang == "" || sourceLang == "auto" {
		from = "the source language (detect it automatically)"
	} else {
		from = langName(sourceLang)
	}
	to := langName(targetLang)

	return fmt.Sprintf(
		"You are translating one line from a song's lyrics, from %s to %s. "+
			"This line is one of many sent as separate, independent requests "+
			"for the same song, so you won't see the other lines — follow the "+
			"rules below exactly as given so every line stays consistent with "+
			"the rest of the song even without that context.\n\n"+
			"Rules:\n"+
			"1. Register: song lyrics address people informally. Use the "+
			"casual/informal form of \"I\" and \"you\" in %s, never a formal or "+
			"polite register (e.g. in Indonesian use \"aku\"/\"kamu\", never "+
			"\"saya\"/\"Anda\"; in French use \"tu\", never \"vous\") — unless "+
			"the source line is itself unmistakably formal or reverent in "+
			"tone. Never mix formal and informal address across lines.\n"+
			"2. Translate the FULL line. Every word must end up in %s. Never "+
			"leave part or all of the line untranslated, and never output the "+
			"source text unchanged — the only exception is a proper noun "+
			"(a name, a place) with no natural equivalent.\n"+
			"3. Prioritize meaning over literal wording: render what the line "+
			"actually means and how a native %s speaker would naturally say it "+
			"in a song, not a stiff word-for-word mapping of the source "+
			"grammar.\n"+
			"4. Keep it roughly as concise as the original line, so it still "+
			"reads like a lyric.\n\n"+
			"Output ONLY the translated line itself, with no quotation marks, "+
			"romanization, explanation, or extra commentary.\n\nLine: %q",
		from, to, to, to, to, text,
	)
}
