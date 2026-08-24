import type {
  AlignTrackResponse,
  ClearTranslationResponse,
  Line,
  ResetTrackResponse,
  RomanizeResponse,
  ScrapeTrackResponse,
  SearchResult,
  Track,
  TrackSummary,
  TranslateRequest,
  TranslateResponse,
  UpdateLineRequest,
} from "./types";

// Relative by default so requests go through Vite's dev proxy (see vite.config.ts) —
// this keeps everything on one origin/port, handy when tunneling (ngrok, etc.).
const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api";

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });

  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new Error(body.error ?? `Request failed with status ${res.status}`);
  }

  if (res.status === 204) {
    return undefined as T;
  }
  return res.json() as Promise<T>;
}

export const api = {
  search(title: string): Promise<SearchResult[]> {
    const params = new URLSearchParams({ title });
    return request(`/search?${params.toString()}`);
  },

  importTrack(lrclibId: number): Promise<Track> {
    return request("/tracks/import", {
      method: "POST",
      body: JSON.stringify({ lrclib_id: lrclibId }),
    });
  },

  listTracks(): Promise<TrackSummary[]> {
    return request("/tracks");
  },

  getTrack(id: string): Promise<Track> {
    return request(`/tracks/${id}`);
  },

  deleteTrack(id: string): Promise<void> {
    return request(`/tracks/${id}`, { method: "DELETE" });
  },

  updateLine(trackId: string, lineId: number, body: UpdateLineRequest): Promise<Line> {
    return request(`/tracks/${trackId}/lines/${lineId}`, {
      method: "PUT",
      body: JSON.stringify(body),
    });
  },

  revertLine(trackId: string, lineId: number): Promise<Line> {
    return request(`/tracks/${trackId}/lines/${lineId}/revert`, { method: "POST" });
  },

  /** Wipes translation/romanization/method progress on every line back to pristine. */
  resetTrack(trackId: string): Promise<ResetTrackResponse> {
    return request(`/tracks/${trackId}/reset`, { method: "POST" });
  },

  translateTrack(trackId: string, body: TranslateRequest): Promise<TranslateResponse> {
    return request(`/tracks/${trackId}/translate`, {
      method: "POST",
      body: JSON.stringify(body),
    });
  },

  /** Wipes translation (not lyric/timestamp) on the targeted lines — omit lineIds for all lines. */
  clearTranslation(trackId: string, lineIds?: number[]): Promise<ClearTranslationResponse> {
    return request(`/tracks/${trackId}/translate/clear`, {
      method: "POST",
      body: JSON.stringify({ line_ids: lineIds }),
    });
  },

  romanizeTrack(trackId: string): Promise<RomanizeResponse> {
    return request(`/tracks/${trackId}/romanize`, { method: "POST" });
  },

  /** Omit sourceUrl to try auto-discovering a utatime.com page first. */
  scrapeTrack(trackId: string, sourceUrl?: string): Promise<ScrapeTrackResponse> {
    return request(`/tracks/${trackId}/scrape`, {
      method: "POST",
      body: sourceUrl ? JSON.stringify({ source_url: sourceUrl }) : undefined,
    });
  },

  /** language fills in a scrape source that came back with no auto-detected language (i.e. anything but utatime.com). */
  alignTrack(trackId: string, scrapeSourceId: number, language?: string): Promise<AlignTrackResponse> {
    return request(`/tracks/${trackId}/align`, {
      method: "POST",
      body: JSON.stringify({ scrape_source_id: scrapeSourceId, language }),
    });
  },
};
