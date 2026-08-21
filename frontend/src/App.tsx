import { useState } from "react";
import { SearchPage } from "./pages/SearchPage";
import { EditorPage } from "./pages/EditorPage";
import "./App.css";

export default function App() {
  const [trackId, setTrackId] = useState<string | null>(null);

  return trackId ? (
    <EditorPage trackId={trackId} onBack={() => setTrackId(null)} />
  ) : (
    <SearchPage onImported={setTrackId} />
  );
}
