import { useEffect, useMemo, useRef, useState } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { motion } from "motion/react";
import { Languages, Loader2, Trash2 } from "lucide-react";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";
import { formatElapsed, useElapsedSeconds } from "../hooks/useElapsedSeconds";
import type { HealthResponse, Line, TranslateSource } from "../api/types";

interface Props {
  trackId: string;
}

/**
 * In-progress caption shown next to the Translate button — tailored per
 * backend (see HealthResponse) because a self-hosted LLM (internal/localllm)
 * genuinely runs for minutes on a long track, and a plain spinner with no
 * explanation reads as "hung", not "working". Names the actual configured
 * model (health.translate_model) rather than the bare provider id —
 * "localllm" on its own is an internal implementation detail (this same
 * backend path works with whatever OpenAI-compatible model is configured,
 * not one specific brand) and would read oddly to whoever's actually using
 * a hosted instance of this app. Falls back to a generic message before the
 * health check resolves, for libretranslate (translate_model is "" — plain
 * NMT has no single "model" the way an LLM does), or an unrecognized
 * provider.
 */
function translatingCaption(health: HealthResponse | undefined): string {
  const model = health?.translate_model ? ` (${health.translate_model})` : "";
  switch (health?.translate_provider) {
    case "localllm":
      return `Menerjemahkan pakai LLM lokal${model} — makin panjang lagunya, makin lama (bisa beberapa menit untuk lagu penuh).`;
    case "gemini":
      return `Menerjemahkan pakai Gemini${model}…`;
    default:
      return "Menerjemahkan…";
  }
}

const LANGS = [
  { code: "id", label: "Indonesia" },
  { code: "en", label: "English" },
  { code: "ja", label: "日本語" },
];

/**
 * Most common translation_lang among scrape-sourced lines — a track is
 * normally aligned from one scrape source, so this is almost always
 * unanimous; the vote just keeps it sane if lines came from more than one.
 */
function dominantScrapeLang(lines: Line[]): string {
  const counts = new Map<string, number>();
  for (const l of lines) {
    if (l.method !== "scrape" || !l.translation_lang) continue;
    counts.set(l.translation_lang, (counts.get(l.translation_lang) ?? 0) + 1);
  }
  let best = "";
  let bestCount = 0;
  for (const [lang, count] of counts) {
    if (count > bestCount) {
      best = lang;
      bestCount = count;
    }
  }
  return best;
}

/**
 * Cabang A: kirim baris ke LibreTranslate sekaligus (plan.md #5).
 *
 * Sumber teks yang diterjemahkan tergantung `source`:
 *  - "original": selalu dari lirik asli (Line.Original) — perilaku lama.
 *  - "scrape": chain dari Line.Translation pada baris yang sudah
 *    method="scrape" (mis. hasil scrape EN dari utatime.com), bukan dari
 *    lirik asli — lihat backend httpapi/translate_handler.go. Baris tanpa
 *    data scrape tetap fallback ke lirik asli di backend.
 *
 * target_lang yang sama dengan bahasa lirik asli, atau (dalam mode "scrape")
 * sama dengan bahasa hasil scrape, gak masuk akal buat ditranslate — panel
 * ini menandainya lewat `blockedReason` dan menawarkan "Hapus translation"
 * sebagai gantinya, bukan cuma diam-diam gagal.
 */
