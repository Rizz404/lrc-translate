import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { Line } from "../api/types";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";

interface Props {
  trackId: string;
  line: Line;
}

/**
 * Milestone 1 scope: editable original lyric text + timestamp only.
 * Milestone 2 adds romanized/translation side-by-side display, the method
 * badge, and the revert-to-suggestion action.
 */
export function LineRow({ trackId, line }: Props) {
  const patchLine = useEditorStore((s) => s.patchLine);
  const [original, setOriginal] = useState(line.original);
  const [timestamp, setTimestamp] = useState(line.timestamp);
  const [error, setError] = useState<string | null>(null);

  const mutation = useMutation({
    mutationFn: (body: { original?: string; timestamp?: string }) =>
      api.updateLine(trackId, line.id, body),
    onSuccess: (updated) => {
      patchLine(line.id, updated);
      setError(null);
    },
    onError: (err: Error) => setError(err.message),
  });

  function saveIfChanged() {
    const patch: { original?: string; timestamp?: string } = {};
    if (original !== line.original) patch.original = original;
    if (timestamp !== line.timestamp) patch.timestamp = timestamp;
    if (Object.keys(patch).length > 0) mutation.mutate(patch);
  }

  return (
    <div className="line-row">
      <input
        className="line-timestamp"
        value={timestamp}
        onChange={(e) => setTimestamp(e.target.value)}
        onBlur={saveIfChanged}
      />
      <textarea
        className="line-original"
        value={original}
        onChange={(e) => setOriginal(e.target.value)}
        onBlur={saveIfChanged}
        rows={1}
      />
      {mutation.isPending && <span className="muted small">Menyimpan…</span>}
      {error && <span className="error small">{error}</span>}
    </div>
  );
}
