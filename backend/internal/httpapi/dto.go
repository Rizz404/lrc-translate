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
	Translation string `json:"translation"`
	Method      string `json:"method"`
	NeedsReview bool   `json:"needs_review"`
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
// suggestion) and sets Method to "manual" — see handleUpdateLine.
type UpdateLineRequest struct {
	Original    *string `json:"original"`
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

// RomanizeResponse reports the outcome of a romanize batch.
type RomanizeResponse struct {
	Lines []LineDTO `json:"lines"`
}

// ScrapeTrackRequest is the body of POST /api/tracks/:id/scrape.
// SourceURL is a page the *user* found and pasted in — this app does not
// auto-discover or crawl translation sites on its own (see
// internal/scrape/scrape.go for why).
type ScrapeTrackRequest struct {
	SourceURL string `json:"source_url" binding:"required"`
}

// ScrapeTrackResponse is stage 1 of Cabang C: raw extracted text, not yet
// aligned to any line.
type ScrapeTrackResponse struct {
	ScrapeSourceID uint   `json:"scrape_source_id"`
	SourceURL      string `json:"source_url"`
	RawText        string `json:"raw_text"`
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
