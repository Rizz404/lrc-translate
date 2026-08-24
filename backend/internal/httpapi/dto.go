package httpapi

// SearchResultDTO is one candidate returned by GET /api/search.
type SearchResultDTO struct {
	LrclibID        int64  `json:"lrclib_id"`
	Title           string `json:"title"`
	Artist          string `json:"artist"`
	Album           string `json:"album"`
	DurationMs      int64  `json:"duration_ms"`
	Instrumental    bool   `json:"instrumental"`
	HasSyncedLyrics bool   `json:"has_synced_lyrics"`
}

// ImportTrackRequest is the body of POST /api/tracks/import.
type ImportTrackRequest struct {
	LrclibID int64 `json:"lrclib_id" binding:"required"`
}

// LineDTO is one synced lyric line as exposed by the API.
type LineDTO struct {
	ID        uint   `json:"id"`
	LineIndex int    `json:"line_index"`
	TimeMs    int64  `json:"time_ms"`
	Timestamp string `json:"timestamp"`
	Original  string `json:"original"`
	Romanized string `json:"romanized"`
	// RomanizedSource is "internal" | "scrape" | "" (never romanized) — see
	// db.Line.RomanizedSource.
	RomanizedSource string `json:"romanized_source,omitempty"`
	Translation     string `json:"translation"`
	// TranslationLang is Translation's language code (e.g. "en"), set only
	// when Translation came from a scrape source — see db.Line.TranslationLang.
	TranslationLang string `json:"translation_lang,omitempty"`
	Method          string `json:"method"`
	NeedsReview     bool   `json:"needs_review"`
}

// TrackDTO is a full track with its lines, as returned by GET /api/tracks/:id
// and POST /api/tracks/import.
type TrackDTO struct {
	ID         string    `json:"id"`
	LrclibID   string    `json:"lrclib_id,omitempty"`
	Title      string    `json:"title"`
	Artist     string    `json:"artist"`
	Album      string    `json:"album"`
	DurationMs int64     `json:"duration_ms"`
	Language   string    `json:"language"`
	Source     string    `json:"source"`
	Lines      []LineDTO `json:"lines"`
}

// TrackSummaryDTO is a track without its lines, used for list views.
type TrackSummaryDTO struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	DurationMs int64  `json:"duration_ms"`
	Language   string `json:"language"`
	Source     string `json:"source"`
}

// UpdateTrackRequest is the body of PUT /api/tracks/:id. All fields optional
// (partial update) — nil pointer means "leave unchanged".
type UpdateTrackRequest struct {
	Title    *string `json:"title"`
	Artist   *string `json:"artist"`
	Album    *string `json:"album"`
	Language *string `json:"language"`
}

// UpdateLineRequest is the body of PUT /api/tracks/:id/lines/:lineId.
// Editing Translation snapshots the line's current Translation/Method into
// SuggestedTranslation/SuggestedMethod (unless it's already "manual", so a
// second manual edit doesn't overwrite the original auto-generated
// suggestion) and sets Method to "manual" — see handleUpdateLine. Editing
// Romanized sets RomanizedSource to "manual", protecting it from later
// being overwritten by a bulk Romanize the same way a "scrape" source is.
type UpdateLineRequest struct {
	Original    *string `json:"original"`
	Romanized   *string `json:"romanized"`
	Timestamp   *string `json:"timestamp"` // "[mm:ss.xx]"
	TimeMs      *int64  `json:"time_ms"`
	Translation *string `json:"translation"`
}

// TranslateSource selects what text handleTranslateTrack feeds to
// LibreTranslate as the source of each line.
type TranslateSource string

const (
	// TranslateSourceOriginal translates from Line.Original (the original
	// lyric text), same as before this field existed — the default.
	TranslateSourceOriginal TranslateSource = "original"
	// TranslateSourceScrape translates from Line.Translation on lines whose
	// Method is "scrape" (chaining off an already-scraped translation, e.g.
	// EN from utatime.com, instead of the original lyric — see
	// plan-extended.md). Lines without scrape data fall back to
	// TranslateSourceOriginal so the batch still completes for them.
	TranslateSourceScrape TranslateSource = "scrape"
)

