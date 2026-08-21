// Package align implements the alignment heuristic for Cabang C (scraping):
// mapping a loose, unstructured block of scraped translation text onto the
// fixed, timestamp-ordered lines of a track's original lyrics.
//
// This is a pure function package (no I/O) by design — see plan-extended.md
// milestone M3 — so the heuristic can be unit tested without any network or
// scraping involved.
package align

import (
	"regexp"
	"strings"
)

// annotationRe matches a line that is purely a structural marker, e.g.
// "[Chorus]", "(Verse 1)", "[x2]" — these carry no translatable content and
// are dropped before alignment.
var annotationRe = regexp.MustCompile(`^\s*[\[(].*[\])]\s*$`)

// Align maps cleaned-up scraped lines onto original line positions.
//
// original is the track's synced lines in order (an empty string marks an
// instrumental/gap line, same convention LRC uses). scrapedRaw is the raw
// text pulled from a web page, split into lines; blank lines are treated as
// block separators (e.g. verse/chorus breaks) and annotation-only lines
// (like "[Chorus]") are discarded.
//
// Strategy, in order of preference:
//  1. If the cleaned scraped line count matches len(original) exactly,
//     map 1:1 by position.
//  2. Else, if blank-line blocks can be split out on both sides and the
//     block counts match, align block-by-block, then positionally within
//     each block.
//  3. Else, fall back to a proportional index mapping.
//
// The result is a slice parallel to original: result[i] is the aligned
// translation for original[i], or "" if nothing was mapped to it. Every
// non-empty result should be treated as unverified — plan.md requires
// Cabang C output to always be flagged needs_review, regardless of which
// strategy produced it.
func Align(original []string, scrapedRaw []string) []string {
	result := make([]string, len(original))
	if len(original) == 0 {
		return result
	}

	scrapedBlocks := splitBlocks(scrapedRaw)
	cleanedFlat := flatten(scrapedBlocks)

	originalBlocks := splitBlocksKeepEmpty(original)

	switch {
	case len(cleanedFlat) == len(original):
		copy(result, cleanedFlat)

	case len(scrapedBlocks) == len(originalBlocks) && len(scrapedBlocks) > 0:
		alignByBlock(result, originalBlocks, scrapedBlocks)

	default:
		alignProportional(result, original, cleanedFlat)
	}

	return result
}

// splitBlocks splits raw scraped text into blank-line-delimited blocks,
// dropping empty lines and annotation-only lines within each block.
func splitBlocks(raw []string) [][]string {
	var blocks [][]string
	var current []string

	flush := func() {
		if len(current) > 0 {
			blocks = append(blocks, current)
			current = nil
		}
	}

	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			flush()
			continue
		}
		if annotationRe.MatchString(trimmed) {
			continue
		}
		current = append(current, trimmed)
	}
	flush()

	return blocks
}

// splitBlocksKeepEmpty splits original lines into blocks separated by empty
// strings (LRC's convention for instrumental gaps), preserving each
// original line's index in indexedLine so alignByBlock can write results
// back to the right position.
type indexedLine struct {
	index int
	text  string
}

func splitBlocksKeepEmpty(original []string) [][]indexedLine {
	var blocks [][]indexedLine
	var current []indexedLine

	flush := func() {
		if len(current) > 0 {
			blocks = append(blocks, current)
			current = nil
		}
	}

	for i, line := range original {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current = append(current, indexedLine{index: i, text: line})
	}
	flush()

	return blocks
}

func flatten(blocks [][]string) []string {
	var out []string
	for _, b := range blocks {
		out = append(out, b...)
	}
	return out
}

func alignByBlock(result []string, originalBlocks [][]indexedLine, scrapedBlocks [][]string) {
	for bi, origBlock := range originalBlocks {
		scrapedBlock := scrapedBlocks[bi]
		for i, ol := range origBlock {
			var text string
			switch {
			case i < len(scrapedBlock):
				text = scrapedBlock[i]
			case len(scrapedBlock) > 0:
				// Block sizes differ within a matched pair of blocks — fall
				// back to proportional mapping inside this block rather
				// than leaving trailing lines empty.
				idx := i * len(scrapedBlock) / len(origBlock)
				if idx >= len(scrapedBlock) {
					idx = len(scrapedBlock) - 1
				}
				text = scrapedBlock[idx]
			}
			result[ol.index] = text
		}
	}
}

func alignProportional(result []string, original []string, cleanedFlat []string) {
	if len(cleanedFlat) == 0 {
		return
	}

	nonEmptyCount := 0
	for _, l := range original {
		if strings.TrimSpace(l) != "" {
			nonEmptyCount++
		}
	}
	if nonEmptyCount == 0 {
		return
	}

	seen := 0
	for i, l := range original {
		if strings.TrimSpace(l) == "" {
			continue
		}
		idx := seen * len(cleanedFlat) / nonEmptyCount
		if idx >= len(cleanedFlat) {
			idx = len(cleanedFlat) - 1
		}
		result[i] = cleanedFlat[idx]
		seen++
	}
}
