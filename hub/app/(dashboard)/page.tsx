"use client";

import { useAuthStore } from "@/stores/auth-store";
import { useDevices } from "@/hooks/use-devices";
import { WelcomeBanner } from "@/components/dashboard/welcome-banner";
import { StatCards } from "@/components/dashboard/stat-cards";

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const { data, isLoading } = useDevices();
  const devices = data?.devices ?? [];

  if (isLoading) {
    return (
      <div className="space-y-6">
        <div className="skeleton h-12 w-64" />
        <div className="grid grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="skeleton h-24" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <WelcomeBanner user={user} devices={devices} />
      <StatCards devices={devices} />
    </div>
  );
}
