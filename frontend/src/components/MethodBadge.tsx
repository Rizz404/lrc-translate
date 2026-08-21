import type { Method } from "../api/types";

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
    <span className={`method-badge method-${method}${needsReview ? " needs-review" : ""}`}>
      {LABELS[method]}
      {needsReview && " · perlu dicek"}
    </span>
  );
}
