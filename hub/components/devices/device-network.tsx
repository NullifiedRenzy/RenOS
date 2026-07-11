"use client";

import { useNetworkInterfaces, getInterfaceIcon, formatSpeed } from "@/hooks/use-network";

interface NetworkInfoProps {
  deviceId: string;
}

const STATUS_COLORS: Record<string, string> = {
  up: "var(--color-success)",
  down: "var(--color-text-tertiary)",
};

export function NetworkInfo({ deviceId }: NetworkInfoProps) {
  const { data, isLoading, error } = useNetworkInterfaces(deviceId);
  const interfaces = data?.interfaces ?? [];

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Network Interfaces</h2>

      {isLoading && (
        <div className="space-y-2">
          <div className="skeleton h-8 w-full" />
          <div className="skeleton h-8 w-full" />
        </div>
      )}

      {error && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          Unable to load network interfaces.
        </p>
      )}

      {!isLoading && !error && interfaces.length === 0 && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          No network interfaces found.
        </p>
      )}

      {!isLoading && !error && interfaces.length > 0 && (
        <div className="space-y-3">
          {interfaces.map(iface => (
            <div
              key={iface.id}
              className="border border-[var(--color-border)] rounded-sm p-3"
            >
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <span className="text-body">{getInterfaceIcon(iface.interface_type)}</span>
                  <span className="text-body-small font-mono font-medium text-[var(--color-text-primary)]">
                    {iface.name}
                  </span>
                  <span
                    className="inline-block w-2 h-2 rounded-full"
                    style={{ backgroundColor: STATUS_COLORS[iface.status] ?? "var(--color-text-tertiary)" }}
                  />
                  <span className="text-caption text-[var(--color-text-secondary)]">
                    {iface.status === "up" ? "Up" : "Down"}
                  </span>
                  {iface.is_default_gateway && (
                    <span className="text-caption text-[var(--color-accent)] bg-[var(--color-accent)]/10 px-1.5 py-0.5 rounded-sm">
                      Default Gateway
                    </span>
                  )}
                </div>
                <span className="text-caption text-[var(--color-text-tertiary)]">
                  {iface.interface_type}
                  {iface.speed > 0 ? ` \u00B7 ${formatSpeed(iface.speed)}` : ""}
                </span>
              </div>

              {iface.mac_address && iface.interface_type !== "loopback" && (
                <p className="text-caption font-mono text-[var(--color-text-tertiary)] mb-1">
                  {iface.mac_address}
                </p>
              )}

              <div className="flex flex-wrap gap-x-4 gap-y-1">
                {iface.ipv4.length > 0 && (
                  <div>
                    <span className="text-caption text-[var(--color-text-tertiary)]">IPv4: </span>
                    {iface.ipv4.map(ip => (
                      <span key={ip} className="text-caption font-mono text-[var(--color-text-secondary)] mr-2">
                        {ip}
                      </span>
                    ))}
                  </div>
                )}
                {iface.ipv6.length > 0 && (
                  <div>
                    <span className="text-caption text-[var(--color-text-tertiary)]">IPv6: </span>
                    {iface.ipv6.slice(0, 2).map(ip => (
                      <span key={ip} className="text-caption font-mono text-[var(--color-text-secondary)] mr-2">
                        {ip}{ip.length > 40 ? "\u2026" : ""}
                      </span>
                    ))}
                    {iface.ipv6.length > 2 && (
                      <span className="text-caption text-[var(--color-text-tertiary)]">
                        +{iface.ipv6.length - 2} more
                      </span>
                    )}
                  </div>
                )}
              </div>

              {iface.dns_servers.length > 0 && (
                <p className="text-caption text-[var(--color-text-tertiary)] mt-1">
                  DNS: {iface.dns_servers.join(", ")}
                </p>
              )}

              {iface.mtu > 0 && (
                <p className="text-caption text-[var(--color-text-tertiary)]">
                  MTU: {iface.mtu}
                </p>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
