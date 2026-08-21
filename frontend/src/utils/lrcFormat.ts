import type { Track } from "../api/types";

/**
 * Builds plain LRC text from a track's lines, purely client-side. This is
 * what makes "skip terjemahan -> copy/download langsung" instant with no
 * backend round-trip (see plan-extended.md M1 + "Export" note).
 *
 * Mirrors the format backend/internal/lrc/lrc.go parses: "[mm:ss.xx]text"
 * per line, sorted by time.
 */
export function trackToLrcText(track: Track): string {
  const header = [`[ti:${track.title}]`, `[ar:${track.artist}]`];
  if (track.album) header.push(`[al:${track.album}]`);

  const body = [...track.lines]
    .sort((a, b) => a.time_ms - b.time_ms)
    .map((line) => `${line.timestamp}${line.original}`);

  return [...header, "", ...body].join("\n") + "\n";
}

export function lrcFileName(track: Track): string {
  const safe = (s: string) => s.replace(/[\\/:*?"<>|]/g, "").trim();
  return `${safe(track.artist)} - ${safe(track.title)}.lrc`;
}
