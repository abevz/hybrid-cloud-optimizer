# Requirements

## Functional Requirements

### FR-001 HybridWorkload API

The system must define a `HybridWorkload` custom resource.

Required spec fields:

- priority: `low`, `medium`, or `high`
- max monthly cost in USD
- resource requirements
- workload template
- capacity type: `spot` or `on-demand`

Money values are business-level USD amounts, but the API representation should
avoid floating-point CRD fields. See
[detailed-design.md#api-contract](detailed-design.md#api-contract) for the
storage format.

Required status fields:

- phase
- recommended platform: `proxmox`, `aws`, or `pending`
- estimated monthly cost in USD
- Karpenter NodePool name when AWS is selected
- conditions
- VPN health state

Acceptance Scenarios:

- Given a HybridWorkload manifest with `priority=medium` and valid
  `resources.requests`, When `kubectl apply -f`, Then the API server accepts
  the object and CRD schema validation passes.
- Given a manifest with `priority=urgent`, When `kubectl apply -f`, Then the
  API server rejects the object because `priority` is not in the enum.
- Given a manifest with `maxMonthlyCostCents=-100`, When `kubectl apply -f`,
  Then the API server rejects the object due to the `Minimum=0` validation.

### FR-002 Placement Decision

The decision engine must choose placement using this priority order:

1. High priority workloads prefer AWS after the VPN health check passes.
2. If Proxmox utilization is below the scale-out threshold, place on Proxmox.
3. If the workload is currently on AWS and Proxmox is below the scale-back
   threshold, recommend returning to Proxmox.
4. If Proxmox is full, VPN is healthy, and budget is available, place on AWS.
5. If budget or health checks block AWS placement, mark placement as pending.

Acceptance Scenarios:

- Given `priority=high` AND `vpnHealth=Healthy`, When reconcile, Then
  `status.recommendedPlatform=aws`.
- Given `priority=low` AND `proxmoxUtil=0.5`, When reconcile, Then
  `status.recommendedPlatform=proxmox`.
- Given current platform is `aws` AND `proxmoxUtil=0.6`, When reconcile, Then
  `status.recommendedPlatform=proxmox` (scale-back).
- Given `proxmoxUtil=0.9` AND `vpnHealth=Healthy` AND budget available, When
  reconcile, Then `status.recommendedPlatform=aws`.
- Given `proxmoxUtil=0.9` AND `vpnHealth=Unhealthy`, When reconcile, Then
  `status.recommendedPlatform=pending` and `SchedulingDecision` condition
  reason is `VPNUnhealthy`.

### FR-003 Hysteresis

The system must use separate thresholds to prevent flapping:

- scale out to AWS when Proxmox utilization is above the scale-out threshold
- scale back to Proxmox when utilization is below the scale-back threshold

Both thresholds are configurable through environment variables. Defaults are
`0.85` (scale-out) and `0.70` (scale-back); see
[detailed-design.md#config-inventory](detailed-design.md#config-inventory) for
the exact env var names and validation rules. The configurable values are the
**source of truth at runtime**; the defaults above are reference values only.

Acceptance Scenarios:

- Given Proxmox utilization oscillates between `0.83` and `0.87`, When
  reconciling repeatedly over 5 minutes, Then `status.recommendedPlatform`
  changes at most once.
- Given current platform is `aws` AND `proxmoxUtil=0.72`, When reconcile, Then
  `status.recommendedPlatform=aws` (above scale-back, no flap).
- Given current platform is `aws` AND `proxmoxUtil=0.65`, When reconcile, Then
  `status.recommendedPlatform=proxmox` (below scale-back).

### FR-004 AWS Cost Estimate

The system must estimate AWS monthly cost before selecting AWS.

The MVP may use on-demand EC2 pricing as the baseline estimate. Spot capacity
applies the MVP discount model (0.3 × on-demand) until a dedicated Spot pricing
lookup is implemented; see
[detailed-design.md#scheduler-interfaces](detailed-design.md#scheduler-interfaces).

Acceptance Scenarios:

- Given on-demand price for `t3.medium` is `$0.0416/h`, When estimating monthly
  cost, Then `estimatedMonthlyCostCents = round(0.0416 * 730 * 100) = 3037`.
- Given `capacityType=spot` and on-demand monthly cost `3037`, When estimating
  monthly cost, Then `estimatedMonthlyCostCents = round(3037 * 0.3) = 911`.
- Given the AWS pricing API is unreachable AND a cached price exists, When
  estimating cost, Then the cached price is used.

### FR-005 VPN Health Gate

The system must check VPN health before creating or recommending AWS placement.

If VPN health fails, AWS placement must be blocked and the workload should
remain pending with a clear condition.

Acceptance Scenarios:

- Given `vpnHealth=Unhealthy` AND a workload that would otherwise be placed on
  AWS, When reconcile, Then `status.recommendedPlatform=pending` and
  `conditions[VPNHealthy]=False`.
- Given `vpnHealth=Healthy`, When reconcile a high-priority workload, Then
  `status.recommendedPlatform=aws` and `conditions[VPNHealthy]=True`.
- Given the VPN health checker returns an error, When reconcile, Then the
  controller requeues after the `VPNUnhealthyError` retry interval (60s).

### FR-006 Karpenter NodePool Management

When AWS is selected and dry-run is disabled, the controller must create or
update the required Karpenter `NodePool`.

When a workload is deleted, the controller must clean up owned NodePools before
removing its finalizer.

Acceptance Scenarios:

- Given `status.recommendedPlatform=aws` AND no NodePool exists for the
  workload, When reconcile, Then a Karpenter NodePool named
  `hcro-<workload-name>` is created and `status.karpenterNodePoolName` is set.
- Given a NodePool already exists with stale instance type, When the decision
  engine returns a different instance type, Then the NodePool is updated.
- Given a HybridWorkload with the project finalizer is deleted, When reconcile,
  Then the owned NodePool is deleted before the finalizer is removed.

### FR-007 Dry Run

When the annotation `hcro.io/dry-run: "true"` is set, the controller must
compute and publish the decision without creating, updating, or deleting
external resources.

Acceptance Scenarios:

- Given annotation `hcro.io/dry-run=true` AND a workload that would normally
  cause NodePool creation, When reconcile, Then no NodePool is created and
  `status.dryRun=true`.
- Given annotation `hcro.io/dry-run=true` AND an existing owned NodePool, When
  the workload is deleted, Then the NodePool is NOT deleted by dry-run logic
  (regular reconciliation is responsible for cleanup).
- Given the annotation is removed, When reconcile, Then external resources are
  reconciled normally and `status.dryRun=false`.

### FR-008 Status Updates

The controller must update status separately from spec and include enough
information for users to understand the placement decision.

Acceptance Scenarios:

- Given a freshly applied HybridWorkload, When the first reconciliation
  completes, Then `status.lastReconcileTime` is set and at least one condition
  is present.
- Given a successful AWS recommendation, When `kubectl get hybridworkload`,
  Then the print columns show non-empty `Platform`, `Cost`, and `Phase`.
- Given a status update, When inspecting the API server audit log, Then the
  patch targets the `/status` subresource only.

## Non-Functional Requirements

### NFR-001 Reliability

The controller must use leader election when running in-cluster.

Measurable criteria:

- Leader election lease ID matches the project's deployment manifest
  (`hybridworkload-controller`).
- Two replicas can run concurrently with only the leader performing
  reconciliations (verified via metric or log assertion).
- Leader takeover after a kill completes within 30 seconds.

### NFR-002 Observability

The controller must emit structured logs and Prometheus metrics for placement
count, decision duration, and estimated cost.

Measurable criteria:

- Metrics endpoint exposes at least: `hcro_placement_total{platform=}`,
  `hcro_decision_duration_seconds`, `hcro_estimated_cost_cents{workload=}`.
- Structured logs use `slog` and include the workload namespace/name on every
  reconciliation log line.
- A `/metrics` HTTP scrape returns HTTP 200 with content-type
  `text/plain; version=0.0.4`.

### NFR-003 Security

AWS credentials must use IRSA. The code must not rely on hardcoded credentials
or local credential files in production.

Measurable criteria:

- The deployment manifest does not mount any AWS credentials volume.
- The ServiceAccount used by the controller has an IRSA annotation in the
  production overlay.
- A code-level check (lint or test) flags any direct read of `AWS_ACCESS_KEY_ID`
  or `~/.aws/credentials` paths in non-test packages.

### NFR-004 Testability

Business logic must be testable with pure constructors and mocks. Service code
must not import the dependency injection container.

Measurable criteria:

- `internal/scheduler` test coverage ≥ 80%.
- `internal/config` test coverage ≥ 90%.
- `internal/scheduler` package contains no import of `samber/do` (verified via
  `go list -deps`).
- The decision engine accepts mock implementations of metrics, pricing, and
  health interfaces in unit tests.

### NFR-005 Maintainability

Provider wiring must stay isolated in `provider.go` files. Core services must
accept explicit dependencies through constructors.

Measurable criteria:

- The only files allowed to import `samber/do` are named `provider.go` (or the
  composition root `cmd/main.go`).
- Every constructor in `internal/` exposes its dependencies as explicit
  parameters; no hidden globals.
- Adding a new service requires editing one `provider.go` file and the
  constructor; no other change should be required to wire it in.
