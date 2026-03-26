import { create } from "zustand";

const BRIDGE_URL = process.env.NEXT_PUBLIC_BRIDGE_URL || "http://localhost:3001";
const WS_URL = BRIDGE_URL.replace(/^http/, "ws") + "/ws";

export interface NowPlaying {
  stage_id: string;
  status: string;
  title: string;
  artist: string;
  album: string;
  elapsed?: string;
  duration?: string;
  file?: string;
}

export interface StageInfo {
  id: string;
  name: string;
  genre: string;
  color: string;
  nowPlaying: NowPlaying | null;
  alive: boolean;
}

interface WSState {
  connected: boolean;
  stages: Map<string, StageInfo>;
  subscribedStages: Set<string>;

  connect: () => void;
  disconnect: () => void;
  subscribe: (stageIds: string[]) => void;
  unsubscribe: (stageIds: string[]) => void;
}

let ws: WebSocket | null = null;
let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
let reconnectDelay = 1000;

export const useWSStore = create<WSState>((set, get) => ({
  connected: false,
  stages: new Map(),
  subscribedStages: new Set(),

  connect: () => {
    if (ws && ws.readyState <= WebSocket.OPEN) return;

    try {
      ws = new WebSocket(WS_URL);
    } catch {
      scheduleReconnect(get);
      return;
    }

    ws.onopen = () => {
      set({ connected: true });
      reconnectDelay = 1000;

      // Re-subscribe to previously subscribed stages
      const subs = get().subscribedStages;
      if (subs.size > 0) {
        ws?.send(JSON.stringify({ type: "subscribe", stages: Array.from(subs) }));
      }
    };

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === "now_playing" && msg.data) {
          const np = msg.data as NowPlaying;
          set((state) => {
            const stages = new Map(state.stages);
            const existing = stages.get(np.stage_id);
            if (existing) {
              stages.set(np.stage_id, { ...existing, nowPlaying: np, alive: true });
            } else {
              stages.set(np.stage_id, {
                id: np.stage_id,
                name: np.stage_id,
                genre: "",
                color: "#00ffc8",
                nowPlaying: np,
                alive: true,
              });
            }
            return { stages };
          });
        }
      } catch {
        // ignore malformed messages
      }
    };

    ws.onclose = () => {
      set({ connected: false });
      ws = null;
      scheduleReconnect(get);
    };

    ws.onerror = () => {
      ws?.close();
    };
  },

  disconnect: () => {
    if (reconnectTimer) clearTimeout(reconnectTimer);
    ws?.close();
    ws = null;
    set({ connected: false });
  },

  subscribe: (stageIds: string[]) => {
    set((state) => {
      const subs = new Set(state.subscribedStages);
      stageIds.forEach((id) => subs.add(id));
      return { subscribedStages: subs };
    });
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "subscribe", stages: stageIds }));
    }
  },

  unsubscribe: (stageIds: string[]) => {
    set((state) => {
      const subs = new Set(state.subscribedStages);
      stageIds.forEach((id) => subs.delete(id));
      return { subscribedStages: subs };
    });
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "unsubscribe", stages: stageIds }));
    }
  },
}));

function scheduleReconnect(get: () => WSState) {
  if (reconnectTimer) return;
  reconnectTimer = setTimeout(() => {
    reconnectTimer = null;
    reconnectDelay = Math.min(reconnectDelay * 2, 30000);
    get().connect();
  }, reconnectDelay);
}
