import { useState } from "react";
import { motion } from "motion/react";
import { Check, Copy, Download } from "lucide-react";
import type { Track } from "../api/types";
import { lrcFileName, trackToLrcText } from "../utils/lrcFormat";

interface Props {
  track: Track;
}

/**
 * "Skip terjemahan -> langsung copy/download LRC apa adanya" (plan.md #4).
 * Generated entirely client-side from the current editor state — no backend
 * round-trip, so this works even before any translation step runs.
 */
export function CopyDownloadBar({ track }: Props) {
  const [copied, setCopied] = useState(false);

  async function handleCopy() {
    await navigator.clipboard.writeText(trackToLrcText(track));
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  }

  function handleDownload() {
    const blob = new Blob([trackToLrcText(track)], { type: "text/plain;charset=utf-8" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = lrcFileName(track);
    a.click();
    URL.revokeObjectURL(url);
  }

  return (
    <div className="flex gap-2">
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
    </div>
  );
}
