import { useEffect, useLayoutEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { motion, AnimatePresence } from "motion/react";
import { Search, Music2, Loader2 } from "lucide-react";
import type { SearchResult } from "../api/types";
import { api } from "../api/client";
import { SongCandidateList } from "../components/SongCandidateList";
import { useSearchStore } from "../store/searchStore";

const SEARCH_DEBOUNCE_MS = 400;
const MIN_QUERY_LENGTH = 2;

// Trims and collapses internal whitespace runs (stray double spaces, tabs)
// down to one — a query like "naruto  shippuden" or " naruto" otherwise
// comes back with zero LRCLIB results even though the clean version
// matches. Mirrors backend/internal/httpapi/search_handler.go normalizeQuery
// as defense in depth (this is also enforced server-side).
function normalizeQuery(s: string): string {
  return s.trim().replace(/\s+/g, " ");
}

export function SearchPage() {
  const navigate = useNavigate();
  const title = useSearchStore((s) => s.title);
  const setTitle = useSearchStore((s) => s.setTitle);
  const results = useSearchStore((s) => s.results);
  const setResults = useSearchStore((s) => s.setResults);
  const [importingId, setImportingId] = useState<number | null>(null);

  const searchMutation = useMutation({
    mutationFn: (query: string) => api.search(query),
    onSuccess: setResults,
  });
  const searchMutationRef = useRef(searchMutation);
  searchMutationRef.current = searchMutation;

  // Restore the scroll position the user was at before they navigated away
  // to the editor — runs before paint so there's no visible jump back to
  // the top first. Only on mount: this must not re-fire on every re-render
  // (e.g. as results come back), or it'd fight the user's own scrolling.
  useLayoutEffect(() => {
    window.scrollTo(0, useSearchStore.getState().scrollY);
  }, []);

  // Auto-search as the user types, debounced so we don't hit the API on every keystroke.
  useEffect(() => {
    const query = normalizeQuery(title);
    if (query.length < MIN_QUERY_LENGTH) {
      setResults(null);
      return;
    }

    const timer = setTimeout(() => {
      searchMutationRef.current.mutate(query);
    }, SEARCH_DEBOUNCE_MS);

    return () => clearTimeout(timer);
  }, [title, setResults]);

  const importMutation = useMutation({
    mutationFn: (lrclibId: number) => api.importTrack(lrclibId),
    onSuccess: (track) => navigate(`/track/${track.id}`),
    onSettled: () => setImportingId(null),
  });

  function handleSelect(result: SearchResult) {
    // Snapshot scroll position now, before the import round-trip — restored
    // in the useLayoutEffect above whenever the user comes back here.
    useSearchStore.getState().setScrollY(window.scrollY);
    setImportingId(result.lrclib_id);
    importMutation.mutate(result.lrclib_id);
  }

  return (
    <div className="mx-auto flex min-h-svh max-w-2xl flex-col items-center px-4 pt-20 pb-16 sm:pt-28">
      <motion.div
        initial={{ opacity: 0, y: -12 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5, ease: "easeOut" }}
        className="mb-10 flex flex-col items-center text-center"
      >
        <div className="mb-4 flex size-14 items-center justify-center rounded-2xl bg-gradient-to-br from-violet-500 to-fuchsia-500 shadow-lg shadow-violet-500/30">
          <Music2 className="size-7 text-white" strokeWidth={2.2} />
        </div>
        <h1 className="bg-gradient-to-br from-white to-slate-400 bg-clip-text text-4xl font-extrabold tracking-tight text-transparent sm:text-5xl">
          lrc-translate
        </h1>
        <p className="mt-3 max-w-md text-balance text-slate-400">
          Cari lagu, tarik lirik tersinkron, terjemahkan per baris — semua dalam satu editor.
        </p>
      </motion.div>

      <motion.form
        initial={{ opacity: 0, y: 8 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1 }}
        onSubmit={(e) => {
          e.preventDefault();
          // Enter triggers an immediate search, bypassing the debounce.
          const query = normalizeQuery(title);
          if (query.length >= MIN_QUERY_LENGTH) {
            searchMutation.mutate(query);
          }
        }}
        className="group relative w-full"
      >
        <Search className="pointer-events-none absolute top-1/2 left-4 size-5 -translate-y-1/2 text-slate-500 transition-colors group-focus-within:text-violet-400" />
        <input
          autoFocus
          placeholder="Cari judul lagu…"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="w-full rounded-2xl border border-slate-800 bg-slate-900/70 py-4 pr-12 pl-12 text-base text-slate-100 shadow-lg shadow-black/20 outline-none backdrop-blur transition-all placeholder:text-slate-500 focus:border-violet-500/60 focus:ring-4 focus:ring-violet-500/15"
        />
        {searchMutation.isPending && (
          <Loader2 className="pointer-events-none absolute top-1/2 right-4 size-5 -translate-y-1/2 animate-spin text-violet-400" />
        )}
      </motion.form>

      {searchMutation.isError && (
        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="mt-4 text-sm text-rose-400"
        >
          {(searchMutation.error as Error).message}
        </motion.p>
      )}
      {importMutation.isError && (
        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="mt-4 text-sm text-rose-400"
        >
          {(importMutation.error as Error).message}
        </motion.p>
      )}

      <AnimatePresence mode="wait">
        {results && (
          <motion.div
            key="results"
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            exit={{ opacity: 0 }}
            className="mt-8 w-full"
          >
            <SongCandidateList results={results} onSelect={handleSelect} pendingId={importingId} />
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
