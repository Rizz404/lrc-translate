// Package llmprompt builds the song-lyric translation prompt shared by every
// LLM-based MT backend (internal/gemini, internal/localllm). Unlike plain
// NMT (internal/libretranslate), an LLM can be steered by instructions to
// translate like a song lyric — natural/idiomatic, informal register, full
// coverage of the line — instead of a stiff word-for-word mapping. The
// instructions themselves don't depend on which LLM API is calling them, so
// they live here once instead of being duplicated per backend.
package llmprompt

import (
	"encoding/json"
	"fmt"
	"strings"
)

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

// BuildBatch returns the prompt text for translating an entire song's
// targeted lines together, in one request, from sourceLang to targetLang.
// Unlike Build (one line per independent request — see its doc comment for
// the consistency problem that caused), this hands the model every line at
// once so it can use the whole song as context: keeping pronouns, tone, and
// repeated phrasing/hooks consistent across lines the way a human
// translator working on the whole song would, instead of each line being
// guessed at blind to its neighbors.
//
// The response is expected to be a JSON array of exactly len(lines)
// strings, in the same order as lines — see ParseBatch, which decodes and
// validates that shape.
func BuildBatch(lines []string, sourceLang, targetLang string) string {
	var from string
	if sourceLang == "" || sourceLang == "auto" {
		from = "the source language (detect it automatically)"
	} else {
		from = langName(sourceLang)
	}
	to := langName(targetLang)

	var numbered strings.Builder
	for i, line := range lines {
		fmt.Fprintf(&numbered, "%d. %s\n", i+1, line)
	}

	return fmt.Sprintf(
		"You are translating an entire song's lyrics from %s to %s. All %d "+
			"lines are given together below, in order, so you can use the "+
			"whole song as context.\n\n"+
			"Rules:\n"+
			"1. Register: song lyrics address people informally. Use the "+
			"casual/informal form of \"I\" and \"you\" in %s, never a formal or "+
			"polite register (e.g. in Indonesian use \"aku\"/\"kamu\", never "+
			"\"saya\"/\"Anda\"; in French use \"tu\", never \"vous\") — unless "+
			"the song is itself unmistakably formal or reverent in tone. Never "+
			"mix formal and informal address across lines.\n"+
			"2. Translate every line FULLY. Every word must end up in %s. Never "+
			"leave part or all of a line untranslated, and never output the "+
			"source text unchanged — the only exception is a proper noun (a "+
			"name, a place) with no natural equivalent.\n"+
			"3. Prioritize meaning over literal wording: render what each line "+
			"actually means and how a native %s speaker would naturally say it "+
			"in a song, not a stiff word-for-word mapping of the source "+
			"grammar.\n"+
			"4. Keep each line roughly as concise as its original, so it still "+
			"reads like a lyric.\n"+
			"5. Use the surrounding lines as context: keep the same pronouns, "+
			"tone, and wording for repeated lines/hooks consistent across the "+
			"whole song.\n"+
			"6. Return exactly %d lines, in the same order as given — never "+
			"merge, split, skip, or add lines.\n\n"+
			"Output ONLY a JSON array of exactly %d strings (one translated "+
			"line per array element, same order as below) — no numbering, no "+
			"markdown code fence, no explanation, nothing before or after the "+
			"array.\n\nLines:\n%s",
		from, to, len(lines), to, to, to, len(lines), len(lines), numbered.String(),
	)
}

// ParseBatch decodes the JSON array of translated lines an LLM returned for
// a BuildBatch prompt, tolerating a markdown code fence around it (some
// backends/chat templates wrap an answer in ```json ... ``` even when told
// not to). want must match the array length exactly — a mismatch means the
// model dropped, merged, or added a line, and there's no safe way to guess
// which output goes with which input, so that's reported as an error rather
// than silently misaligning the rest of the song.
func ParseBatch(raw string, want int) ([]string, error) {
	s := strings.TrimSpace(raw)
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("llmprompt: decode batch response as JSON array: %w", err)
	}
	if len(out) != want {
		return nil, fmt.Errorf("llmprompt: expected %d translated lines back, got %d", want, len(out))
	}
	return out, nil
}
