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

export interface Line {
  id: number;
  line_index: number;
  time_ms: number;
  timestamp: string;
  original: string;
  romanized: string;
  translation: string;
  method: Method;
  needs_review: boolean;
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
  timestamp?: string;
  time_ms?: number;
}
