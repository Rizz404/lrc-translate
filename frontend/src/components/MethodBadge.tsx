import { motion, AnimatePresence } from "motion/react";
import { useTranslation } from "react-i18next";
import type { Method } from "../api/types";

const STYLES: Record<Method, string> = {
  none: "bg-neutral-800 text-neutral-500",
  mt: "bg-info-500/15 text-info-300 ring-1 ring-info-500/30",
  ai: "bg-tertiary-500/15 text-tertiary-300 ring-1 ring-tertiary-500/30",
  scrape: "bg-warning-500/15 text-warning-300 ring-1 ring-warning-500/30",
  manual: "bg-success-500/15 text-success-300 ring-1 ring-success-500/30",
};

interface Props {
  method: Method;
}

// "perlu dicek" used to render inline on every single scrape-sourced badge —
// with a whole page of scraped lines that was the same warning repeated a
// few dozen times. The row itself already gets an amber border for
// needs_review (see LineRow.tsx), and EditorPage now surfaces one summary
// disclaimer above the line list, so the per-badge text was pure noise.
export function MethodBadge({ method }: Props) {
  const { t } = useTranslation();
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
        {t(`method.${method}`)}
      </motion.span>
    </AnimatePresence>
  );
}
