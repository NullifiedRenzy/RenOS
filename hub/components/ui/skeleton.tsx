"use client";

import { cn } from "@/lib/utils";

export interface SkeletonProps extends React.HTMLAttributes<HTMLDivElement> {
  lines?: number;
}

export function Skeleton({ lines = 1, className, style, ...rest }: SkeletonProps) {
  if (lines > 1) {
    return (
      <div className={cn("space-y-2", className)} {...rest}>
        {Array.from({ length: lines }, (_, i) => (
          <div
            key={i}
            className={cn("skeleton h-4", i === lines - 1 && "w-3/4")}
            style={style}
            aria-hidden="true"
          />
        ))}
      </div>
    );
  }
  return (
    <div className={cn("skeleton h-4", className)} style={style} role="status" aria-label="Loading…" {...rest} />
  );
}

export function SkeletonCircle({ size = 40, className, ...rest }: React.HTMLAttributes<HTMLDivElement> & { size?: number }) {
  return (
    <div
      className={cn("skeleton rounded-full", className)}
      style={{ width: size, height: size, ...rest.style }}
      aria-hidden="true"
    />
  );
}
