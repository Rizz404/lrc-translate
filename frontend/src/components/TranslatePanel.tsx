import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { motion } from "motion/react";
import { Languages, Loader2 } from "lucide-react";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";

interface Props {
  trackId: string;
}

const LANGS = [
  { code: "id", label: "Indonesia" },
  { code: "en", label: "English" },
  { code: "ja", label: "日本語" },
];

/** Cabang A: kirim semua baris ke LibreTranslate sekaligus (plan.md #5). */
export function TranslatePanel({ trackId }: Props) {
  const [targetLang, setTargetLang] = useState("id");
  const patchLine = useEditorStore((s) => s.patchLine);

  const mutation = useMutation({
    mutationFn: () => api.translateTrack(trackId, { target_lang: targetLang }),
    onSuccess: (resp) => {
      for (const line of resp.lines) patchLine(line.id, line);
    },
  });

  return (
    <div className="flex flex-wrap items-center gap-2">
      <select
        value={targetLang}
        onChange={(e) => setTargetLang(e.target.value)}
        className="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-sm text-slate-300 outline-none transition-colors hover:border-slate-700 focus:border-violet-500/50"
      >
        {LANGS.map((l) => (
          <option key={l.code} value={l.code}>
            {l.label}
          </option>
        ))}
      </select>
      <motion.button
        whileTap={{ scale: 0.95 }}
        onClick={() => mutation.mutate()}
        disabled={mutation.isPending}
        className="inline-flex items-center gap-2 rounded-lg bg-violet-600 px-3.5 py-2 text-sm font-medium text-white shadow-md shadow-violet-600/20 transition-colors hover:bg-violet-500 disabled:cursor-not-allowed disabled:bg-slate-700 disabled:shadow-none"
      >
        {mutation.isPending ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <Languages className="size-4" />
        )}
        Translate All (MT)
      </motion.button>

      {mutation.isSuccess && (
        <motion.span
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="text-xs text-slate-500"
        >
          {mutation.data.cache_hits} dari cache, {mutation.data.cache_misses} baru
          {mutation.data.failed?.length ? `, ${mutation.data.failed.length} gagal` : ""}
        </motion.span>
      )}
      {mutation.isError && (
        <span className="text-xs text-rose-400">{(mutation.error as Error).message}</span>
      )}
    </div>
  );
}
