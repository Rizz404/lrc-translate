import { motion } from "motion/react";
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
  if (results.length === 0) {
    return (
      <p className="py-10 text-center text-sm text-slate-500">
        Tidak ada hasil. Coba judul lain.
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
        <motion.li
          key={r.lrclib_id}
          variants={itemVariants}
          transition={{ duration: 0.25 }}
          className="group flex items-center gap-4 rounded-xl border border-slate-800/80 bg-slate-900/50 p-3.5 pr-4 backdrop-blur transition-colors hover:border-violet-500/40 hover:bg-slate-900"
        >
          <div className="flex size-11 shrink-0 items-center justify-center rounded-lg bg-slate-800/80 text-slate-500 transition-colors group-hover:text-violet-400">
            <Disc3 className="size-5" />
          </div>

          <div className="min-w-0 flex-1 text-left">
            <div className="truncate font-medium text-slate-100">{r.title}</div>
            <div className="mt-0.5 flex items-center gap-1.5 truncate text-sm text-slate-500">
              <MicVocal className="size-3.5 shrink-0" />
              <span className="truncate">{r.artist}</span>
              <span className="text-slate-700">·</span>
              <span className="shrink-0">{r.album || "tanpa album"}</span>
              <span className="text-slate-700">·</span>
              <span className="shrink-0 font-mono text-xs">{formatDuration(r.duration_ms)}</span>
              {!r.has_synced_lyrics && (
                <span className="shrink-0 rounded-full bg-amber-500/10 px-2 py-0.5 text-xs text-amber-400">
                  tanpa synced lyrics
                </span>
              )}
            </div>
          </div>

          <button
            disabled={!r.has_synced_lyrics || pendingId === r.lrclib_id}
            onClick={() => onSelect(r)}
            className="shrink-0 rounded-lg bg-slate-800 px-3.5 py-2 text-sm font-medium text-slate-200 transition-all hover:bg-violet-600 hover:text-white active:scale-95 disabled:cursor-not-allowed disabled:opacity-40"
          >
            {pendingId === r.lrclib_id ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              "Pilih"
            )}
          </button>
        </motion.li>
      ))}
    </motion.ul>
  );
}
