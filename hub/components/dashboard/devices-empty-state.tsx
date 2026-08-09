"use client";

import Link from "next/link";
import { useState } from "react";
import { Database, Laptop, Loader2, CheckCircle2, AlertCircle } from "lucide-react";
import { useDemo } from "@/hooks/use-demo";
import { EmptyState } from "@/components/ui/empty-state";

export function DevicesEmptyState() {
  const { demoStatus, seedDemo } = useDemo();
  const [seeded, setSeeded] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const demoLoaded = demoStatus.data?.demo_loaded ?? false;

  const handleSeed = async () => {
    setError(null);
    try {
      await seedDemo.mutateAsync();
      setSeeded(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load demo data.");
    }
  };

  return (
    <div className="card">
      <EmptyState
        title="No devices enrolled yet"
        description="Enroll your first server or edge device with the RenOS agent, or load demo data to explore a pre-populated system."
        action={
          <div className="flex flex-wrap items-center justify-center gap-3">
            {!demoLoaded && (
              <button onClick={handleSeed} disabled={seedDemo.isPending} className="btn-secondary">
                {seedDemo.isPending ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Database className="h-4 w-4" />
                )}
                Load demo data
              </button>
            )}
            <Link href="/devices" className="btn-primary">
              <Laptop className="h-4 w-4" />
              Enroll your first device
            </Link>
          </div>
        }
      />
      {seeded && (
        <p className="text-body-small text-[var(--color-success)] text-center flex items-center justify-center gap-2 pb-2 -mt-2">
          <CheckCircle2 className="h-4 w-4" />
          Demo data loaded. Refreshing...
        </p>
      )}
      {error && (
        <p className="text-body-small text-[var(--color-error)] text-center flex items-center justify-center gap-2 pb-2 -mt-2">
          <AlertCircle className="h-4 w-4" />
          {error}
        </p>
      )}
    </div>
  );
}
