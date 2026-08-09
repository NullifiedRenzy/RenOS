"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api-client";

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
