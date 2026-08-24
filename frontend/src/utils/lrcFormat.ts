import type { Track } from "../api/types";

export interface LrcExportOptions {
  /** When true, emit a second line at the same timestamp with the translation. */
  includeTranslation: boolean;
  /** Wrap the translation line in "( )" — only relevant when includeTranslation is true. */
  wrapTranslationInParens: boolean;
}

// includeTranslation defaults to true: the whole point of this app is
// ending up with a translated LRC, so that shouldn't be something the user
// has to dig into an options panel to switch on (per user feedback).
export const DEFAULT_LRC_EXPORT_OPTIONS: LrcExportOptions = {
  includeTranslation: true,
  wrapTranslationInParens: true,
};

/**
 * Builds plain LRC text from a track's lines, purely client-side. This is
 * what makes "skip terjemahan -> copy/download langsung" instant with no
 * backend round-trip (see plan-extended.md M1 + "Export" note).
 *
 * Mirrors the format backend/internal/lrc/lrc.go parses: "[mm:ss.xx]text"
 * per line, sorted by time. The main lyric line is WYSIWYG with the editor:
 * `showRomanized` is the same store flag LineRow.tsx uses to switch between
 * original and romanized text (see store/editorStore.ts), so export always
 * matches whatever's currently shown/edited on screen — never both at once.
 * When `includeTranslation` is on, the translation is emitted as a second
 * line sharing the same timestamp (enhanced-LRC style, understood by
 * players like Musixmatch) right below the lyric line.
 */
export function trackToLrcText(
  track: Track,
  showRomanized: boolean,
  options: LrcExportOptions = DEFAULT_LRC_EXPORT_OPTIONS,
): string {
  const header = [`[ti:${track.title}]`, `[ar:${track.artist}]`];
  if (track.album) header.push(`[al:${track.album}]`);

  const body: string[] = [];
  for (const line of [...track.lines].sort((a, b) => a.time_ms - b.time_ms)) {
    const lyric = showRomanized && line.romanized ? line.romanized : line.original;
    body.push(`${line.timestamp}${lyric}`);

    if (options.includeTranslation && line.translation) {
      const text = options.wrapTranslationInParens ? `(${line.translation})` : line.translation;
      body.push(`${line.timestamp}${text}`);
    }
  }

  return [...header, "", ...body].join("\n") + "\n";
}

export function lrcFileName(track: Track): string {
  const safe = (s: string) => s.replace(/[\\/:*?"<>|]/g, "").trim();
  return `${safe(track.artist)} - ${safe(track.title)}.lrc`;
}
