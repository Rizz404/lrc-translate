import { motion } from "motion/react";
import { useTranslation } from "react-i18next";
import { Disc3, Loader2, MicVocal } from "lucide-react";
import type { SearchResult } from "../api/types";

function formatDuration(ms: number): string {
  const totalSec = Math.round(ms / 1000);
  const min = Math.floor(totalSec / 60);
  const sec = totalSec % 60;
  return `${min}:${sec.toString().padStart(2, "0")}`;
}

interface Props {
  results: SearchResult[];
  onSelect: (result: SearchResult) => void;
  pendingId: number | null;
}

const listVariants = {
  hidden: {},
  show: { transition: { staggerChildren: 0.035 } },
};

const itemVariants = {
  hidden: { opacity: 0, y: 10 },
  show: { opacity: 1, y: 0 },
};

export function SongCandidateList({ results, onSelect, pendingId }: Props) {
  const { t } = useTranslation();

  if (results.length === 0) {
    return (
      <p className="py-10 text-center text-sm text-neutral-500">
        {t("candidates.empty")}
      </p>
    );
  }

  return (
    <motion.ul
      variants={listVariants}
      initial="hidden"
      animate="show"
      className="flex flex-col gap-2"
    >
      {results.map((r) => (
        <motion.li key={r.lrclib_id} variants={itemVariants} transition={{ duration: 0.25 }}>
          {/* The whole card is the tap target — no separate "Pilih" button. */}
          <motion.button
            type="button"
            whileTap={{ scale: 0.98 }}
            disabled={!r.has_synced_lyrics || pendingId === r.lrclib_id}
            onClick={() => onSelect(r)}
            className="group flex w-full items-center gap-4 rounded-xl border border-neutral-800/80 bg-neutral-900/50 p-3.5 text-left backdrop-blur transition-colors hover:border-primary-500/40 hover:bg-neutral-900 disabled:cursor-not-allowed disabled:opacity-50 disabled:hover:border-neutral-800/80 disabled:hover:bg-neutral-900/50"
          >
            <div className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-neutral-800/80 text-neutral-500 transition-colors group-hover:text-primary-400">
              <Disc3 className="size-5" />
            </div>

            <div className="min-w-0 flex-1">
              <div className="truncate font-medium text-neutral-100">{r.title}</div>

              <div className="mt-0.5 flex items-center gap-1.5 text-sm text-neutral-500">
                <MicVocal className="size-3.5 shrink-0" />
                <span className="truncate">{r.artist}</span>
              </div>

              {/* Album gets its own wrapping line — a long title shouldn't
                  get clipped or push the duration off-screen (mobile report:
                  couldn't see the full duration). */}
              <div className="mt-0.5 text-xs break-words text-neutral-500">
                {r.album || t("candidates.noAlbum")}
              </div>

              <div className="mt-1 flex flex-wrap items-center gap-1.5 text-xs text-neutral-500">
                <span className="font-mono">{formatDuration(r.duration_ms)}</span>
                {!r.has_synced_lyrics && (
                  <span className="rounded-full bg-warning-500/10 px-2 py-0.5 text-warning-400">
                    {t("candidates.noSyncedLyrics")}
                  </span>
                )}
              </div>
            </div>

            {pendingId === r.lrclib_id && (
              <Loader2 className="size-4 shrink-0 animate-spin text-primary-400" />
            )}
          </motion.button>
        </motion.li>
      ))}
    </motion.ul>
  );
}
