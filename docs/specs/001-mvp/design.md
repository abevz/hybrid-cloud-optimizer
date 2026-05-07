# Design

## Architecture {#architecture}

The MVP is a single Kubernetes operator running inside one hybrid Kubernetes
cluster. Proxmox nodes and AWS nodes are part of the same cluster. The operator
does not federate multiple clusters.

```text
                  HybridWorkload CR
                          │
                          ▼
                  Controller Reconciler ──► (re-queue with retry interval on typed errors)
                          │
                  (read dry-run annotation)
                          │
                          ▼
                   Decision Engine
                    │      │      │
                    ▼      ▼      ▼
              Metrics  Pricing  VPN Health
              Client   Client    Checker
                    │      │      │
                    │      │      └─► VPNUnhealthyError ──► status: pending, retry 60s
                    │      └─► PricingAPIUnavailableError ──► use cache or retry 30s
                    └─► ProxmoxUnavailableError ──► retry 10s
                          │
                          ▼
                   PlacementDecision
                    │              │
            (proxmox or pending)  (aws and not dry-run)
                    │              │
                    │              ▼
                    │     Karpenter NodePool Manager
                    │              │
                    │              ├─► create / update NodePool
                    │              └─► KarpenterTimeoutError ──► retry 30s
                    │
                    ▼
              Status subresource update
              (phase, recommendedPlatform, conditions, dryRun, lastReconcileTime)
```

## Core Components

### API {#api}

Package: `api/v1alpha1`

Defines `HybridWorkload` spec, status, conditions, defaults, and CRD schema
validation markers. The semantic validating webhook is added in Phase 4 (see
[detailed-design.md#webhook](detailed-design.md#webhook)).

### Controller {#controller}

Package: `internal/controller`

Owns reconciliation:

1. handle deletion and finalizers
2. read dry-run annotation
3. call decision engine
4. update status
5. create or delete Karpenter NodePools when needed
6. requeue for drift correction

### Scheduler {#scheduler}

Package: `internal/scheduler`

Contains pure placement logic. It should depend on interfaces for metrics,
pricing, and health checks.

### Cost {#cost}

Package: `internal/cost`

Fetches and caches AWS pricing data. The pricing client must be mockable in
unit tests.

### Metrics {#metrics}

Package: `internal/metrics`

Reads Proxmox utilization through Kubernetes Metrics API. The MVP treats
Proxmox nodes as Kubernetes nodes labeled with platform metadata.

### Health Check {#health-check}

Package: `internal/healthcheck`

Checks VPN tunnel health before AWS placement.

### Karpenter {#karpenter}

Package: `internal/karpenter`

Creates, updates, and deletes Karpenter `NodePool` resources for workloads
placed on AWS.

### Config {#config}

Package: `internal/config`

Loads environment configuration once and passes typed config into providers.

### Dependency Injection {#dependency-injection}

Package: `internal/di`

Uses `samber/do` provider functions. Imports of `samber/do` must stay in
`provider.go` files only (see [P-001](constitution.md)).

## Key Decisions

### Single Hybrid Cluster {#single-hybrid-cluster}

Decision: one Kubernetes cluster with both Proxmox and AWS nodes.
Alternatives considered: multi-cluster federation (rejected: too complex for
MVP), separate clusters with workload mirroring (rejected: doubles operational
surface).
Rationale: one API server, one controller, one resource model; the MVP keeps
all decision-making local to a single reconciler.
Trade-off: cluster reliability becomes a single point of failure.

### Local Controller Development {#local-controller-development}

Decision: CRDs installed via `make install`; the controller runs locally with
`make run` and talks to the lab cluster through kubeconfig.
Alternatives considered: dev image push to a registry on every change
(rejected: slow feedback loop), Tilt/Skaffold (deferred until the team grows).
Rationale: shortest feedback loop for solo development.
Trade-off: production parity gap; fixed by `test/e2e` against kind.

### Karpenter for AWS Burst {#karpenter-decision}

Decision: provision AWS EC2 capacity through Karpenter `NodePool` objects.
Alternatives considered: Cluster Autoscaler with managed node groups
(rejected: less flexible per-workload provisioning), direct EC2 SDK calls
(rejected: re-implements Karpenter).
Rationale: Karpenter supports per-workload dynamic NodePools, which match the
operator's model where each `HybridWorkload` may want a different instance
type.
Trade-off: Karpenter API has churned; pinning is governed by
[P-003](constitution.md).

### Hysteresis {#hysteresis}

Decision: separate scale-out (`0.85` default) and scale-back (`0.70` default)
thresholds.
Alternatives considered: single threshold (rejected: causes flapping),
time-based debounce (deferred to a later phase).
Rationale: two thresholds prevent flap when utilization oscillates near the
boundary.
Trade-off: two configuration knobs instead of one.

### VPN Health Gate {#vpn-gate-decision}

Decision: the operator must not create AWS capacity when the hybrid network
path is unhealthy.
Alternatives considered: best-effort placement with later cleanup (rejected:
creates orphaned NodePools and AWS bill).
Rationale: failing fast is cheaper and clearer in status conditions.
Trade-off: the VPN health checker is a new dependency the operator must
maintain.

### Status as User Interface {#status-as-user-interface}

Decision: the first operational interface is `kubectl get hybridworkload -o
yaml`. Placement, cost, and blocked reasons live in status conditions.
Alternatives considered: dedicated dashboard (deferred), Kubernetes Events only
(rejected: events are ephemeral; conditions are queryable).
Rationale: conditions and print columns work everywhere `kubectl` works, with
no extra infrastructure.
Trade-off: status is operator-defined; users must learn the condition types.
