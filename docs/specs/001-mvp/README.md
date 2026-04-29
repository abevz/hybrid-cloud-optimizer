# 001 MVP

Hybrid Cloud Cost Optimizer is a Kubernetes operator that chooses where a
workload should run: Proxmox for free local capacity, or AWS for paid burst
capacity when policy requires it.

This directory is the SDD version of the legacy root `MVP.md`.

## Documents

- [spec.md](spec.md) - product scope and non-goals
- [requirements.md](requirements.md) - functional and non-functional requirements
- [design.md](design.md) - architecture, components, and key decisions
- [tasks.md](tasks.md) - implementation phases and task checklist
- [testing.md](testing.md) - unit, integration, and E2E testing strategy

## Working Rule

When implementing a task, update the spec first if the intended behavior changes.
Then implement, test, and reference the task ID in the Pull Request.
