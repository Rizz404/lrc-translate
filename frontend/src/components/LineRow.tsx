import { useEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { motion } from "motion/react";
import { RotateCcw, Loader2 } from "lucide-react";
import type { Line } from "../api/types";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";
import { MethodBadge } from "./MethodBadge";

interface Props {
  trackId: string;
  line: Line;
}

const AUTOSAVE_DEBOUNCE_MS = 800;

export function LineRow({ trackId, line }: Props) {
  const patchLine = useEditorStore((s) => s.patchLine);
  const [original, setOriginal] = useState(line.original);
  const [timestamp, setTimestamp] = useState(line.timestamp);
  const [translation, setTranslation] = useState(line.translation);
  const [error, setError] = useState<string | null>(null);

  // Keep local text in sync when the store's line changes from elsewhere
  // (e.g. after a batch Translate All or Romanize call).
  useEffect(() => setTranslation(line.translation), [line.translation]);

  const updateMutation = useMutation({
    mutationFn: (body: { original?: string; timestamp?: string; translation?: string }) =>
      api.updateLine(trackId, line.id, body),
    onSuccess: (updated) => {
      patchLine(line.id, updated);
      setError(null);
    },
    onError: (err: Error) => setError(err.message),
  });

  const revertMutation = useMutation({
    mutationFn: () => api.revertLine(trackId, line.id),
    onSuccess: (updated) => {
      patchLine(line.id, updated);
      setError(null);
    },
    onError: (err: Error) => setError(err.message),
  });

  function saveOriginalIfChanged() {
    const patch: { original?: string; timestamp?: string } = {};
    if (original !== line.original) patch.original = original;
    if (timestamp !== line.timestamp) patch.timestamp = timestamp;
    if (Object.keys(patch).length > 0) updateMutation.mutate(patch);
  }

  // Debounced autosave for translation edits, so typing doesn't fire a
  // request per keystroke (plan-extended.md M2).
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  function handleTranslationChange(value: string) {
    setTranslation(value);
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      if (value !== line.translation) updateMutation.mutate({ translation: value });
    }, AUTOSAVE_DEBOUNCE_MS);
  }
  useEffect(() => () => { if (debounceRef.current) clearTimeout(debounceRef.current); }, []);

  const isSaving = updateMutation.isPending || revertMutation.isPending;

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
      className={`group relative flex gap-3 rounded-xl border p-3 backdrop-blur transition-colors ${
        line.needs_review
          ? "border-amber-500/40 bg-amber-500/[0.04]"
          : "border-slate-800/70 bg-slate-900/40 hover:border-slate-700"
      }`}
    >
      <input
        value={timestamp}
        onChange={(e) => setTimestamp(e.target.value)}
        onBlur={saveOriginalIfChanged}
        className="h-fit w-[92px] shrink-0 rounded-lg border border-transparent bg-slate-800/60 px-2 py-1.5 text-center font-mono text-xs text-slate-400 outline-none transition-colors hover:border-slate-700 focus:border-violet-500/50 focus:text-slate-200 focus:ring-2 focus:ring-violet-500/15"
      />

      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <textarea
          value={original}
          onChange={(e) => setOriginal(e.target.value)}
          onBlur={saveOriginalIfChanged}
          rows={1}
          className="w-full resize-none rounded-lg bg-transparent px-1 py-0.5 text-slate-100 outline-none transition-colors placeholder:text-slate-600 hover:bg-slate-800/40 focus:bg-slate-800/60 focus:ring-2 focus:ring-violet-500/15"
        />

        {line.romanized && (
          <div className="truncate px-1 text-sm text-slate-500 italic">{line.romanized}</div>
        )}

        <div className="flex items-center gap-2">
          <textarea
            placeholder="Terjemahan…"
            value={translation}
            onChange={(e) => handleTranslationChange(e.target.value)}
            rows={1}
            className="min-w-0 flex-1 resize-none rounded-lg bg-slate-800/40 px-2 py-1 text-sm text-slate-300 outline-none transition-colors placeholder:text-slate-600 hover:bg-slate-800/60 focus:bg-slate-800/80 focus:ring-2 focus:ring-violet-500/15"
          />
          <MethodBadge method={line.method} needsReview={line.needs_review} />
          {line.method === "manual" && (
            <button
              onClick={() => revertMutation.mutate()}
              disabled={revertMutation.isPending}
              title="Kembalikan ke saran awal (kalau ada)"
              className="shrink-0 rounded-full p-1.5 text-slate-500 transition-colors hover:bg-slate-800 hover:text-violet-400"
            >
              <RotateCcw className="size-3.5" />
            </button>
          )}
        </div>

        {error && <span className="px-1 text-xs text-rose-400">{error}</span>}
      </div>

      {isSaving && (
        <Loader2 className="absolute top-3 right-3 size-3.5 animate-spin text-slate-600" />
      )}
    </motion.div>
  );
}
