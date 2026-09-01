// Mirrors the JSON shapes returned by the Go backend (see
// backend/internal/httpapi/dto.go). Maintained by hand since frontend and
// backend are separate projects (npm vs Go module) — see plan-extended.md
// "Catatan penting soal shared types".

export type Method = "none" | "mt" | "ai" | "scrape" | "manual";
export type TrackSource = "lrclib" | "manual";

export interface SearchResult {
  lrclib_id: number;
  title: string;
  artist: string;
  album: string;
  duration_ms: number;
  instrumental: boolean;
  has_synced_lyrics: boolean;
}

/**
 * One field's (translation's or romanized's) raw scraped match plus its
 * immediate neighbors in the scraped source text, taken at align time.
 * Prev/next are "" when there's no neighbor on that side. See
 * backend/internal/align.Context — none of Align's heuristic strategies are
 * verified correct, so this lets the editor show what the scrape source
 * actually said around this point, for a quick sanity check against a
 * possible bad alignment guess.
 */
export interface LineScrapeContext {
  prev?: string;
  matched?: string;
  next?: string;
}

/** Bundles LineScrapeContext for both fields Align can independently populate on a line. */
export interface LineScrapeContexts {
  translation?: LineScrapeContext;
  romanized?: LineScrapeContext;
  /**
   * Machine-translated reference for this line's original text, requested
   * on demand via api.aiReference — always correctly positioned (a direct
   * MT call per line, not an alignment guess), so it's useful as a
   * same-language sanity check on `translation` even for a reviewer who
   * can't read the original lyric's language at all.
   */
  ai?: string;
}

export interface Line {
  id: number;
  line_index: number;
  time_ms: number;
  timestamp: string;
  original: string;
  romanized: string;
  /** "internal" (kagome/gojp-kana) | "scrape" (e.g. utatime.com) | "manual" | "" (never romanized). */
  romanized_source: string;
  translation: string;
  /** Translation's language code (e.g. "en"), only set when translation came from a scrape source. */
  translation_lang?: string;
  method: Method;
  needs_review: boolean;
  /** Raw scraped neighborhood this line's scrape-derived text was matched from — undefined until scrape+align has touched this line at least once. */
  scrape_context?: LineScrapeContexts;
}

export interface Track {
  id: string;
  lrclib_id?: string;
  title: string;
  artist: string;
  album: string;
  duration_ms: number;
  language: string;
  source: TrackSource;
  lines: Line[];
}

export interface TrackSummary {
  id: string;
  title: string;
  artist: string;
  album: string;
  duration_ms: number;
  language: string;
  source: TrackSource;
}

export interface UpdateLineRequest {
  original?: string;
  romanized?: string;
  timestamp?: string;
  time_ms?: number;
  translation?: string;
}

/** "original" = translate from the original lyric (default); "scrape" = chain off an already-scraped translation instead (see TranslatePanel.tsx). */
export type TranslateSource = "original" | "scrape";

export interface TranslateRequest {
  target_lang: string;
  line_ids?: number[];
  source?: TranslateSource;
}

export interface TranslateResponse {
  lines: Line[];
  failed?: { line_id: number; error: string }[];
}

export interface ClearTranslationRequest {
  line_ids?: number[];
}

export interface ClearTranslationResponse {
  lines: Line[];
}

export interface RomanizeResponse {
  lines: Line[];
  /** Lines left untouched because they already had a scrape-sourced or manually-edited romanization. */
  skipped_count?: number;
}

export interface ScrapeTrackResponse {
  scrape_source_id: number;
  resolved_url: string;
  raw_text: string;
  /** raw_text's language code, auto-detected only for utatime.com — empty for any other site. */
  language?: string;
  raw_romanized?: string;
  auto_discovered: boolean;
}

export interface AlignTrackResponse {
  lines: Line[];
}

export interface AIReferenceRequest {
  target_lang: string;
  line_ids?: number[];
}

export interface AIReferenceResponse {
  lines: Line[];
  cache_hits: number;
  cache_misses: number;
  failed?: { line_id: number; error: string }[];
}

export interface ResetTrackResponse {
  lines: Line[];
}
