import { useMutation } from "@tanstack/react-query";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";

interface Props {
  trackId: string;
}

export function RomanizeButton({ trackId }: Props) {
  const patchLine = useEditorStore((s) => s.patchLine);

  const mutation = useMutation({
    mutationFn: () => api.romanizeTrack(trackId),
    onSuccess: (resp) => {
      for (const line of resp.lines) patchLine(line.id, line);
    },
  });

  return (
    <div className="romanize-action">
      <button onClick={() => mutation.mutate()} disabled={mutation.isPending}>
        {mutation.isPending ? "Romanizing…" : "Romanize (Jepang → Romaji)"}
      </button>
      {mutation.isError && <span className="error small">{(mutation.error as Error).message}</span>}
    </div>
  );
}
