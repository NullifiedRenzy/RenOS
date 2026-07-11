"use client";

import { useVirtualMachines, useVMAction, formatVmMemory, getVmStateColor } from "@/hooks/use-virtualization";

interface VirtualMachinesProps {
  deviceId: string;
}

export function VirtualMachines({ deviceId }: VirtualMachinesProps) {
  const { data, isLoading, error } = useVirtualMachines(deviceId);
  const vmAction = useVMAction(deviceId);
  const vms = data?.vms ?? [];

  const canStart = (state: string) => ["shut off", "crashed"].includes(state.toLowerCase());
  const canStop = (state: string) => ["running", "paused"].includes(state.toLowerCase());
  const canReset = (state: string) => ["running", "paused", "shut off"].includes(state.toLowerCase());

  const handleAction = (domainName: string, action: string) => {
    vmAction.mutate({ domainName, action });
  };

  return (
    <div className="card">
      <h2 className="text-headline-4 text-[var(--color-text-primary)] mb-3">Virtual Machines</h2>

      {isLoading && (
        <div className="space-y-2">
          <div className="skeleton h-8 w-full" />
          <div className="skeleton h-8 w-full" />
        </div>
      )}

      {error && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          Unable to load virtual machines.
        </p>
      )}

      {!isLoading && !error && vms.length === 0 && (
        <p className="text-body text-[var(--color-text-tertiary)] py-3 text-center">
          No virtual machines found.
        </p>
      )}

      {!isLoading && !error && vms.length > 0 && (
        <div className="overflow-x-auto">
          <table className="w-full text-left">
            <thead>
              <tr className="text-caption text-[var(--color-text-tertiary)] border-b border-[var(--color-border)]">
                <th className="pb-2 pr-3 font-medium">Domain</th>
                <th className="pb-2 pr-3 font-medium">State</th>
                <th className="pb-2 pr-3 font-medium hidden md:table-cell">vCPU</th>
                <th className="pb-2 pr-3 font-medium hidden md:table-cell">Memory</th>
                <th className="pb-2 font-medium">Actions</th>
              </tr>
            </thead>
            <tbody>
              {vms.map(vm => (
                <tr key={vm.id} className="border-b border-[var(--color-border)] last:border-b-0">
                  <td className="py-2 pr-3">
                    <div className="flex items-center gap-2">
                      <span className="text-body">{"\u2386"}</span>
                      <div>
                        <p className="text-body-small font-mono text-[var(--color-text-primary)]">{vm.domain_name}</p>
                        {vm.os_type && (
                          <p className="text-caption text-[var(--color-text-tertiary)]">{vm.os_type}</p>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="py-2 pr-3">
                    <div className="flex items-center gap-1.5">
                      <span
                        className="inline-block w-2 h-2 rounded-full"
                        style={{ backgroundColor: getVmStateColor(vm.state) }}
                      />
                      <span className="text-body-small text-[var(--color-text-secondary)]">
                        {vm.state}
                      </span>
                    </div>
                  </td>
                  <td className="py-2 pr-3 hidden md:table-cell">
                    <span className="text-body-small text-[var(--color-text-secondary)]">
                      {vm.vcpu_current}/{vm.vcpu_max}
                    </span>
                  </td>
                  <td className="py-2 pr-3 hidden md:table-cell">
                    <span className="text-body-small text-[var(--color-text-secondary)]">
                      {formatVmMemory(vm.memory_current_bytes)} / {formatVmMemory(vm.memory_max_bytes)}
                    </span>
                  </td>
                  <td className="py-2">
                    <div className="flex items-center gap-1">
                      <button
                        onClick={() => handleAction(vm.domain_name, "start")}
                        disabled={vmAction.isPending || !canStart(vm.state)}
                        className="btn-ghost text-body-small px-2 py-0.5"
                      >
                        Start
                      </button>
                      <button
                        onClick={() => handleAction(vm.domain_name, "shutdown")}
                        disabled={vmAction.isPending || !canStop(vm.state)}
                        className="btn-ghost text-body-small px-2 py-0.5"
                      >
                        Shutdown
                      </button>
                      <button
                        onClick={() => handleAction(vm.domain_name, "destroy")}
                        disabled={vmAction.isPending || !canStop(vm.state)}
                        className="btn-ghost text-body-small px-2 py-0.5"
                      >
                        Destroy
                      </button>
                      <button
                        onClick={() => handleAction(vm.domain_name, "reset")}
                        disabled={vmAction.isPending || !canReset(vm.state)}
                        className="btn-ghost text-body-small px-2 py-0.5"
                      >
                        Reset
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
