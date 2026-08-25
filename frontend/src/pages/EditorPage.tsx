import { useEffect, useLayoutEffect, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { motion } from "motion/react";
import { AlertTriangle, ArrowLeft, Eraser, Loader2, RotateCcw, Sparkles } from "lucide-react";
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

  // Coming from SearchPage scrolled partway down its results list (e.g. a
  // song near the bottom), the browser keeps that same scrollY on this new
  // route — nothing about an SPA navigation resets it. Left alone, the
  // editor renders "pre-scrolled": whatever ends up at that Y offset (often
  // mid-lyrics) is what's visible on load, looking like a scroll that never
  // actually happened. Force it back to the top on every fresh mount (App.tsx
  // keys this page's wrapper on the trackId-bearing pathname, so this only
  // runs when we've genuinely landed on a new track, not on every re-render).
  useLayoutEffect(() => {
    window.scrollTo(0, 0);
  }, []);

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

  // On-demand MT reference for every needs_review line (see
  // ScrapeReference.tsx's "AI" chip) — a same-language sanity check for a
  // reviewer who can't read the original lyric's language, so a wrong
  // scrape-alignment guess that isn't an exact duplicate is still visible.
  // Deliberately a separate opt-in action rather than something align runs
  // automatically: it costs real MT calls/time, only worth spending once
  // the reviewer actually wants to double-check something.
  // lineIds lets a retry target only the ones that failed last time (see the
  // "coba lagi" button below) instead of re-requesting the whole track —
  // failures are usually Gemini's free-tier rate limit under a big batch
  // (see internal/gemini's exponentialBackoff and
  // ai_reference_handler.go's aiReferenceConcurrency), which a narrower,
  // later retry clears more reliably than hammering everything again at
  // once would.
  const aiReferenceMutation = useMutation({
    mutationFn: (lineIds?: number[]) => {
      const needsReview = track!.lines.filter((l) => l.needs_review);
      const targetLang = needsReview.find((l) => l.translation_lang)?.translation_lang || "en";
      return api.aiReference(track!.id, {
        target_lang: targetLang,
        line_ids: lineIds ?? needsReview.map((l) => l.id),
      });
    },
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

  const needsReviewCount = track.lines.filter((l) => l.needs_review).length;

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

      {/* One summary disclaimer instead of repeating "· perlu dicek" on every
          single scrape-sourced badge below (was pure noise on a page full of
          scraped lines) — rows that need review still get an amber border,
          see LineRow.tsx. */}
      {needsReviewCount > 0 && (
        <div className="mt-3 flex flex-col gap-2 rounded-lg border border-amber-500/30 bg-amber-500/[0.06] px-3 py-2 text-xs text-amber-300">
          <div className="flex flex-wrap items-center gap-2">
            <AlertTriangle className="size-3.5 shrink-0" />
            <span>{needsReviewCount} baris hasil scrape perlu dicek manual — ditandai border kuning di bawah.</span>
          </div>
          <div className="flex flex-wrap items-center gap-2 pl-[22px]">
            <button
              onClick={() => aiReferenceMutation.mutate(undefined)}
              disabled={aiReferenceMutation.isPending}
              title="Minta terjemahan mesin per-baris sebagai pembanding — berguna kalau kamu gak bisa baca lirik aslinya, karena hasil ini selalu di posisi yang benar (bukan tebakan alignment). Diproses satu-satu biar gak kena rate limit provider AI-nya."
              className="inline-flex items-center gap-1.5 rounded-lg border border-sky-500/30 bg-sky-500/[0.08] px-2.5 py-1.5 font-medium text-sky-300 transition-colors hover:bg-sky-500/[0.16] hover:text-sky-200 disabled:cursor-not-allowed disabled:opacity-50"
            >
              {aiReferenceMutation.isPending ? (
                <Loader2 className="size-3.5 animate-spin" />
              ) : (
                <Sparkles className="size-3.5" />
              )}
              {aiReferenceMutation.isPending ? "Memproses…" : "Bandingkan dengan AI"}
            </button>
            {aiReferenceMutation.isSuccess && (
              <span className="text-slate-500">
                Referensi AI siap untuk {aiReferenceMutation.data.lines.length} baris — cek chip biru
                "AI" di bawah tiap terjemahan.
              </span>
            )}
            {aiReferenceMutation.isError && (
              <span className="text-rose-400">{(aiReferenceMutation.error as Error).message}</span>
            )}
          </div>
          {(aiReferenceMutation.data?.failed?.length ?? 0) > 0 && (
            <div className="flex flex-wrap items-center gap-2 pl-[22px] text-amber-400">
              <span>
                {aiReferenceMutation.data!.failed!.length} baris gagal diminta — provider AI-nya lagi
                membatasi permintaan (rate limit/kuota). Kalau langsung dicoba lagi masih gagal juga,
                kemungkinan kuotanya baru reset nanti/besok, bukan cuma butuh nunggu beberapa detik.
              </span>
              <button
                onClick={() => aiReferenceMutation.mutate(aiReferenceMutation.data!.failed!.map((f) => f.line_id))}
                disabled={aiReferenceMutation.isPending}
                className="underline decoration-dotted underline-offset-2 hover:text-amber-300 disabled:cursor-not-allowed disabled:opacity-50"
              >
                coba lagi baris yang gagal
              </button>
            </div>
          )}
        </div>
      )}

      <div className="mt-5 flex flex-col gap-2">
        {track.lines.map((line) => (
          <LineRow key={line.id} trackId={track.id} line={line} />
        ))}
      </div>
    </div>
  );
}
