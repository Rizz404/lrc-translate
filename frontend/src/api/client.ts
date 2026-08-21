import type {
  AlignTrackResponse,
  Line,
  RomanizeResponse,
  ScrapeTrackResponse,
  SearchResult,
  Track,
  TrackSummary,
  TranslateRequest,
  TranslateResponse,
  UpdateLineRequest,
} from "./types";

const BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api";

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

  translateTrack(trackId: string, body: TranslateRequest): Promise<TranslateResponse> {
    return request(`/tracks/${trackId}/translate`, {
      method: "POST",
      body: JSON.stringify(body),
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

  alignTrack(trackId: string, scrapeSourceId: number): Promise<AlignTrackResponse> {
    return request(`/tracks/${trackId}/align`, {
      method: "POST",
      body: JSON.stringify({ scrape_source_id: scrapeSourceId }),
    });
  },
};
