import { useEffect, useRef, useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { Line } from "../api/types";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";
import { MethodBadge } from "./MethodBadge";

interface Props {
  trackId: string;
  line: Line;
}

const AUTOSAVE_DEBOUNCE_MS = 800;

export function LineRow({ trackId, line }: Props) {
  const patchLine = useEditorStore((s) => s.patchLine);
  const [original, setOriginal] = useState(line.original);
  const [timestamp, setTimestamp] = useState(line.timestamp);
  const [translation, setTranslation] = useState(line.translation);
  const [error, setError] = useState<string | null>(null);

  // Keep local text in sync when the store's line changes from elsewhere
  // (e.g. after a batch Translate All or Romanize call).
  useEffect(() => setTranslation(line.translation), [line.translation]);

  const updateMutation = useMutation({
    mutationFn: (body: { original?: string; timestamp?: string; translation?: string }) =>
      api.updateLine(trackId, line.id, body),
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

  function saveOriginalIfChanged() {
    const patch: { original?: string; timestamp?: string } = {};
    if (original !== line.original) patch.original = original;
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

  return (
    <div className={`line-row${line.needs_review ? " needs-review" : ""}`}>
      <input
        className="line-timestamp"
        value={timestamp}
        onChange={(e) => setTimestamp(e.target.value)}
        onBlur={saveOriginalIfChanged}
      />
      <div className="line-texts">
        <textarea
          className="line-original"
          value={original}
          onChange={(e) => setOriginal(e.target.value)}
          onBlur={saveOriginalIfChanged}
          rows={1}
        />
        {line.romanized && <div className="line-romanized muted small">{line.romanized}</div>}
        <div className="line-translation-row">
          <textarea
            className="line-translation"
            placeholder="Terjemahan…"
            value={translation}
            onChange={(e) => handleTranslationChange(e.target.value)}
            rows={1}
          />
          <MethodBadge method={line.method} needsReview={line.needs_review} />
          {line.method === "manual" && (
            <button
              className="link-button small"
              onClick={() => revertMutation.mutate()}
              disabled={revertMutation.isPending}
              title="Kembalikan ke saran awal"
            >
              revert
            </button>
          )}
        </div>
      </div>
      {(updateMutation.isPending || revertMutation.isPending) && (
        <span className="muted small">Menyimpan…</span>
      )}
      {error && <span className="error small">{error}</span>}
    </div>
  );
}
