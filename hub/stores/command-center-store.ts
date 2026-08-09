import { create } from "zustand";

export type CommandCenterMode = "search" | "command";

interface CommandCenterState {
  isOpen: boolean;
  mode: CommandCenterMode;
  open: (mode?: CommandCenterMode) => void;
  close: () => void;
  toggle: () => void;
  setMode: (mode: CommandCenterMode) => void;
}

export const useCommandCenterStore = create<CommandCenterState>((set, get) => ({
  isOpen: false,
  mode: "search",

  open: (mode) => set({ isOpen: true, mode: mode ?? "search" }),

  close: () => set({ isOpen: false }),

  toggle: () => set((s) => ({ isOpen: !s.isOpen })),

  setMode: (mode) => set({ mode }),
}));
