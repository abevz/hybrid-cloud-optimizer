# Design

## Architecture

The MVP is a single Kubernetes operator running inside one hybrid Kubernetes
cluster. Proxmox nodes and AWS nodes are part of the same cluster. The operator
does not federate multiple clusters.

```text
HybridWorkload CR
        |
        v
Controller Reconciler
        |
        v
Decision Engine
   |        |        |
   v        v        v
Metrics   Pricing   VPN Health
Client    Client    Checker
   |
   v
Karpenter NodePool Manager
```

## Core Components

### API

Package: `api/v1alpha1`

Defines `HybridWorkload` spec, status, conditions, defaults, validation markers,
and webhook validation.

### Controller

Package: `internal/controller`

Owns reconciliation:

1. handle deletion and finalizers
2. read dry-run annotation
3. call decision engine
4. update status
5. create or delete Karpenter NodePools when needed
6. requeue for drift correction

### Scheduler

Package: `internal/scheduler`

Contains pure placement logic. It should depend on interfaces for metrics,
pricing, and health checks.

### Cost

Package: `internal/cost`

Fetches and caches AWS pricing data. The pricing client must be mockable in
unit tests.

### Metrics

Package: `internal/metrics`

Reads Proxmox utilization through Kubernetes Metrics API. The MVP treats Proxmox
nodes as Kubernetes nodes labeled with platform metadata.

### Health Check

Package: `internal/healthcheck`

Checks VPN tunnel health before AWS placement.

### Karpenter

Package: `internal/karpenter`

Creates, updates, and deletes Karpenter `NodePool` resources for workloads placed
on AWS.

### Config

Package: `internal/config`

Loads environment configuration once and passes typed config into providers.

### Dependency Injection

Package: `internal/di`

Uses `samber/do` provider functions. Imports of `samber/do` must stay in
`provider.go` files only.

## Key Decisions

### Single Hybrid Cluster

One cluster keeps the MVP understandable: one API server, one controller, one
resource model.

### Local Controller Development

During development, CRDs can be installed into the current Kubernetes context
with `make install`, while the controller runs locally with `make run`.

This lets the developer test against the Proxmox lab cluster without building
and pushing a controller image on every change.


### Karpenter for AWS Burst

Karpenter provisions EC2 capacity directly and supports per-workload dynamic
NodePools.

### Hysteresis

Separate scale-out and scale-back thresholds prevent placement flapping when
utilization sits near the boundary.

### VPN Health Gate

The operator must not create AWS capacity if the hybrid network path is unhealthy.

### Status as User Interface

The first operational interface is `kubectl get hybridworkload -o yaml`.
Placement, cost, and blocked reasons must be visible in status conditions.
