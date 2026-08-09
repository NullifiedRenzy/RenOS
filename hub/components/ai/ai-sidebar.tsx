"use client";

import { useState, useRef, useEffect, useCallback } from "react";
import { useAIStore } from "@/stores/ai-store";
import { api } from "@/lib/api-client";
import { ChatMessageComponent } from "./chat-message";
import { SuggestedActions } from "./suggested-actions";
import type { ChatMessage, StreamChatResponse, SuggestedAction } from "@/lib/types";

export function AiSidebar() {
  const {
    isOpen, toggle, messages, conversationId, isLoading, suggestedActions,
    setMessages, addMessage, updateLastMessage, setConversationId,
    setLoading, setSuggestedActions, setProvider, startNew,
  } = useAIStore();
  const [input, setInput] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [showHistory, setShowHistory] = useState(false);
  const [conversations, setConversations] = useState<{ id: string; title: string }[]>([]);
  const listRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [messages]);

  useEffect(() => {
    if (isOpen && inputRef.current) {
      inputRef.current.focus();
    }
  }, [isOpen]);

  const loadConversations = useCallback(async () => {
    try {
      const res = await api.listConversations();
      setConversations(res.conversations ?? []);
    } catch {}
  }, []);

  const loadConversation = useCallback(async (id: string) => {
    try {
      const res = await api.getConversation(id);
      setConversationId(id);
      setMessages(res.messages);
      setShowHistory(false);
    } catch {}
  }, [setConversationId, setMessages]);

  const searchConversations = useCallback(async () => {
    if (!searchQuery.trim()) {
      loadConversations();
      return;
    }
    try {
      const res = await api.searchConversations(searchQuery);
      setConversations(res.conversations ?? []);
    } catch {}
  }, [searchQuery, loadConversations]);

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
      let newConvId = conversationId;
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
              newConvId = event.conversation_id;
              setConversationId(event.conversation_id);
              if (event.provider) setProvider(event.provider);
            } else if (event.type === "delta" && event.delta) {
              if (firstChunk) {
                addMessage({ role: "assistant", content: "" });
                firstChunk = false;
              }
              assistantContent += event.delta;
              updateLastMessage(assistantContent);
            } else if (event.type === "done") {
              if (event.usage) {
                // usage available but not displayed
              }
            }
          } catch {}
        }
      }
    } catch {
      addMessage({ role: "assistant", content: "Sorry, I encountered an error. Please check your AI provider configuration." });
    } finally {
      setLoading(false);
    }
  }, [input, isLoading, conversationId, addMessage, setLoading, setConversationId, setProvider, updateLastMessage, setSuggestedActions]);

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  if (!isOpen) {
    return (
      <button
        onClick={toggle}
        className="fixed bottom-6 right-6 z-50 w-12 h-12 rounded-full bg-[var(--color-accent)] text-white shadow-lg hover:opacity-90 transition-opacity flex items-center justify-center"
        title="AI Assistant"
      >
        <span className="text-lg">AI</span>
      </button>
    );
  }

  return (
    <div className="fixed right-0 top-0 bottom-0 z-50 w-[400px] max-w-[90vw] bg-[var(--color-bg)] border-l border-[var(--color-border)] flex flex-col shadow-2xl">
      <div className="flex items-center justify-between p-3 border-b border-[var(--color-border)]">
        <div className="flex items-center gap-2">
          <span className="text-body-medium font-medium text-[var(--color-text-primary)]">RenAI</span>
          <span className="text-[10px] px-1.5 py-0.5 rounded-sm bg-[var(--color-accent-tint)] text-[var(--color-accent)]">
            AI Assistant
          </span>
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => { setShowHistory(!showHistory); if (!showHistory) loadConversations(); }}
            className="p-1 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] text-sm"
            title="History"
          >
            &#128196;
          </button>
          <button
            onClick={startNew}
            className="p-1 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)] text-sm"
            title="New Chat"
          >
            +
          </button>
          <button
            onClick={toggle}
            className="p-1 text-[var(--color-text-tertiary)] hover:text-[var(--color-text-primary)]"
          >
            &times;
          </button>
        </div>
      </div>

      {showHistory && (
        <div className="border-b border-[var(--color-border)] p-3 max-h-60 overflow-y-auto">
          <input
            type="text"
            value={searchQuery}
            onChange={(e) => { setSearchQuery(e.target.value); searchConversations(); }}
            placeholder="Search conversations..."
            className="input w-full mb-2 text-[13px] py-1"
          />
          {conversations.map((c) => (
            <button
              key={c.id}
              onClick={() => loadConversation(c.id)}
              className="block w-full text-left text-[13px] text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)] py-1 px-2 rounded hover:bg-[var(--color-surface-elevated)] truncate"
            >
              {c.title}
            </button>
          ))}
          {conversations.length === 0 && (
            <p className="text-[12px] text-[var(--color-text-tertiary)] text-center py-2">No conversations</p>
          )}
        </div>
      )}

      <div ref={listRef} className="flex-1 overflow-y-auto p-4 space-y-4">
        {messages.length === 0 && !isLoading && (
          <div className="text-center py-12">
            <div className="w-12 h-12 rounded-full bg-[var(--color-accent-tint)] flex items-center justify-center mx-auto mb-3">
              <span className="text-[var(--color-accent)] text-xl font-bold">AI</span>
            </div>
            <p className="text-body-medium text-[var(--color-text-primary)] font-medium">RenAI Assistant</p>
            <p className="text-[13px] text-[var(--color-text-tertiary)] mt-1 max-w-[280px] mx-auto">
              Ask about your infrastructure, troubleshoot issues, or get recommendations.
            </p>
            <div className="mt-4 space-y-2 max-w-[300px] mx-auto">
              {[
                "What devices are offline?",
                "Show me recent failed jobs",
                "How do I restart nginx?",
                "Summarize my system health",
              ].map((suggestion) => (
                <button
                  key={suggestion}
                  onClick={() => { setInput(suggestion); }}
                  className="block w-full text-left text-[12px] text-[var(--color-text-secondary)] px-3 py-2 border border-[var(--color-border)] rounded-md hover:bg-[var(--color-surface-elevated)] transition-colors"
                >
                  {suggestion}
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
                <span className="w-1.5 h-1.5 bg-[var(--color-text-tertiary)] rounded-full animate-bounce" />
                <span className="w-1.5 h-1.5 bg-[var(--color-text-tertiary)] rounded-full animate-bounce" style={{ animationDelay: "0.1s" }} />
                <span className="w-1.5 h-1.5 bg-[var(--color-text-tertiary)] rounded-full animate-bounce" style={{ animationDelay: "0.2s" }} />
              </div>
            </div>
          </div>
        )}

        {!isLoading && suggestedActions.length > 0 && (
          <SuggestedActions actions={suggestedActions} />
        )}
      </div>

      <div className="border-t border-[var(--color-border)] p-3">
        <div className="flex gap-2">
          <input
            ref={inputRef}
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={handleKeyDown}
            placeholder="Ask RenAI..."
            disabled={isLoading}
            className="flex-1 text-[13px] px-3 py-2 border border-[var(--color-border)] rounded-md bg-[var(--color-surface)] text-[var(--color-text-primary)] placeholder:text-[var(--color-text-tertiary)] focus:outline-none focus:border-[var(--color-accent)] disabled:opacity-50"
          />
          <button
            onClick={handleSend}
            disabled={isLoading || !input.trim()}
            className="px-3 py-2 bg-[var(--color-accent)] text-white text-[13px] rounded-md hover:opacity-90 disabled:opacity-50 transition-opacity"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  );
}
