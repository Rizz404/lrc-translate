import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";

interface Props {
  trackId: string;
}

/**
 * Cabang C: user pastes a URL to a page they found with a translation
 * (this app does not auto-crawl any translation database — see
 * backend/internal/scrape/scrape.go for why). Stage 1 (Scrape) fetches +
 * extracts raw text; stage 2 (Terapkan Alignment) runs the positional/block
 * heuristic and writes results onto lines, always flagged "perlu dicek".
 */
export function ScrapePanel({ trackId }: Props) {
  const [open, setOpen] = useState(false);
  const [url, setUrl] = useState("");
  const [showRaw, setShowRaw] = useState(false);
  const patchLine = useEditorStore((s) => s.patchLine);

  const scrapeMutation = useMutation({
    mutationFn: () => api.scrapeTrack(trackId, url),
  });

  const alignMutation = useMutation({
    mutationFn: (scrapeSourceId: number) => api.alignTrack(trackId, scrapeSourceId),
    onSuccess: (resp) => {
      for (const line of resp.lines) patchLine(line.id, line);
    },
  });

  return (
    <div className="rounded-xl border border-slate-800/70 bg-slate-900/40">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between gap-2 px-4 py-3 text-left text-sm font-medium text-slate-300 transition-colors hover:text-white"
      >
        <span className="flex items-center gap-2">Cari terjemahan dari web (scraping)</span>
      </button>

      {open && (
        <div className="flex flex-col gap-3 border-t border-slate-800/70 px-4 py-4">
          <p className="text-xs text-slate-500">
            Tempel link halaman yang sudah berisi terjemahan lagu ini (mis. blog fan
            translation). Hasil alignment selalu ditandai <em>"perlu dicek"</em> — posisinya
            cuma perkiraan, bukan hasil verifikasi.
          </p>

          <div className="relative">
            <input
              value={url}
              onChange={(e) => setUrl(e.target.value)}
              placeholder="https://…"
              className="w-full rounded-lg border border-slate-800 bg-slate-950/60 py-2 pr-3 pl-9 text-sm text-slate-200 outline-none transition-colors placeholder:text-slate-600 focus:border-violet-500/50 focus:ring-2 focus:ring-violet-500/15"
            />
          </div>

          <div className="flex flex-wrap items-center gap-2">
            <button
              onClick={() => scrapeMutation.mutate()}
              disabled={scrapeMutation.isPending || !url.trim()}
              className="inline-flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3.5 py-2 text-sm font-medium text-slate-300 transition-colors hover:border-slate-700 hover:text-white disabled:cursor-not-allowed disabled:opacity-40"
            >
              Scrape halaman
            </button>

            {scrapeMutation.isSuccess && (
              <button
                onClick={() => alignMutation.mutate(scrapeMutation.data.scrape_source_id)}
                disabled={alignMutation.isPending}
                className="inline-flex items-center gap-2 rounded-lg bg-amber-600 px-3.5 py-2 text-sm font-medium text-white shadow-md shadow-amber-600/20 transition-colors hover:bg-amber-500 disabled:cursor-not-allowed disabled:bg-slate-700 disabled:shadow-none"
              >
                Terapkan Alignment
              </button>
            )}
          </div>

          {scrapeMutation.isError && (
            <p className="text-xs text-rose-400">{(scrapeMutation.error as Error).message}</p>
          )}

          {scrapeMutation.isSuccess && (
            <div className="rounded-lg border border-slate-800 bg-slate-950/50">
              <button
                onClick={() => setShowRaw((v) => !v)}
                className="w-full px-3 py-2 text-left text-xs text-slate-500 hover:text-slate-300"
              >
                {showRaw ? "Sembunyikan" : "Lihat"} teks mentah hasil scrape (
                {scrapeMutation.data.raw_text.split("\n").length} baris)
              </button>
              {showRaw && (
                <pre className="max-h-48 overflow-auto border-t border-slate-800 p-3 text-xs whitespace-pre-wrap text-slate-400">
                  {scrapeMutation.data.raw_text}
                </pre>
              )}
            </div>
          )}

          {alignMutation.isError && (
            <p className="text-xs text-rose-400">{(alignMutation.error as Error).message}</p>
          )}
          {alignMutation.isSuccess && (
            <p className="text-xs text-emerald-400">
              {alignMutation.data.lines.length} baris ter-alignment — cek satu-satu, baris
              ditandai amber di bawah.
            </p>
          )}
        </div>
      )}
    </div>
  );
}
