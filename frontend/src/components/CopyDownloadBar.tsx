import { useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { Check, Copy, Download, SlidersHorizontal } from "lucide-react";
import type { Track } from "../api/types";
import { useEditorStore } from "../store/editorStore";
import {
  DEFAULT_LRC_EXPORT_OPTIONS,
  lrcFileName,
  trackToLrcText,
  type LrcExportOptions,
} from "../utils/lrcFormat";

interface Props {
  track: Track;
}

/**
 * "Skip terjemahan -> langsung copy/download LRC apa adanya" (plan.md #4).
 * Generated entirely client-side from the current editor state — no backend
 * round-trip, so this works even before any translation step runs.
 *
 * The main lyric line (original vs romanized) isn't chosen here — it just
 * mirrors whatever the editor is currently showing (RomanizeButton's
 * Asli/Romaji toggle, see store/editorStore.ts showRomanized), so export is
 * always WYSIWYG. The gear icon only opens translation-line options.
 */
export function CopyDownloadBar({ track }: Props) {
  const showRomanized = useEditorStore((s) => s.showRomanized);
  const [copied, setCopied] = useState(false);
  const [optionsOpen, setOptionsOpen] = useState(false);
  const [options, setOptions] = useState<LrcExportOptions>(DEFAULT_LRC_EXPORT_OPTIONS);

  async function handleCopy() {
    await navigator.clipboard.writeText(trackToLrcText(track, showRomanized, options));
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  function handleDownload() {
    const blob = new Blob([trackToLrcText(track, showRomanized, options)], {
      type: "text/plain;charset=utf-8",
    });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = lrcFileName(track);
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="relative flex gap-2">
      <motion.button
        whileTap={{ scale: 0.95 }}
        onClick={handleCopy}
        className="inline-flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3.5 py-2 text-sm font-medium text-slate-300 transition-colors hover:border-slate-700 hover:text-white"
      >
        {copied ? <Check className="size-4 text-emerald-400" /> : <Copy className="size-4" />}
        {copied ? "Tersalin!" : "Copy LRC"}
      </motion.button>
      <motion.button
        whileTap={{ scale: 0.95 }}
        onClick={handleDownload}
        className="inline-flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3.5 py-2 text-sm font-medium text-slate-300 transition-colors hover:border-slate-700 hover:text-white"
      >
        <Download className="size-4" />
        Download .lrc
      </motion.button>
      <motion.button
        whileTap={{ scale: 0.95 }}
        onClick={() => setOptionsOpen((v) => !v)}
        title="Opsi export"
        aria-label="Opsi export"
        className={`inline-flex items-center justify-center rounded-lg border px-2.5 py-2 text-sm font-medium transition-colors ${
          optionsOpen
            ? "border-violet-500/50 bg-violet-500/10 text-violet-300"
            : "border-slate-800 bg-slate-900/60 text-slate-300 hover:border-slate-700 hover:text-white"
        }`}
      >
        <SlidersHorizontal className="size-4" />
      </motion.button>

      <AnimatePresence>
        {optionsOpen && (
          <motion.div
            initial={{ opacity: 0, y: -4, scale: 0.98 }}
            animate={{ opacity: 1, y: 0, scale: 1 }}
            exit={{ opacity: 0, y: -4, scale: 0.98 }}
            transition={{ duration: 0.15 }}
            className="absolute top-full left-0 z-20 mt-2 w-64 rounded-xl border border-slate-800 bg-slate-950/95 p-3.5 shadow-xl shadow-black/40 backdrop-blur"
          >
            <p className="text-xs text-slate-500">
              Lirik utama ikut tampilan editor saat ini:{" "}
              <span className="text-slate-300">{showRomanized ? "Romaji" : "Asli"}</span> — ganti
              lewat tombol Asli/Romaji di sebelah Romanize.
            </p>

            <div className="mt-3 border-t border-slate-800/70 pt-3">
              <label className="flex items-center gap-2 text-sm text-slate-300">
                <input
                  type="checkbox"
                  checked={options.includeTranslation}
                  onChange={(e) =>
                    setOptions((o) => ({ ...o, includeTranslation: e.target.checked }))
                  }
                  className="accent-violet-500"
                />
                Sertakan terjemahan
              </label>
              <p className="mt-1 pl-6 text-[11px] text-slate-500">
                Ditambahkan sebagai baris kedua di timestamp yang sama.
              </p>
              <label
                className={`mt-1.5 flex items-center gap-2 pl-6 text-sm ${
                  options.includeTranslation ? "text-slate-300" : "cursor-not-allowed text-slate-600"
                }`}
              >
                <input
                  type="checkbox"
                  checked={options.wrapTranslationInParens}
                  disabled={!options.includeTranslation}
                  onChange={(e) =>
                    setOptions((o) => ({ ...o, wrapTranslationInParens: e.target.checked }))
                  }
                  className="accent-violet-500"
                />
                Bungkus dengan kurung "( )"
              </label>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
