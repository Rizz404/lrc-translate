import { create } from "zustand";
import type { Track } from "../api/types";

interface EditorState {
  track: Track | null;
  setTrack: (track: Track | null) => void;
  /** Optimistically patch one line's fields in local state (before/without waiting for the PUT response). */
  patchLine: (lineId: number, patch: Partial<Track["lines"][number]>) => void;
}

export const useEditorStore = create<EditorState>((set) => ({
  track: null,
  setTrack: (track) => set({ track }),
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
}));