// TranslateRequest is the body of POST /api/tracks/:id/translate.
type TranslateRequest struct {
	TargetLang string `json:"target_lang" binding:"required"`
	LineIDs    []uint `json:"line_ids"` // empty = translate all lines
	// Source picks what text to translate from — see TranslateSource.
	// Defaults to TranslateSourceOriginal when empty.
	Source TranslateSource `json:"source"`
}

// TranslateResponse reports the outcome of a translate batch.
type TranslateResponse struct {
	Lines       []LineDTO `json:"lines"`
	CacheHits   int       `json:"cache_hits"`
	CacheMisses int       `json:"cache_misses"`
	Failed      []struct {
		LineID uint   `json:"line_id"`
		Error  string `json:"error"`
	} `json:"failed,omitempty"`
}

// ClearTranslationRequest is the body of POST /api/tracks/:id/translate/clear.
type ClearTranslationRequest struct {
	LineIDs []uint `json:"line_ids"` // empty = clear all lines
}

// ClearTranslationResponse reports every line actually cleared (lines that
// were already empty/method "none" are left out).
type ClearTranslationResponse struct {
	Lines []LineDTO `json:"lines"`
}

// RomanizeResponse reports the outcome of a romanize batch. SkippedCount
// counts lines left untouched because they already carry a scrape-sourced or
// manually-edited romanization (see db.Line.RomanizedSource) — those are
// trusted over this app's internal kagome/gojp-kana pipeline, so they're
// never overwritten by a bulk Romanize.
type RomanizeResponse struct {
	Lines        []LineDTO `json:"lines"`
	SkippedCount int       `json:"skipped_count,omitempty"`
}

// ScrapeTrackRequest is the body of POST /api/tracks/:id/scrape.
// SourceURL is optional: leave it empty to have the backend try guessing a
// utatime.com URL from the track's title/artist first (see
// internal/scrape/utatime.go); if that guess fails, or for any other site,
// pass the page URL the user found themselves.
type ScrapeTrackRequest struct {
	SourceURL string `json:"source_url"`
}

// ScrapeTrackResponse is stage 1 of Cabang C: raw extracted text, not yet
// aligned to any line. ResolvedURL is always populated — it's either the
// SourceURL the caller passed, or the URL auto-discovery guessed and
// successfully fetched.
type ScrapeTrackResponse struct {
	ScrapeSourceID uint   `json:"scrape_source_id"`
	ResolvedURL    string `json:"resolved_url"`
	RawText        string `json:"raw_text"`
	// Language is RawText's language code, auto-detected only for
	// utatime.com (from its picked translation subtab); empty for any other
	// site — the frontend should then prompt the user to fill it in (see
	// AlignTrackRequest.Language) before relying on it for the
	// same-language checks in handleTranslateTrack.
	Language       string `json:"language,omitempty"`
	RawRomanized   string `json:"raw_romanized,omitempty"` // only present for sources that provide one, e.g. utatime.com
	AutoDiscovered bool   `json:"auto_discovered"`
}

// AlignTrackRequest is the body of POST /api/tracks/:id/align. Language is
// optional and only meant to fill in a scrape source that came back with an
// empty ScrapeTrackResponse.Language (i.e. every non-utatime.com source) —
// when set, it overwrites the stored ScrapeSource.Language before aligning.
type AlignTrackRequest struct {
	ScrapeSourceID uint   `json:"scrape_source_id" binding:"required"`
	Language       string `json:"language,omitempty"`
}

// AlignTrackResponse is stage 2 of Cabang C: lines updated with the
// heuristically-aligned translation, all flagged needs_review.
type AlignTrackResponse struct {
	Lines []LineDTO `json:"lines"`
}

// ResetTrackResponse reports every line after POST /api/tracks/:id/reset
// wipes translation/romanization/method progress back to pristine (just the
// imported original text + timestamps, same as right after import).
type ResetTrackResponse struct {
	Lines []LineDTO `json:"lines"`
}
