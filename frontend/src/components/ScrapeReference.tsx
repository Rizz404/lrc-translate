import type { LineScrapeContext } from "../api/types";

interface Props {
  context: LineScrapeContext;
  /** The field's current value (translation, or romanized when in Romaji mode) — used to mark which chip (if any) is already applied. */
  current: string;
  onPick: (text: string) => void;
  /**
   * Machine-translated reference for this line — see LineScrapeContexts.ai.
   * Rendered as an extra, visually distinct chip: unlike the scraped
   * prev/matched/next chips (a heuristic guess at position, in whatever
   * language the scrape source happened to be in), this is always
   * correctly positioned and in the reviewer's own target language — the
   * one thing a reviewer who can't read the original lyric's language can
   * actually judge unaided.
   */
  aiText?: string;
}

/**
 * Compact, clickable strip showing the raw scraped line(s) immediately
 * around wherever this field's value was matched from during alignment —
 * see api/types.ts's LineScrapeContext. None of the align heuristics are
 * verified correct (every scrape-aligned line is flagged needs_review), so
 * this exists purely as a side-by-side sanity check: a mismatch between the
 * original lyric above and what's shown here is usually obvious at a
 * glance, and clicking any chip applies it — the same "pick scrape's
 * suggestion or type your own" the translation/lyric textarea already
 * allows, just with the raw candidates put right next to it instead of
 * requiring a trip to the collapsed raw-text viewer in ScrapePanel.
 *
 * Renders nothing when the context is empty (e.g. this line was never
 * touched by a scrape+align, or scrapedRaw didn't cover it).
 */
export function ScrapeReference({ context, current, onPick, aiText }: Props) {
  const items: { text: string; isMatch: boolean }[] = [];
  if (context.prev) items.push({ text: context.prev, isMatch: false });
  if (context.matched) items.push({ text: context.matched, isMatch: true });
  if (context.next) items.push({ text: context.next, isMatch: false });

  if (items.length === 0 && !aiText) return null;

  return (
    <div className="flex flex-wrap items-center gap-1">
      <span
        title="Baris ini diambil dari hasil scrape mentah, dari posisi terdekat yang berhasil dicocokkan otomatis — klik salah satu untuk menerapkannya"
        className="shrink-0 cursor-help text-[10px] tracking-wide text-slate-600 uppercase underline decoration-dotted decoration-slate-700 underline-offset-2"
      >
        mentah
      </span>
      {items.map((item, i) => {
        const isCurrent = item.text === current;
        return (
          <button
            key={i}
            type="button"
            onClick={() => onPick(item.text)}
            title={`Klik untuk pakai: "${item.text}"`}
            className={`max-w-[55vw] truncate rounded-md border px-1.5 py-0.5 text-left text-[11px] transition-colors sm:max-w-[240px] ${
              isCurrent
                ? "border-emerald-500/40 bg-emerald-500/[0.08] text-emerald-300"
                : item.isMatch
                  ? "border-amber-500/30 bg-amber-500/[0.05] text-amber-200/90 hover:bg-amber-500/[0.12] hover:text-amber-100"
                  : "border-slate-800 bg-slate-900/30 text-slate-500 hover:border-slate-700 hover:text-slate-300"
            }`}
          >
            {item.text}
          </button>
        );
      })}
      {aiText && (
        <button
          type="button"
          onClick={() => onPick(aiText)}
          title={`Referensi terjemahan mesin (selalu di posisi yang benar) — klik untuk pakai: "${aiText}"`}
          className={`max-w-[55vw] truncate rounded-md border px-1.5 py-0.5 text-left text-[11px] transition-colors sm:max-w-[240px] ${
            aiText === current
              ? "border-emerald-500/40 bg-emerald-500/[0.08] text-emerald-300"
              : "border-sky-500/30 bg-sky-500/[0.06] text-sky-300/90 hover:bg-sky-500/[0.14] hover:text-sky-200"
          }`}
        >
          <span className="mr-1 font-semibold">AI</span>
          {aiText}
        </button>
      )}
    </div>
  );
}
