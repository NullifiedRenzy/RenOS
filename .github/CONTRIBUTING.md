# Contributing to RenOS

Thank you for your interest in contributing to RenOS.

## Code of Conduct

This project follows an open and welcoming code of conduct. Be respectful,
constructive, and assume good faith.

## Getting Started

See [docs/development/getting-started.md](../docs/development/getting-started.md)
for environment setup instructions.

## Pull Request Process

1. Create a feature branch from `dev`.
2. Keep changes focused — one feature or fix per PR.
3. Write tests for new functionality.
4. Ensure CI passes (lint, test, build).
5. Request review from the relevant CODEOWNERS.
6. Squash-merge to `dev`.

### Commit Messages

Use conventional commits:

```
type(scope): description

- type: feat, fix, refactor, test, docs, chore, security, perf
- scope: core, agent, hub, proto, ci
```

Examples:
- `feat(core): add device enrollment with mTLS`
- `fix(agent): correct reconnect backoff jitter`
- `docs: add ADR for plugin sandbox`

## PR Checklist

Before submitting, verify:

- [ ] Tests pass
- [ ] Lint passes
- [ ] No new warnings
- [ ] Documentation updated (if needed)
- [ ] Changes are backward-compatible (or documented as breaking)

## Architecture Decision Records

Significant architectural decisions are documented as ADRs in
`docs/architecture/`. See existing ADRs for the template.
