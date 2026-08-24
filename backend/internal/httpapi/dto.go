package httpapi

// SearchResultDTO is one candidate returned by GET /api/search.
type SearchResultDTO struct {
	LrclibID         int64   `json:"lrclib_id"`
	Title            string  `json:"title"`
	Artist           string  `json:"artist"`
	Album            string  `json:"album"`
	DurationMs       int64   `json:"duration_ms"`
	Instrumental     bool    `json:"instrumental"`
	HasSyncedLyrics  bool    `json:"has_synced_lyrics"`
}

// ImportTrackRequest is the body of POST /api/tracks/import.
type ImportTrackRequest struct {
	LrclibID int64 `json:"lrclib_id" binding:"required"`
}

// LineDTO is one synced lyric line as exposed by the API.
type LineDTO struct {
	ID          uint   `json:"id"`
	LineIndex   int    `json:"line_index"`
	TimeMs      int64  `json:"time_ms"`
	Timestamp   string `json:"timestamp"`
	Original    string `json:"original"`
	Romanized   string `json:"romanized"`
	// RomanizedSource is "internal" | "scrape" | "manual" | "" (never
	// romanized) — see db.Line.RomanizedSource.
	RomanizedSource string `json:"romanized_source,omitempty"`
	Translation     string `json:"translation"`
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
// Romanized sets RomanizedSource to "manual", which — like "scrape" —
// protects it from being overwritten by a later Romanize run.
type UpdateLineRequest struct {
	Original    *string `json:"original"`
	Romanized   *string `json:"romanized"`
	Timestamp   *string `json:"timestamp"` // "[mm:ss.xx]"
	TimeMs      *int64  `json:"time_ms"`
	Translation *string `json:"translation"`
}

// TranslateRequest is the body of POST /api/tracks/:id/translate.
type TranslateRequest struct {
	TargetLang string `json:"target_lang" binding:"required"`
	LineIDs    []uint `json:"line_ids"` // empty = translate all lines
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

// RomanizeResponse reports the outcome of a romanize batch. SkippedCount
// counts lines left untouched because they already carry a scrape-sourced or
// manually-edited romanization (see db.Line.RomanizedSource) — both more
// trustworthy than this app's internal kagome/gojp-kana pipeline, so neither
// is ever overwritten by it.
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
	RawRomanized   string `json:"raw_romanized,omitempty"` // only present for sources that provide one, e.g. utatime.com
	AutoDiscovered bool   `json:"auto_discovered"`
}

// AlignTrackRequest is the body of POST /api/tracks/:id/align.
type AlignTrackRequest struct {
	ScrapeSourceID uint `json:"scrape_source_id" binding:"required"`
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
