"use client";

import { useState, useCallback } from "react";
import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";
import type { ChatMessage } from "@/lib/types";

export function useConversations() {
  return useQuery({
    queryKey: ["ai-conversations"],
    queryFn: () => api.listConversations(),
  });
}

export function useConversation(id: string | null) {
  return useQuery({
    queryKey: ["ai-conversation", id],
    queryFn: () => api.getConversation(id!),
    enabled: !!id,
  });
}

export function useChat() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [conversationId, setConversationId] = useState<string | null>(null);
  const [isLoading, setIsLoading] = useState(false);

  const send = useCallback(async (content: string) => {
    const userMsg: ChatMessage = { role: "user", content };
    setMessages(prev => [...prev, userMsg]);
    setIsLoading(true);

    try {
      const res = await api.chat(content, conversationId ?? undefined);
      setConversationId(res.conversation_id);
      const replyMsg: ChatMessage = { role: "assistant", content: res.reply };
      setMessages(prev => [...prev, replyMsg]);
    } catch {
      const errMsg: ChatMessage = { role: "assistant", content: "Sorry, I couldn't process that request." };
      setMessages(prev => [...prev, errMsg]);
    } finally {
      setIsLoading(false);
    }
  }, [conversationId]);

  const loadConversation = useCallback((id: string, msgs: ChatMessage[]) => {
    setConversationId(id);
    setMessages(msgs);
  }, []);

  const startNew = useCallback(() => {
    setConversationId(null);
    setMessages([]);
  }, []);

  return { messages, conversationId, isLoading, send, loadConversation, startNew };
}
