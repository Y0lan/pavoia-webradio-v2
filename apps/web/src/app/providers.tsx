"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, useEffect, type ReactNode } from "react";
import { PlayerBar } from "@/components/player-bar";
import { Sidebar } from "@/components/sidebar";
import { useWSStore } from "@/lib/ws";
import { STAGES } from "@/lib/stages";

// Global WS subscriber. Without this, only pages that explicitly call
// subscribe() (home, stage/[id]) receive now-playing events — the
// dashboard's "WHAT'S PLAYING NOW" and the player bar got "WAITING" forever
// on a cold page visit because the WS was connected but had no stages
// registered. Subscribing to all 9 stages globally matches what the
// radical-transparency UX promises (every page shows live activity).
function WSConnector() {
  const connect = useWSStore((s) => s.connect);
  const subscribe = useWSStore((s) => s.subscribe);
  useEffect(() => {
    connect();
    subscribe(STAGES.map((s) => s.id));
  }, [connect, subscribe]);
  return null;
}

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 30_000,
            refetchOnWindowFocus: false,
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      <WSConnector />
      <Sidebar />
      <div className="min-[900px]:ml-[220px] pb-[72px]">
        {children}
      </div>
      <PlayerBar />
    </QueryClientProvider>
  );
}
