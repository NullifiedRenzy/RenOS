"use client";

import type { Device, User } from "@/lib/types";
import { computeHealthScore } from "@/lib/utils";

interface WelcomeBannerProps {
  user: User | null;
  devices: Device[];
}

export function WelcomeBanner({ user, devices }: WelcomeBannerProps) {
  const hour = new Date().getHours();
  const greeting = hour < 12 ? "Good morning" : hour < 18 ? "Good afternoon" : "Good evening";
  const displayName = user?.display_name ?? "User";

  const onlineCount = devices.filter((d) => d.status === "online").length;
  const avgScore = devices.length > 0
    ? Math.round(devices.reduce((sum, d) => sum + computeHealthScore(d), 0) / devices.length)
    : 0;

  return (
    <div className="flex items-center justify-between">
      <div>
        <h1 className="text-headline-1 text-[var(--color-text-primary)]">
          {greeting}, {displayName}.
        </h1>
        <p className="text-body text-[var(--color-text-secondary)] mt-1">
          {devices.length === 0
            ? "No devices enrolled yet."
            : `${onlineCount} of ${devices.length} device${devices.length !== 1 ? "s" : ""} online${avgScore > 0 ? ` \u00B7 Avg health ${avgScore}/100` : ""}.`
          }
        </p>
      </div>
    </div>
  );
}
