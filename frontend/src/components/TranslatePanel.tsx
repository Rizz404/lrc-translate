import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";

interface Props {
  trackId: string;
}

const LANGS = [
  { code: "id", label: "Indonesia" },
  { code: "en", label: "English" },
  { code: "ja", label: "日本語" },
];

/** Cabang A: kirim semua baris ke LibreTranslate sekaligus (plan.md #5). */
export function TranslatePanel({ trackId }: Props) {
  const [targetLang, setTargetLang] = useState("id");
  const patchLine = useEditorStore((s) => s.patchLine);

  const mutation = useMutation({
    mutationFn: () => api.translateTrack(trackId, { target_lang: targetLang }),
    onSuccess: (resp) => {
      for (const line of resp.lines) patchLine(line.id, line);
    },
  });

  return (
    <div className="translate-panel">
      <select value={targetLang} onChange={(e) => setTargetLang(e.target.value)}>
        {LANGS.map((l) => (
          <option key={l.code} value={l.code}>
            {l.label}
          </option>
        ))}
      </select>
      <button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
        {mutation.isPending ? "Menerjemahkan…" : "Translate All (MT)"}
      </button>

      {mutation.isSuccess && (
        <span className="muted small">
          {mutation.data.cache_hits} dari cache, {mutation.data.cache_misses} baru
          {mutation.data.failed?.length ? `, ${mutation.data.failed.length} gagal` : ""}
        </span>
      )}
      {mutation.isError && <span className="error small">{(mutation.error as Error).message}</span>}
    </div>
  );
}
