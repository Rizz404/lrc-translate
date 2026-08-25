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
//  0. If scrapedRaw, with only annotation-only lines dropped (blank lines
//     kept in place), is exactly as long as original AND its blank-line
//     positions line up with original's instrumental gaps, the source is
//     already position-aligned line-for-line — map 1:1 by index directly,
//     blanks included. This is the common case for sources that render one
//     entry per original line themselves (e.g. utatime.com's ".line-text"
//     spans, see internal/scrape/utatime.go) — trusting that alignment
//     beats guessing at it via blocks.
//  1. Else, if the cleaned (blank-stripped) scraped line count matches
//     len(original) exactly, map 1:1 by position.
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
	result, _ := AlignWithContext(original, scrapedRaw)
	return result
}

// Context is the raw scraped line a Context result was actually read from
// (Matched), plus its immediate neighbors in the scraped source (Prev/Next,
// empty when there is none). It exists so a caller — the editor UI, in
// practice — can show a human "here's what the scrape source said around
// this point" next to a heuristically-aligned line: since none of Align's
// strategies beyond #0/#1 are verified to be correct (see plan.md's
// needs_review requirement), surfacing the raw neighborhood lets a person
// spot a bad guess (e.g. two adjacent original lines mapped to the same
// scraped line — a merge Align had no way to detect) at a glance instead of
// having to hunt through the full raw text by hand.
type Context struct {
	Prev    string
	Matched string
	Next    string
}

// AlignWithContext does the same mapping as Align, but also returns a
// Context per original position — see Context's doc comment. Both slices
// are parallel to original; a position with nothing mapped to it (e.g. an
// instrumental gap, or scrapedRaw being too short to cover it) gets a
// zero-value Context.
func AlignWithContext(original []string, scrapedRaw []string) ([]string, []Context) {
	result := make([]string, len(original))
	contexts := make([]Context, len(original))
	if len(original) == 0 {
		return result, contexts
	}

	scrapedPositional := trimAnnotationsKeepBlanks(scrapedRaw)

	scrapedBlocks := splitBlocks(scrapedRaw)
	cleanedFlat := flatten(scrapedBlocks)

	originalBlocks := splitBlocksKeepEmpty(original)

	switch {
	case blanksLineUp(original, scrapedPositional):
		copy(result, scrapedPositional)
		fillContext(contexts, scrapedPositional, identityIndices(original))

	case len(cleanedFlat) == len(original):
		copy(result, cleanedFlat)
		fillContext(contexts, cleanedFlat, identityIndices(original))

	case len(scrapedBlocks) == len(originalBlocks) && len(scrapedBlocks) > 0:
		idxs := alignByBlock(result, originalBlocks, scrapedBlocks)
		fillContext(contexts, cleanedFlat, idxs)

	default:
		idxs := alignProportional(result, original, cleanedFlat)
		fillContext(contexts, cleanedFlat, idxs)
	}

	return result, contexts
}

// identityIndices returns, for each position in original, that same
// position's index into a same-length, position-aligned scraped slice —
// or -1 for a blank (instrumental gap) position, which never gets a
// Context. Used by AlignWithContext's strategies 0 and 1, where the
// mapping is a direct 1:1 copy by index.
func identityIndices(original []string) []int {
	idxs := make([]int, len(original))
	for i, l := range original {
		if strings.TrimSpace(l) == "" {
			idxs[i] = -1
		} else {
			idxs[i] = i
		}
	}
	return idxs
}

// fillContext writes contexts[i] from flat[idxs[i]] plus its immediate
// neighbors, for every i where idxs[i] is a valid index into flat.
func fillContext(contexts []Context, flat []string, idxs []int) {
	for i, idx := range idxs {
		if idx < 0 || idx >= len(flat) {
			continue
		}
		ctx := Context{Matched: flat[idx]}
		if idx > 0 {
			ctx.Prev = flat[idx-1]
		}
		if idx+1 < len(flat) {
			ctx.Next = flat[idx+1]
		}
		contexts[i] = ctx
	}
}

