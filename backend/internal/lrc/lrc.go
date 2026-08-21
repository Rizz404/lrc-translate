// Package lrc parses and formats LRC (synced lyrics) text.
//
// This is a pure, side-effect-free package by design (see plan-extended.md
// milestone M1) so it can be unit tested without any network or DB access.
package lrc

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Line is one synced lyric line.
type Line struct {
	TimeMs    int64
	Timestamp string // formatted as "[mm:ss.xx]"
	Text      string
}

var timeTagRe = regexp.MustCompile(`\[(\d{1,3}):(\d{2})(?:[.:](\d{1,3}))?\]`)

// Parse reads raw LRC text and returns synced lines ordered by time.
// Metadata tags (e.g. [ti:...], [ar:...], [offset:...]) and blank lines are
// ignored. A line carrying multiple timestamps (e.g. "[00:01.00][00:05.00]
// text") expands into one Line per timestamp.
func Parse(raw string) ([]Line, error) {
	var lines []Line

	for _, rawLine := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		rawLine = strings.TrimRight(rawLine, "\r")
		if strings.TrimSpace(rawLine) == "" {
			continue
		}

		matches := timeTagRe.FindAllStringSubmatchIndex(rawLine, -1)
		if len(matches) == 0 {
			continue // metadata tag or stray text, not a synced line
		}

		// Text is everything after the last timestamp tag.
		lastEnd := matches[len(matches)-1][1]
		text := strings.TrimSpace(rawLine[lastEnd:])

		for _, m := range matches {
			minStr := rawLine[m[2]:m[3]]
			secStr := rawLine[m[4]:m[5]]
			fracStr := ""
			if m[6] != -1 {
				fracStr = rawLine[m[6]:m[7]]
			}

			timeMs, err := toMs(minStr, secStr, fracStr)
			if err != nil {
				return nil, fmt.Errorf("parse timestamp %q: %w", rawLine[m[0]:m[1]], err)
			}

			lines = append(lines, Line{
				TimeMs:    timeMs,
				Timestamp: formatTimestamp(timeMs),
				Text:      text,
			})
		}
	}

	sort.SliceStable(lines, func(i, j int) bool { return lines[i].TimeMs < lines[j].TimeMs })
	return lines, nil
}

// Format renders lines back into LRC text, one "[mm:ss.xx]text" per line,
// sorted by time.
func Format(lines []Line) string {
	sorted := make([]Line, len(lines))
	copy(sorted, lines)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].TimeMs < sorted[j].TimeMs })

	var b strings.Builder
	for _, l := range sorted {
		ts := l.Timestamp
		if ts == "" {
			ts = formatTimestamp(l.TimeMs)
		}
		b.WriteString(ts)
		b.WriteString(l.Text)
		b.WriteByte('\n')
	}
	return b.String()
}

func toMs(minStr, secStr, fracStr string) (int64, error) {
	min, err := strconv.Atoi(minStr)
	if err != nil {
		return 0, err
	}
	sec, err := strconv.Atoi(secStr)
	if err != nil {
		return 0, err
	}

	frac := 0
	if fracStr != "" {
		// Normalize 1-3 digit fractional part to milliseconds (pad/truncate to 3 digits).
		padded := (fracStr + "000")[:3]
		frac, err = strconv.Atoi(padded)
		if err != nil {
			return 0, err
		}
	}

	return int64(min)*60_000 + int64(sec)*1_000 + int64(frac), nil
}

func formatTimestamp(timeMs int64) string {
	if timeMs < 0 {
		timeMs = 0
	}
	totalCentis := timeMs / 10
	min := totalCentis / 100 / 60
	sec := (totalCentis / 100) % 60
	centis := totalCentis % 100
	return fmt.Sprintf("[%02d:%02d.%02d]", min, sec, centis)
}
