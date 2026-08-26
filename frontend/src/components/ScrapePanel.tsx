import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { motion, AnimatePresence } from "motion/react";
import { ChevronDown, ExternalLink, Globe2, Link2, Loader2, Wand2 } from "lucide-react";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";

interface Props {
  trackId: string;
}

/**
 * Cabang C. Two ways in:
 *  1. "Cari otomatis" tries guessing a utatime.com URL from the track's
 *     title/artist (best-effort — utatime.com's own search sits behind a
 *     bot challenge we can't/won't get around, see
 *     backend/internal/scrape/utatime.go).
 *  2. When that fails (or for any other site), paste a page URL directly.
 * Either way, stage 2 ("Terapkan Alignment") runs the positional/block
 * heuristic and writes results onto lines, always flagged "perlu dicek".
 */
export function ScrapePanel({ trackId }: Props) {
  const [open, setOpen] = useState(false);
  const [showManual, setShowManual] = useState(false);
  const [url, setUrl] = useState("");
  const [showRaw, setShowRaw] = useState(false);
  // Only asked for when the scrape itself couldn't auto-detect a language
  // (i.e. every site except utatime.com — see ScrapeTrackResponse.language).
  // Needed so handleTranslateTrack's "translate dari data scrape" mode knows
  // what language it's chaining off of.
  const [scrapedLanguage, setScrapedLanguage] = useState("");
  const patchLine = useEditorStore((s) => s.patchLine);

  const scrapeMutation = useMutation({
    mutationFn: (sourceUrl?: string) => api.scrapeTrack(trackId, sourceUrl),
    onError: () => setShowManual(true), // auto-discovery failed -> offer the manual fallback
    onSuccess: () => setScrapedLanguage(""),
  });

  const alignMutation = useMutation({
    mutationFn: (scrapeSourceId: number) =>
      api.alignTrack(trackId, scrapeSourceId, scrapedLanguage.trim() || undefined),
    onSuccess: (resp) => {
      for (const line of resp.lines) patchLine(line.id, line);
    },
  });

  const needsManualLanguage = scrapeMutation.isSuccess && !scrapeMutation.data.language;

  return (
    <div className="rounded-xl border border-slate-800/70 bg-slate-900/40">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between gap-2 px-4 py-3 text-left text-sm font-medium text-slate-300 transition-colors hover:text-white"
      >
        <span className="flex items-center gap-2">
          <Globe2 className="size-4" />
          Cari terjemahan dari web (scraping)
        </span>
        <ChevronDown className={`size-4 transition-transform ${open ? "rotate-180" : ""}`} />
      </button>

      <AnimatePresence initial={false}>
        {open && (
          <motion.div
            initial={{ height: 0, opacity: 0 }}
            animate={{ height: "auto", opacity: 1 }}
            exit={{ height: 0, opacity: 0 }}
            transition={{ duration: 0.2 }}
            className="overflow-hidden"
          >
            <div className="flex flex-col gap-3 border-t border-slate-800/70 px-4 py-4">
              <p className="text-xs text-slate-500">
                {/* KISS 2026-08-26: strengthened from a soft "perlu dicek" —
                    translate (tombol di atas) sudah memprioritaskan Gemini/
                    LibreTranslate; scrape ini opsi terakhir, cuma dipakai
                    kalau perlu terjemahan siap-pakai dari sumber luar. */}
                Opsi terakhir, bukan yang pertama dicoba — pakai Translate (Gemini/LibreTranslate) di
                atas dulu. Alignment di sini cuma perkiraan posisi, dan pernah salah pasang baris tanpa
                kelihatan jelas per-baris (baris individual tetap masuk akal, cuma ketuker sama baris
                asli yang salah — lihat riwayat bug di{" "}
                <code className="text-slate-400">docs/backend/fixes-2026-08-25-scrape-alignment.md</code>).
                Hasilnya selalu ditandai <em>"perlu dicek"</em>, bukan hasil verifikasi.
              </p>

              <div className="flex flex-wrap items-center gap-2">
                <motion.button
                  whileTap={{ scale: 0.95 }}
                  onClick={() => scrapeMutation.mutate(undefined)}
                  disabled={scrapeMutation.isPending}
                  className="inline-flex items-center gap-2 rounded-lg bg-violet-600 px-3.5 py-2 text-sm font-medium text-white shadow-md shadow-violet-600/20 transition-colors hover:bg-violet-500 disabled:cursor-not-allowed disabled:bg-slate-700 disabled:shadow-none"
                >
                  {scrapeMutation.isPending ? (
                    <Loader2 className="size-4 animate-spin" />
                  ) : (
                    <Globe2 className="size-4" />
                  )}
                  Cari otomatis (utatime.com)
                </motion.button>

                {scrapeMutation.isSuccess && !needsManualLanguage && (
                  <motion.button
                    initial={{ opacity: 0 }}
                    animate={{ opacity: 1 }}
                    whileTap={{ scale: 0.95 }}
                    onClick={() => alignMutation.mutate(scrapeMutation.data.scrape_source_id)}
                    disabled={alignMutation.isPending}
                    className="inline-flex items-center gap-2 rounded-lg bg-amber-600 px-3.5 py-2 text-sm font-medium text-white shadow-md shadow-amber-600/20 transition-colors hover:bg-amber-500 disabled:cursor-not-allowed disabled:bg-slate-700 disabled:shadow-none"
                  >
                    {alignMutation.isPending ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : (
                      <Wand2 className="size-4" />
                    )}
                    Terapkan Alignment
                  </motion.button>
                )}
              </div>

              {needsManualLanguage && (
                <motion.div
                  initial={{ opacity: 0 }}
                  animate={{ opacity: 1 }}
                  className="flex flex-wrap items-center gap-2 rounded-lg border border-amber-500/30 bg-amber-500/[0.06] px-3 py-2.5"
                >
                  <span className="text-xs text-slate-400">
                    Situs ini bukan utatime.com — bahasa hasil scrape gak bisa dideteksi otomatis.
                    Isi kode bahasanya (mis. <code>en</code>) biar fitur translate tahu ini bahasa
                    apa:
                  </span>
                  <input
                    value={scrapedLanguage}
                    onChange={(e) => setScrapedLanguage(e.target.value)}
                    placeholder="en"
                    className="w-20 rounded-lg border border-slate-800 bg-slate-950/60 px-2.5 py-1.5 text-center text-sm text-slate-200 outline-none transition-colors placeholder:text-slate-600 focus:border-violet-500/50 focus:ring-2 focus:ring-violet-500/15"
                  />
                  <motion.button
                    whileTap={{ scale: 0.95 }}
                    onClick={() => alignMutation.mutate(scrapeMutation.data!.scrape_source_id)}
                    disabled={alignMutation.isPending || !scrapedLanguage.trim()}
                    className="inline-flex items-center gap-2 rounded-lg bg-amber-600 px-3.5 py-2 text-sm font-medium text-white shadow-md shadow-amber-600/20 transition-colors hover:bg-amber-500 disabled:cursor-not-allowed disabled:bg-slate-700 disabled:shadow-none"
                  >
                    {alignMutation.isPending ? (
                      <Loader2 className="size-4 animate-spin" />
                    ) : (
                      <Wand2 className="size-4" />
                    )}
                    Terapkan Alignment
                  </motion.button>
                </motion.div>
              )}

              {scrapeMutation.isError && (
                <p className="text-xs text-rose-400">
                  {(scrapeMutation.error as Error).message}
                </p>
              )}

              {scrapeMutation.isSuccess && (
                <>
                  <a
                    href={scrapeMutation.data.resolved_url}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex w-fit items-center gap-1.5 text-xs text-emerald-400 hover:underline"
                  >
                    <ExternalLink className="size-3.5" />
                    {scrapeMutation.data.auto_discovered ? "Ditemukan otomatis: " : "Sumber: "}
                    {scrapeMutation.data.resolved_url}
                  </a>

                  {scrapeMutation.data.raw_romanized && (
                    <p className="text-xs text-sky-400">
                      Romanisasi resmi dari situs ini ikut terambil — akan menggantikan romanisasi
                      otomatis pada baris yang ter-alignment.
                    </p>
                  )}

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
                </>
              )}

              {alignMutation.isError && (
                <p className="text-xs text-rose-400">{(alignMutation.error as Error).message}</p>
              )}
              {alignMutation.isSuccess && (
                <p className="text-xs text-emerald-400">
                  {alignMutation.data.lines.length} baris ter-alignment (terjemahan
                  {scrapeMutation.data?.raw_romanized ? " + romanisasi" : ""}) — cek satu-satu,
                  baris ditandai amber di bawah.
                </p>
              )}

              <div className="border-t border-slate-800/70 pt-3">
                {!showManual ? (
                  <button
                    onClick={() => setShowManual(true)}
                    className="text-xs text-slate-500 hover:text-slate-300"
                  >
                    atau paste link halaman manual →
                  </button>
                ) : (
                  <div className="flex flex-col gap-2">
                    <p className="text-xs text-slate-500">
                      Cari sendiri halaman yang berisi terjemahan lagu ini, lalu paste link-nya:
                    </p>
                    <div className="flex flex-wrap items-center gap-2">
                      <div className="relative min-w-0 flex-1">
                        <Link2 className="pointer-events-none absolute top-1/2 left-3 size-4 -translate-y-1/2 text-slate-500" />
                        <input
                          value={url}
                          onChange={(e) => setUrl(e.target.value)}
                          placeholder="https://…"
                          className="w-full rounded-lg border border-slate-800 bg-slate-950/60 py-2 pr-3 pl-9 text-sm text-slate-200 outline-none transition-colors placeholder:text-slate-600 focus:border-violet-500/50 focus:ring-2 focus:ring-violet-500/15"
                        />
                      </div>
                      <motion.button
                        whileTap={{ scale: 0.95 }}
                        onClick={() => scrapeMutation.mutate(url)}
                        disabled={scrapeMutation.isPending || !url.trim()}
                        className="inline-flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3.5 py-2 text-sm font-medium text-slate-300 transition-colors hover:border-slate-700 hover:text-white disabled:cursor-not-allowed disabled:opacity-40"
                      >
                        {scrapeMutation.isPending ? (
                          <Loader2 className="size-4 animate-spin" />
                        ) : (
                          <Globe2 className="size-4" />
                        )}
                        Scrape halaman ini
                      </motion.button>
                    </div>
                  </div>
                )}
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>
    </div>
  );
}