// trimAnnotationsKeepBlanks drops annotation-only lines (see annotationRe)
// but, unlike splitBlocks, keeps blank lines and their positions rather
// than treating them purely as block separators to discard. That's what
// lets the positional strategy in Align compare blank-line positions
// against original directly.
func trimAnnotationsKeepBlanks(raw []string) []string {
	var out []string
	for _, line := range raw {
		trimmed := strings.TrimSpace(line)
		if annotationRe.MatchString(trimmed) {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

// blanksLineUp reports whether a and b are the same length and have a
// blank line (after trimming) at exactly the same positions — the signal
// that b is a genuine position-for-position match with a, not merely a
// same-length coincidence that would corrupt the 1:1 copy in Align.
func blanksLineUp(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if (strings.TrimSpace(a[i]) == "") != (strings.TrimSpace(b[i]) == "") {
			return false
		}
	}
	return true
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

// alignByBlock fills result and returns, per original position, that
// position's index into flatten(scrapedBlocks) (i.e. cleanedFlat) — or -1
// for a position nothing was mapped to. scrapedBlocks are consumed in
// order, so a running offset (the total length of all prior blocks) turns
// each block-local index into that same global index.
func alignByBlock(result []string, originalBlocks [][]indexedLine, scrapedBlocks [][]string) []int {
	idxs := make([]int, len(result))
	for i := range idxs {
		idxs[i] = -1
	}

	offset := 0
	for bi, origBlock := range originalBlocks {
		scrapedBlock := scrapedBlocks[bi]
		if len(scrapedBlock) == 0 {
			offset += len(scrapedBlock)
			continue
		}
		for i, ol := range origBlock {
			var local int
			switch {
			case len(scrapedBlock) == len(origBlock):
				// Same size on both sides within this block — exact 1:1,
				// no rounding involved.
				local = i
			default:
				// Block sizes differ within a matched pair of blocks (e.g.
				// a scraped translation merges two original lines into
				// one). Map every position in the block proportionally —
				// not just the positions past len(scrapedBlock) — so a
				// merge near the start of the block doesn't leave the
				// rest of the block copied one index off from where it
				// should be (which both duplicates a line and lets it
				// bleed into a position that should hold different
				// content; see align_test.go's regression case).
				local = proportionalIndex(i, len(origBlock), len(scrapedBlock))
			}
			result[ol.index] = scrapedBlock[local]
			idxs[ol.index] = offset + local
		}
		offset += len(scrapedBlock)
	}

	return idxs
}

// alignProportional fills result and returns, per original position, that
// position's index into cleanedFlat — or -1 for a blank/unmapped position.
func alignProportional(result []string, original []string, cleanedFlat []string) []int {
	idxs := make([]int, len(result))
	for i := range idxs {
		idxs[i] = -1
	}
	if len(cleanedFlat) == 0 {
		return idxs
	}

	nonEmptyCount := 0
	for _, l := range original {
		if strings.TrimSpace(l) != "" {
			nonEmptyCount++
		}
	}
	if nonEmptyCount == 0 {
		return idxs
	}

	seen := 0
	for i, l := range original {
		if strings.TrimSpace(l) == "" {
			continue
		}
		idx := proportionalIndex(seen, nonEmptyCount, len(cleanedFlat))
		result[i] = cleanedFlat[idx]
		idxs[i] = idx
		seen++
	}

	return idxs
}

// proportionalIndex maps position pos (0-based, out of fromCount total
// positions) onto a 0-based index into a range of toCount items, rounding
// to the nearest index instead of flooring.
//
// Flooring (pos*toCount/fromCount) systematically bunches duplicate/skipped
// mappings at the start of the range whenever toCount < fromCount (integer
// division delays the first increment) — e.g. mapping 61 original lines
// onto 56 scraped lines would floor-map both position 0 and 1 to scraped
// index 0 before ever advancing, even though the "fair" stretch is closer
// to one duplicate every ~11 lines. Rounding to the nearest index spreads
// that unavoidable compression evenly across the whole line instead.
func proportionalIndex(pos, fromCount, toCount int) int {
	idx := (pos*toCount + fromCount/2) / fromCount
	if idx >= toCount {
		idx = toCount - 1
	}
	return idx
}
