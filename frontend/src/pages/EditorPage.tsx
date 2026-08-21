import { useEffect } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "../api/client";
import { useEditorStore } from "../store/editorStore";
import { LineRow } from "../components/LineRow";
import { CopyDownloadBar } from "../components/CopyDownloadBar";
import { TranslatePanel } from "../components/TranslatePanel";
import { RomanizeButton } from "../components/RomanizeButton";

interface Props {
  trackId: string;
  onBack: () => void;
}

export function EditorPage({ trackId, onBack }: Props) {
  const { data: fetchedTrack, isLoading, error } = useQuery({
    queryKey: ["track", trackId],
    queryFn: () => api.getTrack(trackId),
  });

  const track = useEditorStore((s) => s.track);
  const setTrack = useEditorStore((s) => s.setTrack);

  useEffect(() => {
    if (fetchedTrack) setTrack(fetchedTrack);
    return () => setTrack(null);
  }, [fetchedTrack, setTrack]);

  if (isLoading) return <p>Memuat…</p>;
  if (error) return <p className="error">{(error as Error).message}</p>;
  if (!track) return null;

  return (
    <div className="editor-page">
      <button className="link-button" onClick={onBack}>
        ← Cari lagu lain
      </button>
      <h1>
        {track.title} — {track.artist}
      </h1>

      <CopyDownloadBar track={track} />

      <div className="editor-actions">
        {track.language === "ja" && <RomanizeButton trackId={track.id} />}
        <TranslatePanel trackId={track.id} />
      </div>

      <div className="line-list">
        {track.lines.map((line) => (
          <LineRow key={line.id} trackId={track.id} line={line} />
        ))}
      </div>
    </div>
  );
}
