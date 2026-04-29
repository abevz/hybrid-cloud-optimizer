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

- `HybridWorkload` CRD with spec and status subresource
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
- validating webhook for invalid specs
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

- A user can apply a `HybridWorkload` and see a placement recommendation in
  status.
- Low and medium priority workloads prefer Proxmox while it has capacity.
- High priority workloads are placed on AWS when the VPN health gate passes.
- Hysteresis prevents flapping between Proxmox and AWS.
- AWS placement is blocked when estimated cost exceeds budget.
- Dry-run mode computes decisions without creating Karpenter resources.
- The controller can be tested locally without a real AWS account for unit and
  integration tests.
