"use client";

import { create } from "zustand";
import type { ChatMessage, SuggestedAction } from "@/lib/types";

interface AIState {
  isOpen: boolean;
  messages: ChatMessage[];
  conversationId: string | null;
  isLoading: boolean;
  suggestedActions: SuggestedAction[];
  provider: string;
  toggle: () => void;
  setOpen: (open: boolean) => void;
  setMessages: (messages: ChatMessage[]) => void;
  addMessage: (message: ChatMessage) => void;
  updateLastMessage: (content: string) => void;
  setConversationId: (id: string | null) => void;
  setLoading: (loading: boolean) => void;
  setSuggestedActions: (actions: SuggestedAction[]) => void;
  setProvider: (provider: string) => void;
  startNew: () => void;
}

export const useAIStore = create<AIState>((set) => ({
  isOpen: false,
  messages: [],
  conversationId: null,
  isLoading: false,
  suggestedActions: [],
  provider: "",
  toggle: () => set((s) => ({ isOpen: !s.isOpen })),
  setOpen: (open) => set({ isOpen: open }),
  setMessages: (messages) => set({ messages }),
  addMessage: (message) => set((s) => ({ messages: [...s.messages, message] })),
  updateLastMessage: (content) =>
    set((s) => {
      const msgs = [...s.messages];
      if (msgs.length > 0 && msgs[msgs.length - 1].role === "assistant") {
        msgs[msgs.length - 1] = { ...msgs[msgs.length - 1], content };
      }
      return { messages: msgs };
    }),
  setConversationId: (conversationId) => set({ conversationId }),
  setLoading: (isLoading) => set({ isLoading }),
  setSuggestedActions: (suggestedActions) => set({ suggestedActions }),
  setProvider: (provider) => set({ provider }),
  startNew: () =>
    set({
      conversationId: null,
      messages: [],
      suggestedActions: [],
      isLoading: false,
    }),
}));
