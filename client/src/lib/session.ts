import { create } from "zustand";

import type { ConsoleSession } from "../api/types";

type SessionStatus = "unknown" | "authenticated" | "anonymous";

type SessionState = {
  session: ConsoleSession | null;
  status: SessionStatus;
  authenticated: boolean;
  setSession: (session: ConsoleSession) => void;
  clearSession: () => void;
};

export const useSessionStore = create<SessionState>((set) => ({
  session: null,
  status: "unknown",
  authenticated: false,
  setSession: (session) =>
    set({
      session,
      status: "authenticated",
      authenticated: true,
    }),
  clearSession: () =>
    set({
      session: null,
      status: "anonymous",
      authenticated: false,
    }),
}));
