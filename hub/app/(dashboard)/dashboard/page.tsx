"use client";

import { useAuthStore } from "@/stores/auth-store";
import { useDevices } from "@/hooks/use-devices";
import { WelcomeBanner } from "@/components/dashboard/welcome-banner";
import { StatCards } from "@/components/dashboard/stat-cards";
import { DevicesEmptyState } from "@/components/dashboard/devices-empty-state";
import { ErrorState } from "@/components/ui/error-state";
import { LoadingCards } from "@/components/ui/loading-state";

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const { data, isLoading, error, refetch } = useDevices();
  const devices = data?.devices ?? [];

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="skeleton h-12 w-64" />
        <LoadingCards count={3} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="space-y-6">
        <div className="skeleton h-12 w-64" />
        <ErrorState
          error={error as Error}
          onRetry={() => refetch()}
          title="Could not load devices"
        />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <WelcomeBanner user={user} devices={devices} />
      {devices.length === 0 ? <DevicesEmptyState /> : <StatCards devices={devices} />}
    </div>
  );
}
