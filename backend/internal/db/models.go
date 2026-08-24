// Package db defines the GORM models and database connection setup.
//
// Portability rules (see plan-extended.md): only plain Go types (string,
// int64, bool, time.Time) are used so GORM maps them to native column types
// per dialect (sqlite/postgres/mysql) without custom `gorm:"type:..."` tags.
// Track.ID is an app-generated UUID string rather than relying on any
// database's autoincrement/rowid semantics.
package db

import "time"

// Method tags who/what produced a line's translation.
type Method string

const (
	MethodNone   Method = "none"
	MethodMT     Method = "mt"
	MethodAI     Method = "ai" // reserved for Cabang B (not implemented yet)
	MethodScrape Method = "scrape"
	MethodManual Method = "manual"
)

// TrackSource identifies where a track's base LRC came from.
type TrackSource string

const (
	SourceLRCLIB TrackSource = "lrclib"
	SourceManual TrackSource = "manual"
)

// Track is a song with synced lyrics.
type Track struct {
	ID            string `gorm:"primaryKey"`
	LrclibID      string
	Title         string
	Artist        string
	Album         string
	DurationMs    int64
	Language      string
	Source        TrackSource
	RawSyncedLrc  string `gorm:"type:text"`
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Lines []Line `gorm:"constraint:OnDelete:CASCADE"`
}

// Line is a single synced lyric line belonging to a Track.
type Line struct {
	ID        uint   `gorm:"primaryKey"`
	TrackID   string `gorm:"index;not null"`
	LineIndex int
	TimeMs    int64
	Timestamp string // "[mm:ss.xx]"
	Original  string `gorm:"type:text"`
	Romanized string `gorm:"type:text"`
	// RomanizedSource tracks who produced Romanized: "internal" (kagome/gojp-kana
	// pipeline), "scrape" (pulled from a source like utatime.com via alignment),
	// "manual" (user-edited while viewing romanized text), or "" (never
	// romanized). Lets handleRomanizeTrack avoid clobbering a scrape-sourced or
	// manually-corrected romanization, both more trustworthy than this app's own
	// pipeline — see httpapi/romanize_handler.go, scrape_handler.go, tracks_handler.go.
	RomanizedSource string
	Translation string `gorm:"type:text"`
	Method    Method `gorm:"default:none"`

	// Snapshot of the last auto-generated suggestion, taken right before a
	// manual edit overwrites Translation/Method. Powers the "revert to
	// suggestion" feature without a full edit-history table.
	SuggestedTranslation string `gorm:"type:text"`
	SuggestedMethod      Method

	NeedsReview bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TranslationCache memoizes MT/AI results across tracks so identical source
// text (e.g. repeated chorus lines) never needs to hit the external API twice.
type TranslationCache struct {
	ID              uint   `gorm:"primaryKey"`
	CacheKey        string `gorm:"uniqueIndex;not null"` // hash(sourceText+sourceLang+targetLang+provider)
	SourceText      string `gorm:"type:text"`
	TranslatedText  string `gorm:"type:text"`
	Provider        string
	CreatedAt       time.Time
}

// ScrapeSource stores raw scraped text for a track, prior to alignment.
// RawText is translation content; RawRomanized is populated only for
// sources that also expose a romanization (currently just utatime.com,
// which tends to be more accurate than our own kagome/gojp-kana pipeline —
// see internal/scrape/utatime.go).
type ScrapeSource struct {
	ID           uint   `gorm:"primaryKey"`
	TrackID      string `gorm:"index;not null"`
	SourceURL    string
	RawText      string `gorm:"type:text"`
	RawRomanized string `gorm:"type:text"`
	FetchedAt    time.Time
}