export function TranslatePanel({ trackId }: Props) {
  const track = useEditorStore((s) => s.track);
  const patchLine = useEditorStore((s) => s.patchLine);
  const lines = track?.lines ?? [];

  const hasScrapeData = lines.some((l) => l.method === "scrape");
  const [targetLang, setTargetLang] = useState("id");
  const [source, setSource] = useState<TranslateSource>(hasScrapeData ? "scrape" : "original");

  // hasScrapeData can flip false -> true well after this component mounted
  // (user scrapes + aligns after opening the editor) — the useState
  // initializer above only ran once, at the time there was no scrape data
  // yet, so `source` was stuck on "original" even once scrape data showed
  // up. Keep it following "scrape" as the data appears, but stop once the
  // user has actually touched the toggle themselves.
  const userPickedSourceRef = useRef(false);
  useEffect(() => {
    if (hasScrapeData && !userPickedSourceRef.current) setSource("scrape");
  }, [hasScrapeData]);

  function handleSourceChange(next: TranslateSource) {
    userPickedSourceRef.current = true;
    setSource(next);
  }

  const scrapeLang = useMemo(() => dominantScrapeLang(lines), [lines]);
  const originalLang = track?.language ?? "";

  const blockedReason: string | null = !originalLang
    ? null
    : targetLang === originalLang
      ? "sama dengan bahasa lirik asli"
      : source === "scrape" && !!scrapeLang && targetLang === scrapeLang
        ? "sama dengan bahasa hasil scrape"
        : null;

  // Cached indefinitely and never retried: the active provider is a
  // startup-time config choice (see cmd/server/main.go), not something that
  // changes mid-session, and this is only used for in-progress copy — not
  // worth a retry loop or a stale warning if it fails once.
  const healthQuery = useQuery({
    queryKey: ["health"],
    queryFn: api.health,
    staleTime: Infinity,
    retry: false,
  });

  const translateMutation = useMutation({
    mutationFn: () => api.translateTrack(trackId, { target_lang: targetLang, source }),
    onSuccess: (resp) => {
      for (const line of resp.lines) patchLine(line.id, line);
    },
  });
  const translateElapsed = useElapsedSeconds(translateMutation.isPending);

  const clearMutation = useMutation({
    mutationFn: () => api.clearTranslation(trackId),
    onSuccess: (resp) => {
      for (const line of resp.lines) patchLine(line.id, line);
    },
  });

  return (
    <div className="flex flex-wrap items-center gap-2">
      {hasScrapeData && (
        <div className="flex items-center gap-1 rounded-lg border border-slate-800 bg-slate-900/60 p-1 text-xs">
          {/* Scrape listed first, matching the default selection above
              (hasScrapeData ? "scrape" : "original") — when scrape data
              exists it's the safer/preferred source, so it shouldn't sit
              behind the destructive "timpa lirik asli" option visually. */}
          <button
            onClick={() => handleSourceChange("scrape")}
            title="Terjemahkan dari terjemahan hasil scrape yang sudah ada, bukan lirik asli"
            className={`rounded-md px-2 py-1 font-medium transition-colors ${
              source === "scrape" ? "bg-violet-600 text-white" : "text-slate-400 hover:text-slate-200"
            }`}
          >
            Dari data scrape
          </button>
          <button
            onClick={() => handleSourceChange("original")}
            title="Terjemahkan dari lirik asli, timpa apa pun yang ada"
            className={`rounded-md px-2 py-1 font-medium transition-colors ${
              source === "original" ? "bg-violet-600 text-white" : "text-slate-400 hover:text-slate-200"
            }`}
          >
            Timpa dari lirik asli
          </button>
        </div>
      )}

      <select
        value={targetLang}
        onChange={(e) => setTargetLang(e.target.value)}
        className="rounded-lg border border-slate-800 bg-slate-900/60 px-3 py-2 text-sm text-slate-300 outline-none transition-colors hover:border-slate-700 focus:border-violet-500/50"
      >
        {LANGS.map((l) => {
          const disabled =
            (!!originalLang && l.code === originalLang) ||
            (source === "scrape" && !!scrapeLang && l.code === scrapeLang);
          return (
            <option key={l.code} value={l.code} disabled={disabled}>
              {l.label}
              {disabled ? " (sudah dalam bahasa ini)" : ""}
            </option>
          );
        })}
      </select>

      {blockedReason ? (
        <motion.button
          key="clear"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          whileTap={{ scale: 0.95 }}
          onClick={() => clearMutation.mutate()}
          disabled={clearMutation.isPending}
          title={`Target lang ${blockedReason} — hapus translation sebagai gantinya`}
          className="inline-flex items-center gap-2 rounded-lg bg-rose-600/90 px-3.5 py-2 text-sm font-medium text-white shadow-md shadow-rose-600/20 transition-colors hover:bg-rose-500 disabled:cursor-not-allowed disabled:bg-slate-700 disabled:shadow-none"
        >
          {clearMutation.isPending ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Trash2 className="size-4" />
          )}
          Hapus translation
        </motion.button>
      ) : (
        <motion.button
          key="translate"
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          whileTap={{ scale: 0.95 }}
          onClick={() => translateMutation.mutate()}
          disabled={translateMutation.isPending}
          className="inline-flex items-center gap-2 rounded-lg bg-violet-600 px-3.5 py-2 text-sm font-medium text-white shadow-md shadow-violet-600/20 transition-colors hover:bg-violet-500 disabled:cursor-not-allowed disabled:bg-slate-700 disabled:shadow-none"
        >
          {translateMutation.isPending ? (
            <>
              <Loader2 className="size-4 animate-spin" />
              Menerjemahkan… {formatElapsed(translateElapsed)}
            </>
          ) : (
            <>
              <Languages className="size-4" />
              Translate
            </>
          )}
        </motion.button>
      )}

      {translateMutation.isPending && (
        <motion.span
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="text-xs text-slate-500"
        >
          {translatingCaption(healthQuery.data)}
        </motion.span>
      )}

      {translateMutation.isSuccess && (
        <motion.span
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="text-xs text-slate-500"
        >
          {translateMutation.data.lines.length} baris diterjemahkan
          {translateMutation.data.failed?.length ? `, ${translateMutation.data.failed.length} gagal` : ""}
          {" · selesai dalam "}
          {formatElapsed(translateElapsed)}
        </motion.span>
      )}
      {translateMutation.isError && (
        <span className="text-xs text-rose-400">
          {(translateMutation.error as Error).message} (setelah {formatElapsed(translateElapsed)})
        </span>
      )}
      {clearMutation.isSuccess && (
        <motion.span
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="text-xs text-slate-500"
        >
          {clearMutation.data.lines.length} baris dihapus translation-nya
        </motion.span>
      )}
      {clearMutation.isError && (
        <span className="text-xs text-rose-400">{(clearMutation.error as Error).message}</span>
      )}
    </div>
  );
}
