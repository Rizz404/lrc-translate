import { useState } from "react";
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
    <div className="copy-download-bar">
      <button onClick={handleCopy}>{copied ? "Tersalin!" : "Copy LRC"}</button>
      <button onClick={handleDownload}>Download .lrc</button>
    </div>
  );
}
