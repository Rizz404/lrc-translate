import { motion, AnimatePresence } from "motion/react";
import type { Method } from "../api/types";

const STYLES: Record<Method, string> = {
  none: "bg-slate-800 text-slate-500",
  mt: "bg-sky-500/15 text-sky-300 ring-1 ring-sky-500/30",
  ai: "bg-fuchsia-500/15 text-fuchsia-300 ring-1 ring-fuchsia-500/30",
  scrape: "bg-amber-500/15 text-amber-300 ring-1 ring-amber-500/30",
  manual: "bg-emerald-500/15 text-emerald-300 ring-1 ring-emerald-500/30",
};

const LABELS: Record<Method, string> = {
  none: "belum diterjemahkan",
  mt: "MT",
  ai: "AI",
  scrape: "scrape",
  manual: "manual",
};

interface Props {
  method: Method;
  needsReview?: boolean;
}

export function MethodBadge({ method, needsReview }: Props) {
  return (
    <AnimatePresence mode="wait" initial={false}>
      <motion.span
        key={method}
        initial={{ opacity: 0, scale: 0.85 }}
        animate={{ opacity: 1, scale: 1 }}
        exit={{ opacity: 0, scale: 0.85 }}
        transition={{ duration: 0.15 }}
        className={`inline-flex shrink-0 items-center whitespace-nowrap rounded-full px-2.5 py-1 text-xs font-medium ${STYLES[method]}`}
      >
        {LABELS[method]}
        {needsReview && <span className="ml-1 text-amber-400">· perlu dicek</span>}
      </motion.span>
    </AnimatePresence>
  );
}
