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
	MethodNone Method = "none"
	// MethodMT is set by handleTranslateTrack when the active Translator is
	// plain NMT (libretranslate) — literal, not steered by a prompt.
	MethodMT Method = "mt"
	// MethodAI is set by handleTranslateTrack when the active Translator is
	// an LLM (gemini/localllm) — steered by internal/llmprompt to translate
	// like a song lyric rather than word-for-word. Originally reserved for a
	// separate, never-built "Cabang B" endpoint (see plan-extended.md); the
	// LLM-vs-NMT distinction ended up folded into the one /translate
	// endpoint instead (see resolvedIsLLM in cmd/server/main.go), so this is
	// now what actually marks that distinction on a line.
	MethodAI     Method = "ai"
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
	ID           string `gorm:"primaryKey"`
	LrclibID     string
	Title        string
	Artist       string
	Album        string
	DurationMs   int64
	Language     string
	Source       TrackSource
	RawSyncedLrc string `gorm:"type:text"`
	CreatedAt    time.Time
	UpdatedAt    time.Time

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
	// RomanizedSource tracks who produced Romanized ("internal" = kagome/gojp-kana
	// pipeline, "scrape" = pulled from a source like utatime.com via alignment,
	// "manual" = user-edited, "" = never romanized). Lets handleRomanizeTrack
	// avoid clobbering a scrape-sourced or manually-corrected romanization —
	// see httpapi/romanize_handler.go, httpapi/scrape_handler.go, and
	// handleUpdateLine in httpapi/tracks_handler.go.
	RomanizedSource string
	Translation     string `gorm:"type:text"`
	// TranslationLang is the language code of whatever's in Translation right
	// now (e.g. "en"), set when Translation comes from a scrape source (see
	// ScrapeSource.Language and httpapi/scrape_handler.go's handleAlignTrack).
	// Empty for method "mt"/"ai"/"manual"/"none" — those either already
	// target the language the caller asked for, or have no known language.
	// Lets handleTranslateTrack in "translate from scrape data" mode chain
	// MT off of this text/language instead of the original lyric, and refuse
	// a same-language no-op translate (see plan-extended.md).
	TranslationLang string
	Method          Method `gorm:"default:none"`

	// Snapshot of the last auto-generated suggestion, taken right before a
	// manual edit overwrites Translation/Method(/TranslationLang). Powers
	// the "revert to suggestion" feature without a full edit-history table.
	SuggestedTranslation     string `gorm:"type:text"`
	SuggestedMethod          Method
	SuggestedTranslationLang string

	// ScrapeContext is a JSON-encoded snapshot (see httpapi.LineScrapeContextsDTO)
	// of the raw scraped line(s) immediately around wherever Translation
	// and/or Romanized were actually matched from during alignment — taken
	// once, at align time (httpapi.handleAlignTrack), from
	// internal/align.AlignWithContext. Empty until a scrape+align has
	// mapped something onto this line. None of Align's heuristic strategies
	// are verified correct (needs_review is always set alongside this), so
	// the editor UI shows this neighborhood next to the line to make a bad
	// guess — e.g. two original lines both mapped to the same scraped
	// line — visible at a glance instead of requiring a manual diff against
	// the full raw scraped text. Kept as a single JSON text column rather
	// than typed columns, consistent with the portability rule at the top
	// of this file (this one shape, not scalar, doesn't map cleanly to a
	// native column type across dialects anyway).
	ScrapeContext string `gorm:"type:text"`

	NeedsReview bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// TranslationCache memoizes MT/AI results across tracks so identical source
// text (e.g. repeated chorus lines) never needs to hit the external API twice.
type TranslationCache struct {
	ID             uint   `gorm:"primaryKey"`
	CacheKey       string `gorm:"uniqueIndex;not null"` // hash(sourceText+sourceLang+targetLang+provider)
	SourceText     string `gorm:"type:text"`
	TranslatedText string `gorm:"type:text"`
	Provider       string
	CreatedAt      time.Time
}

// ScrapeSource stores raw scraped text for a track, prior to alignment.
// RawText is translation content; RawRomanized is populated only for
// sources that also expose a romanization (currently just utatime.com,
// which tends to be more accurate than our own kagome/gojp-kana pipeline —
// see internal/scrape/utatime.go).
type ScrapeSource struct {
	ID        uint   `gorm:"primaryKey"`
	TrackID   string `gorm:"index;not null"`
	SourceURL string
	RawText   string `gorm:"type:text"`
	// Language is RawText's language code (e.g. "en"). For utatime.com,
	// filled in automatically from the picked translation subtab (see
	// utatimeLangCode in internal/scrape/utatime.go). For any other site
	// there's no reliable way to detect it, so it starts empty and the user
	// fills it in via AlignTrackRequest.Language before/at align time — see
	// handleAlignTrack.
	Language     string
	RawRomanized string `gorm:"type:text"`
	FetchedAt    time.Time
}
