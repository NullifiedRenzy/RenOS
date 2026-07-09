# ADR-001: Monorepo and Technology Stack

## Status

Accepted

## Context

The RenOS project needs to decide on repository structure and primary
technologies. The key decisions are:

1. Single repository vs multiple repositories
2. Language for the Core backend
3. Language for the Agent
4. Frontend framework

## Decision

### Monorepo

All components (Core, Agent, Hub, Proto) share one repository.

Rationale:
- Cross-component changes (e.g., adding a proto field and updating both
  Core and Agent) land in a single commit.
- CI runs all tests together — no integration gaps between repos.
- Single version tag covers all components.
- Go and Rust workspaces support monorepo structure well.

Trade-off: Larger clone size. Mitigated by `git clone --filter=blob:none`.

### Go for Core

The central server is written in Go.

Rationale:
- Excellent standard library for HTTP/gRPC servers.
- Goroutines provide lightweight concurrency for managing many agent
  connections.
- Fast compile times and single-binary deployment.
- Large ecosystem of networking and security libraries.

### Rust for Agent

The device agent is written in Rust.

Rationale:
- C-level performance with memory safety (no GC).
- Tiny binary (~5MB stripped) suitable for Raspberry Pi Zero.
- Cross-compilation to ARM/aarch64 is well-supported.
- No runtime dependencies — single static binary.

### Next.js + React + TypeScript for Hub

The web UI uses Next.js with React and TypeScript.

Rationale:
- Server components for fast initial loads.
- App Router provides layouts, loading states, error boundaries.
- Largest frontend ecosystem.
- TypeScript catches class of bugs at compile time.
- PWA support via next-pwa or custom service worker.

## Consequences

Positive:
- Single `git log` shows full project history.
- Consistent CI across all components.
- Shared proto definitions in one place.

Negative:
- Developers need Go, Rust, and Node toolchains installed.
- CI must build all three languages (mitigated by path-filtered jobs).
- Monorepo grows large over time (mitigated by sparse checkout).

## Alternatives Considered

- **Multi-repo**: Rejected due to cross-repo coordination overhead during
  proto changes.
- **Python for Core**: Rejected — slower, harder to deploy as single binary.
- **C for Agent**: Rejected — memory safety risks outweigh performance gain.
- **Remix for Hub**: Rejected — smaller ecosystem than Next.js.
