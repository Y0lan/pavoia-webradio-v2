import React from "react";

export function BusMysteryCard({ open, onClose }) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-[60] pointer-events-auto" onClick={onClose}>
      <div className="absolute inset-0 bg-gradient-to-b from-[#3d2a0a] via-[#2a1a08] to-[#1a1004]" />
      <div className="absolute inset-0 flex flex-col items-center justify-center p-8 text-center">
        <div className="text-8xl mb-8">🚌</div>
        <h2 className="text-3xl md:text-5xl font-bold text-white mb-4">
          Some things must be<br />experienced in person
        </h2>
        <p className="text-white/50 text-lg max-w-md">
          The Bus is out there, somewhere at Pavoia. Find it, hop on, and let the music surprise you.
        </p>
        <p className="mt-8 text-white/30 text-sm">tap anywhere to close</p>
      </div>
    </div>
  );
}
