"use client";

import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { useState, useEffect, type ReactNode } from "react";
import { PlayerBar } from "@/components/player-bar";
import { useWSStore } from "@/lib/ws";

function WSConnector() {
  const connect = useWSStore((s) => s.connect);
  useEffect(() => {
    connect();
  }, [connect]);
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
      <div className="pb-[72px]">
        {children}
      </div>
      <PlayerBar />
    </QueryClientProvider>
  );
}
