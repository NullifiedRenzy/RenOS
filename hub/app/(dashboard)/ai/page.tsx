"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { useAIStore } from "@/stores/ai-store";
import { useConversations } from "@/hooks/use-ai";
import { api } from "@/lib/api-client";
import { ChatMessageComponent } from "@/components/ai/chat-message";
import { SuggestedActions } from "@/components/ai/suggested-actions";
import type { ChatMessage, StreamChatResponse } from "@/lib/types";

export default function AIPage() {
  const {
    messages, conversationId, isLoading, suggestedActions,
    addMessage, updateLastMessage, setConversationId,
    setLoading, setSuggestedActions, setProvider, startNew, setMessages,
  } = useAIStore();
  const { data: convData } = useConversations();
  const [input, setInput] = useState("");
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [selectedConvId, setSelectedConvId] = useState<string | null>(null);
  const listRef = useRef<HTMLDivElement>(null);

  const conversations = convData?.conversations ?? [];

  useEffect(() => {
    if (selectedConvId) {
      api.getConversation(selectedConvId).then((conv) => {
        setConversationId(selectedConvId);
        setMessages(conv.messages);
        setSelectedConvId(null);
        setSidebarOpen(false);
      }).catch(() => {});
    }
  }, [selectedConvId, setConversationId, setMessages]);

  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSend = useCallback(async () => {
    const trimmed = input.trim();
    if (!trimmed || isLoading) return;
    setInput("");
    setSuggestedActions([]);

    const userMsg: ChatMessage = { role: "user", content: trimmed };
    addMessage(userMsg);
    setLoading(true);

    try {
      const stream = await api.streamChat(trimmed, conversationId ?? undefined);
      const reader = stream.getReader();
      const decoder = new TextDecoder();
      let assistantContent = "";
      let firstChunk = true;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        const text = decoder.decode(value, { stream: true });
        const lines = text.split("\n").filter(Boolean);

        for (const line of lines) {
          try {
            const event: StreamChatResponse = JSON.parse(line);
            if (event.type === "started" && event.conversation_id) {
              setConversationId(event.conversation_id);
              if (event.provider) setProvider(event.provider);
            } else if (event.type === "delta" && event.delta) {
              if (firstChunk) {
                addMessage({ role: "assistant", content: "" });
                firstChunk = false;
              }
              assistantContent += event.delta;
              updateLastMessage(assistantContent);
            }
          } catch {}
        }
      }
    } catch {
      addMessage({ role: "assistant", content: "Sorry, I encountered an error." });
    } finally {
      setLoading(false);
    }
  }, [input, isLoading, conversationId, addMessage, setLoading, setConversationId, setProvider, updateLastMessage, setSuggestedActions]);

  return (
    <div className="flex h-[calc(100vh-4rem)]">
      <div className={`${sidebarOpen ? "block" : "hidden"} md:block w-64 border-r border-[var(--color-border)] p-4 overflow-y-auto`}>
        <button onClick={() => { startNew(); setSidebarOpen(false); }} className="btn-primary w-full mb-4 text-body-small py-1.5">
          New Chat
        </button>
        {conversations.map((c) => (
          <button
            key={c.id}
            onClick={() => setSelectedConvId(c.id)}
            className="block w-full text-left text-body-small text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] py-1.5 px-2 rounded hover:bg-[var(--color-surface-elevated)] truncate"
          >
            {c.title}
          </button>
        ))}
      </div>

      <div className="flex-1 flex flex-col">
        <div className="md:hidden p-3 border-b border-[var(--color-border)]">
          <button onClick={() => setSidebarOpen(!sidebarOpen)} className="text-body-small">
            {sidebarOpen ? "Close" : "Conversations"}
          </button>
        </div>

        <div ref={listRef} className="flex-1 overflow-y-auto p-4 space-y-4">
          {messages.length === 0 && (
            <div className="text-center py-16">
              <div className="w-16 h-16 rounded-full bg-[var(--color-accent-tint)] flex items-center justify-center mx-auto mb-4">
                <span className="text-[var(--color-accent)] text-2xl font-bold">AI</span>
              </div>
              <p className="text-headline-4 text-[var(--color-text-primary)]">RenAI Assistant</p>
              <p className="text-body text-[var(--color-text-tertiary)] mt-2 max-w-md mx-auto">
                Ask me about your devices, troubleshoot issues, or get infrastructure recommendations.
              </p>
              <div className="mt-6 grid grid-cols-2 gap-3 max-w-lg mx-auto">
                {[
                  "What devices are offline?",
                  "Show recent failed jobs",
                  "How do I restart nginx?",
                  "Summarize system health",
                ].map((s) => (
                  <button
                    key={s}
                    onClick={() => setInput(s)}
                    className="text-left text-[13px] text-[var(--color-text-secondary)] p-3 border border-[var(--color-border)] rounded-lg hover:bg-[var(--color-surface-elevated)] transition-colors"
                  >
                    {s}
                  </button>
                ))}
              </div>
            </div>
          )}

          {messages.map((msg, i) => (
            <ChatMessageComponent key={i} message={msg} />
          ))}

          {isLoading && messages.length > 0 && messages[messages.length - 1].role === "user" && (
            <div className="flex justify-start">
              <div className="w-7 h-7 rounded-full bg-[var(--color-accent-tint)] flex items-center justify-center flex-shrink-0 mr-2">
                <span className="text-[var(--color-accent)] text-xs font-bold">AI</span>
              </div>
              <div className="bg-[var(--color-surface-elevated)] rounded-lg px-4 py-2">
                <div className="flex gap-1">
                  <span className="w-2 h-2 bg-[var(--color-text-tertiary)] rounded-full animate-bounce" />
                  <span className="w-2 h-2 bg-[var(--color-text-tertiary)] rounded-full animate-bounce" style={{ animationDelay: "0.1s" }} />
                  <span className="w-2 h-2 bg-[var(--color-text-tertiary)] rounded-full animate-bounce" style={{ animationDelay: "0.2s" }} />
                </div>
              </div>
            </div>
          )}

          {!isLoading && suggestedActions.length > 0 && (
            <SuggestedActions actions={suggestedActions} />
          )}
        </div>

        <form onSubmit={(e) => { e.preventDefault(); handleSend(); }} className="border-t border-[var(--color-border)] p-4">
          <div className="flex gap-2">
            <input
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder="Ask anything..."
              disabled={isLoading}
              className="input flex-1"
            />
            <button type="submit" disabled={isLoading || !input.trim()} className="btn-primary">
              Send
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
