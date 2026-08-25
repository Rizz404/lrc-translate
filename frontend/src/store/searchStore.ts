import { create } from "zustand";
import type { SearchResult } from "../api/types";

/**
 * Lifts SearchPage's query/results/scroll out of component state so they
 * survive an unmount — SearchPage fully unmounts whenever the route changes
 * to /track/:id (App.tsx keys its motion.div on location.pathname), so a
 * plain useState there was wiped every time the user went back from the
 * editor: query gone, results gone, scrolled back to the top of the list.
 * Backed by a module-level Zustand store instead of sessionStorage since it
 * only needs to survive within the same SPA session, not a hard refresh.
 */
interface SearchState {
  title: string;
  results: SearchResult[] | null;
  /** window.scrollY at the moment the user navigated away to the editor. */
  scrollY: number;
  setTitle: (title: string) => void;
  setResults: (results: SearchResult[] | null) => void;
  setScrollY: (scrollY: number) => void;
}

export const useSearchStore = create<SearchState>((set) => ({
  title: "",
  results: null,
  scrollY: 0,
  setTitle: (title) => set({ title }),
  setResults: (results) => set({ results }),
  setScrollY: (scrollY) => set({ scrollY }),
}));
