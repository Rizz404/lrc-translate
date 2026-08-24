import { useMutation } from "@tanstack/react-query";
import { motion } from "motion/react";
import { CaseSensitive, Loader2 } from "lucide-react";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";

interface Props {
  trackId: string;
}

/**
 * Runs the internal kagome+gojp/kana pipeline over the track and switches
 * the editor to show/edit romanized text in place of the original (see
 * store/editorStore.ts showRomanized + LineRow.tsx). Lines that already
 * carry a scrape-sourced or manually-edited romanization (more trustworthy
 * than this pipeline) are protected server-side and never overwritten; this
 * button surfaces that so the user isn't left wondering why some lines
 * didn't change.
 */
export function RomanizeButton({ trackId }: Props) {
  const track = useEditorStore((s) => s.track);
  const patchLine = useEditorStore((s) => s.patchLine);
  const showRomanized = useEditorStore((s) => s.showRomanized);
  const setShowRomanized = useEditorStore((s) => s.setShowRomanized);

  const protectedCount =
    track?.lines.filter((l) => l.romanized_source === "scrape" || l.romanized_source === "manual")
      .length ?? 0;
  const hasAnyRomanized = track?.lines.some((l) => l.romanized) ?? false;

  const mutation = useMutation({
    mutationFn: () => api.romanizeTrack(trackId),
    onSuccess: (resp) => {
      for (const line of resp.lines) patchLine(line.id, line);
      setShowRomanized(true);
    },
  });

  return (
    <div className="flex flex-wrap items-center gap-2">
      <motion.button
        whileTap={{ scale: 0.95 }}
        onClick={() => mutation.mutate()}
        disabled={mutation.isPending}
        title={
          protectedCount > 0
            ? `${protectedCount} baris sudah pakai romanisasi resmi/manual — itu tidak akan ditimpa, cuma baris lain yang diproses.`
            : undefined
        }
        className="inline-flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3.5 py-2 text-sm font-medium text-slate-300 transition-colors hover:border-slate-700 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
      >
        {mutation.isPending ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <CaseSensitive className="size-4" />
        )}
        Romanize
      </motion.button>

      {hasAnyRomanized && (
        <div className="inline-flex rounded-lg border border-slate-800 bg-slate-900/60 p-0.5 text-xs">
          <button
            onClick={() => setShowRomanized(false)}
            className={`rounded-md px-2.5 py-1 font-medium transition-colors ${
              !showRomanized ? "bg-violet-600 text-white" : "text-slate-400 hover:text-white"
            }`}
          >
            Asli
          </button>
          <button
            onClick={() => setShowRomanized(true)}
            className={`rounded-md px-2.5 py-1 font-medium transition-colors ${
              showRomanized ? "bg-violet-600 text-white" : "text-slate-400 hover:text-white"
            }`}
          >
            Romaji
          </button>
        </div>
      )}

      {protectedCount > 0 && !mutation.isError && (
        <span className="text-xs text-amber-400/80">
          {protectedCount} baris romanisasi resmi/manual — tidak akan ditimpa
        </span>
      )}
      {mutation.isSuccess && (mutation.data.skipped_count ?? 0) > 0 && (
        <span className="text-xs text-emerald-400">
          {mutation.data.skipped_count} baris dipertahankan
        </span>
      )}
      {mutation.isError && (
        <span className="text-xs text-rose-400">{(mutation.error as Error).message}</span>
      )}
    </div>
  );
}
