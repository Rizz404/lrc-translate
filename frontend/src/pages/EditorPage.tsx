import { useEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { motion } from "motion/react";
import { ArrowLeft, Eraser, Loader2, RotateCcw } from "lucide-react";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";
import { LineRow } from "../components/LineRow";
import { CopyDownloadBar } from "../components/CopyDownloadBar";
import { TranslatePanel } from "../components/TranslatePanel";
import { RomanizeButton } from "../components/RomanizeButton";
import { ScrapePanel } from "../components/ScrapePanel";

export function EditorPage() {
  const { trackId } = useParams<{ trackId: string }>();
  const navigate = useNavigate();

  const { data: fetchedTrack, isLoading, error } = useQuery({
    queryKey: ["track", trackId],
    queryFn: () => api.getTrack(trackId!),
    enabled: !!trackId,
  });

  const track = useEditorStore((s) => s.track);
  const setTrack = useEditorStore((s) => s.setTrack);
  const patchLine = useEditorStore((s) => s.patchLine);

  useEffect(() => {
    if (fetchedTrack) setTrack(fetchedTrack);
    return () => setTrack(null);
  }, [fetchedTrack, setTrack]);

  // Remounting ScrapePanel via a key wipes its internal useState/useMutation
  // state (search results, resolved URL, raw text, alignment status) without
  // touching any already-applied line data — same effect a browser refresh
  // has on that panel today, just without losing the rest of the editor.
  const [scrapeResetKey, setScrapeResetKey] = useState(0);

  const resetTrackMutation = useMutation({
    mutationFn: () => api.resetTrack(trackId!),
    onSuccess: (resp) => {
      for (const line of resp.lines) patchLine(line.id, line);
    },
  });

  function handleResetAllLines() {
    if (
      window.confirm(
        "Hapus semua romanisasi/terjemahan dan kembalikan tiap baris ke LRC asli? Aksi ini tidak bisa dibatalkan.",
      )
    ) {
      resetTrackMutation.mutate();
    }
  }

  if (isLoading) {
    return (
      <div className="flex min-h-svh items-center justify-center text-slate-500">
        <Loader2 className="size-6 animate-spin" />
      </div>
    );
  }
  if (error) {
    return (
      <div className="flex min-h-svh items-center justify-center text-rose-400">
        {(error as Error).message}
      </div>
    );
  }
  if (!track) return null;

  return (
    <div className="mx-auto min-h-svh max-w-3xl px-4 pt-8 pb-20">
      <button
        onClick={() => navigate("/")}
        className="mb-4 inline-flex items-center gap-1.5 text-sm text-slate-500 transition-colors hover:text-slate-300"
      >
        <ArrowLeft className="size-4" />
        Cari lagu lain
      </button>

      <motion.h1
        initial={{ opacity: 0, y: -6 }}
        animate={{ opacity: 1, y: 0 }}
        className="text-2xl font-bold text-slate-100 sm:text-3xl"
      >
        {track.title}
        <span className="text-slate-500"> — {track.artist}</span>
      </motion.h1>

      <div className="sticky top-0 z-10 -mx-4 mt-5 flex flex-wrap items-center gap-3 border-b border-slate-800/70 bg-slate-950/85 px-4 py-3 backdrop-blur-md">
        <CopyDownloadBar track={track} />
        {/* Hidden below sm: in a wrapped flex row this 1px divider ends up
            stranded alone at the row break, rendering as a stray floating
            bar — the gap gives enough visual separation on narrow screens. */}
        <div className="hidden h-6 w-px bg-slate-800 sm:block" />
        {track.language === "ja" && <RomanizeButton trackId={track.id} />}
        <TranslatePanel trackId={track.id} />
      </div>

      {/* Reset controls: intentionally separate from the sticky toolbar above
          (which stays pinned while scrolling the lines below) — these are
          occasional/destructive actions, not something that needs to follow
          you down the page. */}
      <div className="mt-3 flex flex-wrap items-center gap-2">
        <button
          onClick={() => setScrapeResetKey((k) => k + 1)}
          title="Bersihkan hasil pencarian scraping (URL, teks mentah, status alignment) tanpa mengubah baris yang sudah ter-alignment"
          className="inline-flex items-center gap-1.5 rounded-lg px-2 py-1 text-xs text-slate-500 transition-colors hover:bg-slate-900/60 hover:text-slate-300"
        >
          <Eraser className="size-3.5" />
          Reset pencarian scraping
        </button>
        <button
          onClick={handleResetAllLines}
          disabled={resetTrackMutation.isPending}
          title="Hapus semua romanisasi/terjemahan, kembalikan tiap baris ke LRC asli"
          className="inline-flex items-center gap-1.5 rounded-lg px-2 py-1 text-xs text-rose-400/80 transition-colors hover:bg-rose-950/30 hover:text-rose-300 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {resetTrackMutation.isPending ? (
            <Loader2 className="size-3.5 animate-spin" />
          ) : (
            <RotateCcw className="size-3.5" />
          )}
          Reset semua baris
        </button>
        {resetTrackMutation.isError && (
          <span className="text-xs text-rose-400">{(resetTrackMutation.error as Error).message}</span>
        )}
      </div>

      <div className="mt-3">
        <ScrapePanel key={scrapeResetKey} trackId={track.id} />
      </div>

      <div className="mt-5 flex flex-col gap-2">
        {track.lines.map((line) => (
          <LineRow key={line.id} trackId={track.id} line={line} />
        ))}
      </div>
    </div>
  );
}
