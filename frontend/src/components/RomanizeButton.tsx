import { useMutation } from "@tanstack/react-query";
import { motion } from "motion/react";
import { CaseSensitive, Loader2 } from "lucide-react";
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
    <div className="flex items-center gap-2">
      <motion.button
        whileTap={{ scale: 0.95 }}
        onClick={() => mutation.mutate()}
        disabled={mutation.isPending}
        className="inline-flex items-center gap-2 rounded-lg border border-slate-800 bg-slate-900/60 px-3.5 py-2 text-sm font-medium text-slate-300 transition-colors hover:border-slate-700 hover:text-white disabled:cursor-not-allowed disabled:opacity-50"
      >
        {mutation.isPending ? (
          <Loader2 className="size-4 animate-spin" />
        ) : (
          <CaseSensitive className="size-4" />
        )}
        Romanize
      </motion.button>
      {mutation.isError && (
        <span className="text-xs text-rose-400">{(mutation.error as Error).message}</span>
      )}
    </div>
  );
}
