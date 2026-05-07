# 001 MVP

Hybrid Cloud Cost Optimizer is a Kubernetes operator that chooses where a
workload should run: Proxmox for free local capacity, or AWS for paid burst
capacity when policy requires it.

This directory is the SDD version of the legacy root `MVP.md`.

## Documents

- [constitution.md](constitution.md) — immutable project principles
- [glossary.md](glossary.md) — domain terminology
- [spec.md](spec.md) — product scope, success criteria, and non-goals
- [requirements.md](requirements.md) — functional and non-functional requirements with Acceptance Scenarios
- [design.md](design.md) — architecture, components, and key decisions
- [detailed-design.md](detailed-design.md) — API, config, controller, status, and testing contracts
- [tasks.md](tasks.md) — implementation phases and task checklist
- [testing.md](testing.md) — unit, integration, and E2E testing strategy
- [traceability.md](traceability.md) — bidirectional matrix mapping requirements to design, tasks, and tests

## Recommended Reading Order

For a newcomer or an AI agent entering the project:

1. [constitution.md](constitution.md) — non-negotiable rules
2. [glossary.md](glossary.md) — domain terms
3. [spec.md](spec.md) — what and why
4. [requirements.md](requirements.md) — testable behaviors
5. [design.md](design.md) — components and rationale
6. [detailed-design.md](detailed-design.md) — implementation contracts
7. [tasks.md](tasks.md) — pick the next task
8. [testing.md](testing.md) — verification strategy
9. [traceability.md](traceability.md) — coverage audit

The strategy document
[`STRATEGY/2026-05-07-sdd-workflow-hcro.md`](../../../../Obsidian/moonbase/STRATEGY/2026-05-07-sdd-workflow-hcro.md)
in the author's Obsidian vault explains the SDD philosophy and contains
ready-to-fill templates.

## Working Rule

When implementing a task:

1. If the intended behavior changes, update [spec.md](spec.md) first.
2. If a contract changes, update [detailed-design.md](detailed-design.md).
3. Refresh [traceability.md](traceability.md) on the same commit.
4. Implement from the approved contract.
5. Add or update the test that closes the requirement.
6. Reference the task ID (`T0XX`) in the Pull Request title or body.
