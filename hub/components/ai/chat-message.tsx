"use client";

import { useMemo } from "react";
import type { ChatMessage } from "@/lib/types";

export function ChatMessageComponent({ message }: { message: ChatMessage }) {
  const isUser = message.role === "user";

  const renderedContent = useMemo(() => {
    if (isUser) return message.content;
    return renderMarkdown(message.content);
  }, [message.content, isUser]);

  return (
    <div className={`flex ${isUser ? "justify-end" : "justify-start"}`}>
      {!isUser && (
        <div className="w-7 h-7 rounded-full bg-[var(--color-accent-tint)] flex items-center justify-center flex-shrink-0 mr-2 mt-1">
          <span className="text-[var(--color-accent)] text-xs font-bold">AI</span>
        </div>
      )}
      <div
        className={`max-w-[80%] rounded-lg px-4 py-2 text-body-small ${
          isUser
            ? "bg-[var(--color-primary)] text-white"
            : "bg-[var(--color-surface-elevated)] text-[var(--color-text-primary)]"
        }`}
      >
        {isUser ? (
          <p className="whitespace-pre-wrap">{message.content}</p>
        ) : (
          <div className="prose prose-sm max-w-none prose-pre:bg-[var(--color-bg)] prose-code:text-[var(--color-accent)]">
            {renderedContent}
          </div>
        )}
      </div>
      {isUser && (
        <div className="w-7 h-7 rounded-full bg-[var(--color-primary)] flex items-center justify-center flex-shrink-0 ml-2 mt-1">
          <span className="text-white text-xs font-bold">U</span>
        </div>
      )}
    </div>
  );
}

function renderMarkdown(content: string): React.ReactNode {
  const lines = content.split("\n");
  const elements: React.ReactNode[] = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    if (line.startsWith("```")) {
      const lang = line.slice(3).trim();
      const codeLines: string[] = [];
      i++;
      while (i < lines.length && !lines[i].startsWith("```")) {
        codeLines.push(lines[i]);
        i++;
      }
      i++;
      elements.push(
        <div key={elements.length} className="relative my-2">
          {lang && (
            <div className="absolute top-0 right-0 px-2 py-0.5 text-[10px] text-[var(--color-text-tertiary)] bg-[var(--color-surface)] rounded-bl">
              {lang}
            </div>
          )}
          <pre className="bg-[var(--color-bg)] border border-[var(--color-border)] rounded-md p-3 overflow-x-auto text-[13px]">
            <code>{codeLines.join("\n")}</code>
          </pre>
        </div>
      );
      continue;
    }

    if (line.startsWith("### ")) {
      elements.push(<h3 key={elements.length} className="text-headline-4 mt-3 mb-1">{line.slice(4)}</h3>);
      i++;
      continue;
    }
    if (line.startsWith("## ")) {
      elements.push(<h2 key={elements.length} className="text-headline-3 mt-4 mb-2">{line.slice(3)}</h2>);
      i++;
      continue;
    }

    if (line.startsWith("- ") || line.startsWith("* ")) {
      const listItems: string[] = [];
      while (i < lines.length && (lines[i].startsWith("- ") || lines[i].startsWith("* "))) {
        listItems.push(lines[i].slice(2));
        i++;
      }
      elements.push(
        <ul key={elements.length} className="list-disc list-inside space-y-1 my-1">
          {listItems.map((item, idx) => (
            <li key={idx}>{renderInlineMarkdown(item)}</li>
          ))}
        </ul>
      );
      continue;
    }

    if (/^\d+\.\s/.test(line)) {
      const listItems: string[] = [];
      while (i < lines.length && /^\d+\.\s/.test(lines[i])) {
        listItems.push(lines[i].replace(/^\d+\.\s/, ""));
        i++;
      }
      elements.push(
        <ol key={elements.length} className="list-decimal list-inside space-y-1 my-1">
          {listItems.map((item, idx) => (
            <li key={idx}>{renderInlineMarkdown(item)}</li>
          ))}
        </ol>
      );
      continue;
    }

    if (line.startsWith("|")) {
      const tableLines: string[] = [];
      while (i < lines.length && lines[i].startsWith("|")) {
        tableLines.push(lines[i]);
        i++;
      }
      elements.push(renderTable(tableLines, elements.length));
      continue;
    }

    if (line.trim() === "") {
      elements.push(<div key={elements.length} className="h-2" />);
      i++;
      continue;
    }

    elements.push(<p key={elements.length} className="my-1">{renderInlineMarkdown(line)}</p>);
    i++;
  }

  return elements;
}

function renderInlineMarkdown(text: string): React.ReactNode {
  const parts: React.ReactNode[] = [];
  const regex = /(\*\*(.+?)\*\*|`(.+?)`|\*(.+?)\*)/g;
  let lastIndex = 0;
  let match;
  let key = 0;

  while ((match = regex.exec(text)) !== null) {
    if (match.index > lastIndex) {
      parts.push(text.slice(lastIndex, match.index));
    }
    if (match[2]) {
      parts.push(<strong key={key++}>{match[2]}</strong>);
    } else if (match[3]) {
      parts.push(
        <code key={key++} className="px-1 py-0.5 bg-[var(--color-surface)] rounded text-[var(--color-accent)] text-[13px]">
          {match[3]}
        </code>
      );
    } else if (match[4]) {
      parts.push(<em key={key++}>{match[4]}</em>);
    }
    lastIndex = match.index + match[0].length;
  }

  if (lastIndex < text.length) {
    parts.push(text.slice(lastIndex));
  }

  return parts.length > 0 ? parts : text;
}

function renderTable(lines: string[], key: number): React.ReactNode {
  if (lines.length < 2) return null;
  const headers = lines[0].split("|").map((c) => c.trim()).filter(Boolean);
  const rows = lines.slice(2).map((line) =>
    line.split("|").map((c) => c.trim()).filter(Boolean)
  );

  return (
    <table key={key} className="border-collapse border border-[var(--color-border)] my-2 text-[13px]">
      <thead>
        <tr>
          {headers.map((h, i) => (
            <th scope="col" key={i} className="border border-[var(--color-border)] px-2 py-1 text-left bg-[var(--color-surface)]">
              {h}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {rows.map((row, ri) => (
          <tr key={ri}>
            {row.map((cell, ci) => (
              <td key={ci} className="border border-[var(--color-border)] px-2 py-1">
                {cell}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
}
