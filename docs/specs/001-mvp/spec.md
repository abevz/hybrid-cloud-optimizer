# MVP Specification

## Problem

Small teams can run baseline Kubernetes workloads on Proxmox, but still need a
safe way to burst into AWS when local capacity is exhausted, latency or priority
requires it, or reliability policy demands it.

The operator should make this placement decision explicitly and transparently so
the cost tradeoff is visible before AWS resources are created.

## Goal

Build a Kubernetes operator that watches `HybridWorkload` resources and
recommends or provisions placement on:

- `proxmox` for free on-prem capacity
- `aws` for burst capacity through Karpenter
- `pending` when placement is blocked by budget, capacity, or health checks

## MVP Scope

The MVP includes:

- `HybridWorkload` CRD with spec and status subresource (CRD schema validation only; semantic validating webhook is deferred to Phase 4 — see [detailed-design.md#webhook](detailed-design.md#webhook))
- controller reconciliation loop
- placement decision engine with hysteresis
- Proxmox utilization lookup through Kubernetes Metrics API
- AWS EC2 pricing lookup with caching
- VPN health gate before AWS placement
- Karpenter `NodePool` management for AWS burst
- dry-run annotation support
- structured errors with retry guidance
- structured logging with `slog`
- basic Prometheus metrics
- CLI report for cost savings
- unit, integration, and E2E test coverage for the critical paths

## Non-Goals

Deferred from MVP:

- automatic live migration between platforms
- multi-cluster federation
- OpenTelemetry tracing
- Helm chart packaging
- Vault integration
- production WireGuard automation beyond the planned Terraform module

## Success Criteria

- After applying a `HybridWorkload`, a non-empty `status.recommendedPlatform`
  appears within 10 seconds in 95% of reconciliations.
- For workloads with `priority` in `low` or `medium`, when Proxmox utilization
  is below the scale-out threshold, `status.recommendedPlatform == proxmox`.
- For workloads with `priority == high`, when VPN health is `Healthy`,
  `status.recommendedPlatform == aws`.
- Hysteresis prevents platform flapping: at most one platform switch per
  5 minutes when Proxmox utilization oscillates around the scale-out threshold.
- When `estimatedMonthlyCostCents > maxMonthlyCostCents`,
  `status.recommendedPlatform == pending` and the `SchedulingDecision`
  condition reports `BudgetExceeded`.
- Dry-run mode (annotation `hcro.io/dry-run: "true"`) sets `status.dryRun=true`
  and creates zero Karpenter `NodePool` objects.
- Unit and integration tests run locally without AWS credentials. `make test`
  passes on a workstation with no AWS configuration present.

## Review Checklist

Before marking the spec ready for implementation:

- [ ] User scenarios listed and mapped to FRs in [requirements.md](requirements.md)
- [ ] Edge cases covered: VPN unavailable, budget exceeded, Proxmox metrics missing
- [ ] Success criteria are measurable (numbers or observable states)
- [ ] Non-goals explicit and current
- [ ] Glossary linked: see [glossary.md](glossary.md)
- [ ] No implementation details (Go types, env vars, library names) in this file
