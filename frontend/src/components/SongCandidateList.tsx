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

export function SongCandidateList({ results, onSelect, pendingId }: Props) {
  if (results.length === 0) {
    return <p className="muted">Tidak ada hasil.</p>;
  }

  return (
    <ul className="candidate-list">
      {results.map((r) => (
        <li key={r.lrclib_id} className="candidate-row">
          <div className="candidate-info">
            <strong>{r.title}</strong> — {r.artist}
            <div className="muted">
              {r.album || "(no album)"} · {formatDuration(r.duration_ms)}
              {!r.has_synced_lyrics && " · tanpa synced lyrics"}
            </div>
          </div>
          <button
            disabled={!r.has_synced_lyrics || pendingId === r.lrclib_id}
            onClick={() => onSelect(r)}
          >
            {pendingId === r.lrclib_id ? "Mengambil…" : "Pilih"}
          </button>
        </li>
      ))}
    </ul>
  );
}
