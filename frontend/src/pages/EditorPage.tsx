import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate, useParams } from "react-router-dom";
import { motion } from "motion/react";
import { ArrowLeft, Loader2 } from "lucide-react";
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

  useEffect(() => {
    if (fetchedTrack) setTrack(fetchedTrack);
    return () => setTrack(null);
  }, [fetchedTrack, setTrack]);

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
        <div className="h-6 w-px bg-slate-800" />
        {track.language === "ja" && <RomanizeButton trackId={track.id} />}
        <TranslatePanel trackId={track.id} />
      </div>

      <div className="mt-4">
        <ScrapePanel trackId={track.id} />
      </div>

      <div className="mt-5 flex flex-col gap-2">
        {track.lines.map((line) => (
          <LineRow key={line.id} trackId={track.id} line={line} />
        ))}
      </div>
    </div>
  );
}
