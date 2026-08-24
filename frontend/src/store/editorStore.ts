import { create } from "zustand";
import type { Track } from "../api/types";

interface EditorState {
  track: Track | null;
  setTrack: (track: Track | null) => void;
  /** Optimistically patch one line's fields in local state (before/without waiting for the PUT response). */
  patchLine: (lineId: number, patch: Partial<Track["lines"][number]>) => void;
  /**
   * Whether the editor (and LRC export) shows/edits `romanized` instead of
   * `original` as the main lyric text. Flips to true automatically once
   * Romanize succeeds (see RomanizeButton.tsx); the user can flip it back.
   * Resets whenever a track is (re)loaded.
   */
  showRomanized: boolean;
  setShowRomanized: (show: boolean) => void;
}

export const useEditorStore = create<EditorState>((set) => ({
  track: null,
  setTrack: (track) => set({ track, showRomanized: false }),
  patchLine: (lineId, patch) =>
    set((state) => {
      if (!state.track) return state;
      return {
        track: {
          ...state.track,
          lines: state.track.lines.map((l) => (l.id === lineId ? { ...l, ...patch } : l)),
        },
      };
    }),
  showRomanized: false,
  setShowRomanized: (show) => set({ showRomanized: show }),
}));
