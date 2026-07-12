"use client";

import { useState, useRef, useEffect } from "react";
import { useChat, useConversations, useConversation } from "@/hooks/use-ai";
import type { ConversationSummary } from "@/lib/types";

export default function AIPage() {
  const { messages, conversationId, isLoading, send, loadConversation, startNew } = useChat();
  const { data: convData } = useConversations();
  const [input, setInput] = useState("");
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [selectedConvId, setSelectedConvId] = useState<string | null>(null);
  const { data: convDetail } = useConversation(selectedConvId);
  const listRef = useRef<HTMLDivElement>(null);

  const conversations = convData?.conversations ?? [];

  useEffect(() => {
    if (convDetail && selectedConvId) {
      loadConversation(selectedConvId, convDetail.messages);
      setSelectedConvId(null);
    }
  }, [convDetail, selectedConvId, loadConversation]);

  useEffect(() => {
    if (listRef.current) {
      listRef.current.scrollTop = listRef.current.scrollHeight;
    }
  }, [messages]);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = input.trim();
    if (!trimmed || isLoading) return;
    setInput("");
    send(trimmed);
  };

  return (
    <div className="flex h-[calc(100vh-4rem)]">
      <div className={`${sidebarOpen ? "block" : "hidden"} md:block w-64 border-r border-[var(--color-border)] p-4 overflow-y-auto`}>
        <button onClick={startNew} className="btn-primary w-full mb-4 text-body-small py-1.5">
          New Chat
        </button>
        {conversations.map(c => (
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
              <p className="text-headline-4 text-[var(--color-text-tertiary)]">AI Assistant</p>
              <p className="text-body text-[var(--color-text-tertiary)] mt-2">
                Ask me about your devices, run commands, or get help managing your infrastructure.
              </p>
            </div>
          )}
          {messages.map((msg, i) => (
            <div key={i} className={`flex ${msg.role === "user" ? "justify-end" : "justify-start"}`}>
              <div
                className={`max-w-[70%] rounded-lg px-4 py-2 text-body-small ${
                  msg.role === "user"
                    ? "bg-[var(--color-primary)] text-white"
                    : "bg-[var(--color-surface-elevated)] text-[var(--color-text-primary)]"
                }`}
              >
                <p className="whitespace-pre-wrap">{msg.content}</p>
              </div>
            </div>
          ))}
          {isLoading && (
            <div className="flex justify-start">
              <div className="bg-[var(--color-surface-elevated)] rounded-lg px-4 py-2">
                <div className="flex gap-1">
                  <span className="w-2 h-2 bg-[var(--color-text-tertiary)] rounded-full animate-bounce" />
                  <span className="w-2 h-2 bg-[var(--color-text-tertiary)] rounded-full animate-bounce" style={{ animationDelay: "0.1s" }} />
                  <span className="w-2 h-2 bg-[var(--color-text-tertiary)] rounded-full animate-bounce" style={{ animationDelay: "0.2s" }} />
                </div>
              </div>
            </div>
          )}
        </div>

        <form onSubmit={handleSubmit} className="border-t border-[var(--color-border)] p-4">
          <div className="flex gap-2">
            <input
              type="text"
              value={input}
              onChange={e => setInput(e.target.value)}
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
