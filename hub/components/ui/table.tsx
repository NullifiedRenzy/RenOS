"use client";

import { cn } from "@/lib/utils";

export interface TableColumn<T> {
  key: string;
  header: React.ReactNode;
  cell: (row: T) => React.ReactNode;
  className?: string;
  headerClassName?: string;
}

interface TableProps<T> {
  columns: TableColumn<T>[];
  rows: T[];
  rowKey: (row: T) => string;
  emptyMessage?: string;
  className?: string;
}

export function Table<T>({ columns, rows, rowKey, emptyMessage = "No data", className }: TableProps<T>) {
  return (
    <div className={cn("overflow-x-auto rounded-[var(--radius-md)] border border-[var(--color-border)]", className)}>
      <table className="w-full text-left">
        <thead>
          <tr className="border-b border-[var(--color-border)]">
            {columns.map((c) => (
              <th scope="col" key={c.key} className={cn("px-4 py-3 text-body-small font-medium text-[var(--color-text-tertiary)]", c.headerClassName)}>
                {c.header}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-10 text-center text-body-small text-[var(--color-text-tertiary)]">
                {emptyMessage}
              </td>
            </tr>
          ) : (
            rows.map((row) => (
              <tr key={rowKey(row)} className="border-b border-[var(--color-border)] last:border-0 hover:bg-[var(--color-bg-secondary)]">
                {columns.map((c) => (
                  <td key={c.key} className={cn("px-4 py-3 text-body", c.className)}>
                    {c.cell(row)}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  );
}
