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
// Milestone 1 only wires up manual edits to the original lyric text and its
// timestamp; translation editing (with suggested_* snapshotting) lands in M2.
type UpdateLineRequest struct {
	Original  *string `json:"original"`
	Timestamp *string `json:"timestamp"` // "[mm:ss.xx]"
	TimeMs    *int64  `json:"time_ms"`
}
