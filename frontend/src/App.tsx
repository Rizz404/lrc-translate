import { useState } from "react";
import { AnimatePresence, motion } from "motion/react";
import { SearchPage } from "./pages/SearchPage";
import { EditorPage } from "./pages/EditorPage";

export default function App() {
  const [trackId, setTrackId] = useState<string | null>(null);

  return (
    <AnimatePresence mode="wait">
      {trackId ? (
        <motion.div
          key="editor"
          initial={{ opacity: 0, x: 16 }}
          animate={{ opacity: 1, x: 0 }}
          exit={{ opacity: 0, x: -16 }}
          transition={{ duration: 0.25, ease: "easeOut" }}
        >
          <EditorPage trackId={trackId} onBack={() => setTrackId(null)} />
        </motion.div>
      ) : (
        <motion.div
          key="search"
          initial={{ opacity: 0, x: -16 }}
          animate={{ opacity: 1, x: 0 }}
          exit={{ opacity: 0, x: 16 }}
          transition={{ duration: 0.25, ease: "easeOut" }}
        >
          <SearchPage onImported={setTrackId} />
        </motion.div>
      )}
    </AnimatePresence>
  );
}
