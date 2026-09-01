import { useEffect, useRef, useState } from "react";

/**
 * Seconds elapsed since `active` most recently became true, ticking once a
 * second while it stays true. Used to show "how long has this been running"
 * next to a long-running mutation's spinner (see TranslatePanel.tsx).
 *
 * Deliberately does NOT reset to 0 the instant `active` goes false — it
 * freezes at whatever the count reached, so the caller can still show "that
 * took 1:47" once the run finishes, not just while it's in progress (a
 * counter nobody gets to read the final value of is pointless). It only
 * resets back to 0 when a *new* run actually starts (active flips false ->
 * true again).
 *
 * Recomputed each tick from wall-clock time (Date.now() - start) rather than
 * just counting ticks, so a backgrounded tab throttling setInterval doesn't
 * make the counter drift behind reality — it jumps to the correct value as
 * soon as the next tick does fire, instead of silently under-reporting.
 */
export function useElapsedSeconds(active: boolean): number {
  const [elapsed, setElapsed] = useState(0);
  const startRef = useRef<number | null>(null);

  useEffect(() => {
    if (!active) {
      // Freeze — see doc comment above. startRef is cleared so a stray
      // interval tick from a race right at this transition can't sneak in
      // one more update after the fact.
      startRef.current = null;
      return;
    }

    startRef.current = Date.now();
    setElapsed(0); // reset only here, at the start of a new run
    const id = setInterval(() => {
      if (startRef.current != null) {
        setElapsed(Math.floor((Date.now() - startRef.current) / 1000));
      }
    }, 1000);
    return () => clearInterval(id);
  }, [active]);

  return elapsed;
}

/** "mm:ss", e.g. 75 -> "1:15". */
export function formatElapsed(seconds: number): string {
  const m = Math.floor(seconds / 60);
  const s = seconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}
