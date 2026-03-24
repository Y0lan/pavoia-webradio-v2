import { useEffect, useRef } from "react";

const MIN_SWIPE_X = 50;
const MAX_DRIFT_Y = 30;
const EDGE_DEAD_ZONE = 30;

const INTERACTIVE_TAGS = new Set(["BUTTON", "A", "INPUT", "SELECT", "TEXTAREA"]);

function isInteractive(el) {
  let node = el;
  while (node && node !== document.body) {
    if (INTERACTIVE_TAGS.has(node.tagName) || node.getAttribute("role") === "button") {
      return true;
    }
    node = node.parentElement;
  }
  return false;
}

export function useSwipeNavigation(ref, { onPrev, onNext }) {
  const start = useRef(null);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    el.style.touchAction = "pan-y";

    function handleTouchStart(e) {
      const touch = e.touches[0];

      if (
        touch.clientX < EDGE_DEAD_ZONE ||
        touch.clientX > window.innerWidth - EDGE_DEAD_ZONE
      ) {
        start.current = null;
        return;
      }

      if (isInteractive(e.target)) {
        start.current = null;
        return;
      }

      start.current = { x: touch.clientX, y: touch.clientY };
    }

    function handleTouchMove(e) {
      if (!start.current) return;

      const touch = e.touches[0];
      const dy = Math.abs(touch.clientY - start.current.y);

      if (dy > MAX_DRIFT_Y) {
        start.current = null;
      }
    }

    function handleTouchEnd(e) {
      if (!start.current) return;

      const touch = e.changedTouches[0];
      const dx = touch.clientX - start.current.x;
      const dy = Math.abs(touch.clientY - start.current.y);

      start.current = null;

      if (Math.abs(dx) < MIN_SWIPE_X || dy > MAX_DRIFT_Y) return;

      if (dx > 0) {
        onPrev?.();
      } else {
        onNext?.();
      }
    }

    el.addEventListener("touchstart", handleTouchStart, { passive: true });
    el.addEventListener("touchmove", handleTouchMove, { passive: true });
    el.addEventListener("touchend", handleTouchEnd, { passive: true });

    return () => {
      el.removeEventListener("touchstart", handleTouchStart);
      el.removeEventListener("touchmove", handleTouchMove);
      el.removeEventListener("touchend", handleTouchEnd);
    };
  }, [ref, onPrev, onNext]);
}
