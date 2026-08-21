import { Routes, Route, useLocation } from "react-router-dom";
import { AnimatePresence, motion } from "motion/react";
import { SearchPage } from "./pages/SearchPage";
import { EditorPage } from "./pages/EditorPage";

export default function App() {
  const location = useLocation();
  const isEditor = location.pathname.startsWith("/track/");

  return (
    <AnimatePresence mode="wait">
      <motion.div
        key={location.pathname}
        initial={{ opacity: 0, x: isEditor ? 16 : -16 }}
        animate={{ opacity: 1, x: 0 }}
        exit={{ opacity: 0, x: isEditor ? -16 : 16 }}
        transition={{ duration: 0.25, ease: "easeOut" }}
      >
        <Routes location={location}>
          <Route path="/" element={<SearchPage />} />
          <Route path="/track/:trackId" element={<EditorPage />} />
        </Routes>
      </motion.div>
    </AnimatePresence>
  );
}
