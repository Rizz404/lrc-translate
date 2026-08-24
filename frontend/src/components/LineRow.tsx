import { useEffect, useLayoutEffect, useRef, useState } from "react";
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

// True on browsers that support the `field-sizing: content` CSS property
// (Chrome/Edge 123+, so effectively all current Android/desktop Chrome —
// covers this app's actual users). When supported, the textarea's own
// autosizeClass below tells the browser to size the box to its content
// natively — no JS measurement, so no risk of a stale/short height from a
// measurement that ran before layout (webfont swap, animation, etc.)
// settled. Below, that's exactly the failure mode `resize()` was chasing:
// scrollHeight read at the wrong moment leaves the box a half-line short,
// clipped by `overflow-hidden` with no scrollbar to reveal it's cut off.
const SUPPORTS_FIELD_SIZING =
  typeof CSS !== "undefined" && CSS.supports?.("field-sizing", "content");

/**
 * Grows a textarea to fit its content instead of clipping it at a fixed
 * `rows={1}` height — see user report: "jangan sampai ada scroll di
 * input/textarea". On browsers without `field-sizing` support, falls back
 * to measuring `scrollHeight` on value change, window resize (rotation, or
 * the sm: breakpoint changing the row layout), and once the Inter webfont
 * swaps in.
 */
function useAutosizeTextarea(value: string) {
  const ref = useRef<HTMLTextAreaElement>(null);

  const resize = () => {
    if (SUPPORTS_FIELD_SIZING) return; // browser handles it natively
    const el = ref.current;
    if (!el) return;
    el.style.height = "auto";
    el.style.height = `${el.scrollHeight}px`;
  };

  useLayoutEffect(resize, [value]);

  useEffect(() => {
    if (SUPPORTS_FIELD_SIZING) return;
    window.addEventListener("resize", resize);
    document.fonts?.ready.then(resize);
    return () => window.removeEventListener("resize", resize);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return ref;
}

// Applied to both autosizing textareas via Tailwind's arbitrary-property
// syntax (works regardless of whether a named `field-sizing-*` utility
// exists in this Tailwind version). This is the primary sizing mechanism on
// supporting browsers — native, no JS/timing involved; unsupported browsers
// just ignore the unknown property and fall back to the JS-measured inline
// height from useAutosizeTextarea above.
const AUTOSIZE_CLASS = "[field-sizing:content]";

/**
 * There are only 3 pieces of content per line: timestamp, lyric, translation.
 * "Lyric" is either the original Japanese text or its romanization — never
 * both at once — switched globally via the editor's showRomanized flag (see
 * store/editorStore.ts, flipped on by RomanizeButton). Editing while in
 * romanized mode writes to `romanized`, not `original`, so the source text
 * is never lost.
 */
export function LineRow({ trackId, line }: Props) {
  const patchLine = useEditorStore((s) => s.patchLine);
  const showRomanized = useEditorStore((s) => s.showRomanized);
  const usingRomanized = showRomanized && !!line.romanized;

  const [lyric, setLyric] = useState(usingRomanized ? line.romanized : line.original);
  const [timestamp, setTimestamp] = useState(line.timestamp);
  const [translation, setTranslation] = useState(line.translation);
  const [error, setError] = useState<string | null>(null);

  // Keep local text in sync when switching Asli/Romaji mode, or when the
  // store's line changes from elsewhere (e.g. after a batch Translate All,
  // Romanize, or scrape-align call).
  useEffect(() => {
    setLyric(usingRomanized ? line.romanized : line.original);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [usingRomanized, line.romanized, line.original]);
  useEffect(() => setTranslation(line.translation), [line.translation]);

  const updateMutation = useMutation({
    mutationFn: (body: {
      original?: string;
      romanized?: string;
      timestamp?: string;
      translation?: string;
    }) => api.updateLine(trackId, line.id, body),
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

  function saveLyricIfChanged() {
    const patch: { original?: string; romanized?: string; timestamp?: string } = {};
    const baseline = usingRomanized ? line.romanized : line.original;
    if (lyric !== baseline) {
      if (usingRomanized) patch.romanized = lyric;
      else patch.original = lyric;
    }
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

  const lyricRef = useAutosizeTextarea(lyric);
  const translationRef = useAutosizeTextarea(translation);

  return (
    <motion.div
      layout
      initial={{ opacity: 0, y: 6 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.2 }}
      className={`group relative flex flex-col gap-2 rounded-xl border p-3 backdrop-blur transition-colors sm:flex-row sm:gap-3 ${
        line.needs_review
          ? "border-amber-500/40 bg-amber-500/[0.04]"
          : "border-slate-800/70 bg-slate-900/40 hover:border-slate-700"
      }`}
    >
      <input
        value={timestamp}
        onChange={(e) => setTimestamp(e.target.value)}
        onBlur={saveLyricIfChanged}
        className="h-fit w-[92px] shrink-0 self-start rounded-lg border border-transparent bg-slate-800/60 px-2 py-2.5 text-center font-mono text-xs text-slate-400 outline-none transition-colors hover:border-slate-700 focus:border-violet-500/50 focus:text-slate-200 focus:ring-2 focus:ring-violet-500/15"
      />

      <div className="flex min-w-0 flex-1 flex-col gap-1.5">
        <textarea
          ref={lyricRef}
          value={lyric}
          onChange={(e) => setLyric(e.target.value)}
          onBlur={saveLyricIfChanged}
          rows={1}
          className={`w-full resize-none overflow-hidden rounded-lg bg-slate-800/30 px-3 py-2.5 leading-snug text-slate-100 outline-none transition-colors placeholder:text-slate-600 hover:bg-slate-800/50 focus:bg-slate-800/70 focus:ring-2 focus:ring-violet-500/15 ${AUTOSIZE_CLASS}`}
        />

        <div className="flex flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-2">
          <textarea
            ref={translationRef}
            placeholder="Terjemahan…"
            value={translation}
            onChange={(e) => handleTranslationChange(e.target.value)}
            rows={1}
            className={`min-w-0 flex-1 resize-none overflow-hidden rounded-lg bg-slate-800/40 px-3 py-2 leading-snug text-sm text-slate-300 outline-none transition-colors placeholder:text-slate-600 hover:bg-slate-800/60 focus:bg-slate-800/80 focus:ring-2 focus:ring-violet-500/15 ${AUTOSIZE_CLASS}`}
          />
          <div className="flex shrink-0 items-center gap-2 self-start sm:self-auto">
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
        </div>

        {error && <span className="px-1 text-xs text-rose-400">{error}</span>}
      </div>

      {isSaving && (
        <Loader2 className="absolute top-3 right-3 size-3.5 animate-spin text-slate-600" />
      )}
    </motion.div>
  );
}
