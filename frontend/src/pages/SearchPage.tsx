import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import type { SearchResult } from "../api/types";
import { api } from "../api/client";
import { SongCandidateList } from "../components/SongCandidateList";

interface Props {
  onImported: (trackId: string) => void;
}

export function SearchPage({ onImported }: Props) {
  const [title, setTitle] = useState("");
  const [artist, setArtist] = useState("");
  const [results, setResults] = useState<SearchResult[] | null>(null);
  const [importingId, setImportingId] = useState<number | null>(null);

  const searchMutation = useMutation({
    mutationFn: () => api.search(title, artist),
    onSuccess: setResults,
  });

  const importMutation = useMutation({
    mutationFn: (lrclibId: number) => api.importTrack(lrclibId),
    onSuccess: (track) => onImported(track.id),
    onSettled: () => setImportingId(null),
  });

  function handleSelect(result: SearchResult) {
    setImportingId(result.lrclib_id);
    importMutation.mutate(result.lrclib_id);
  }

  return (
    <div className="search-page">
      <h1>Cari Lagu</h1>
      <form
        onSubmit={(e) => {
          e.preventDefault();
          searchMutation.mutate();
        }}
      >
        <input placeholder="Judul lagu" value={title} onChange={(e) => setTitle(e.target.value)} />
        <input placeholder="Artist" value={artist} onChange={(e) => setArtist(e.target.value)} />
        <button type="submit" disabled={searchMutation.isPending || (!title && !artist)}>
          {searchMutation.isPending ? "Mencari…" : "Cari"}
        </button>
      </form>

      {searchMutation.isError && <p className="error">{(searchMutation.error as Error).message}</p>}
      {importMutation.isError && <p className="error">{(importMutation.error as Error).message}</p>}

      {results && (
        <SongCandidateList results={results} onSelect={handleSelect} pendingId={importingId} />
      )}
    </div>
  );
}
