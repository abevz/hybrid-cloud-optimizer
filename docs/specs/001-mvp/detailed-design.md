# Detailed Design

This document is the contract-level design for the MVP. It turns the product
requirements and architecture into implementation-ready contracts.
Implementation tasks should follow this document instead of making API,
interface, status, or retry-policy decisions in code.

## Scope Pin

The MVP ships with **CRD schema validation only**. The semantic validating
webhook described in [Webhook Scope](#webhook) is deferred to Phase 4 and is
covered by task T026 in [tasks.md](tasks.md). [spec.md](spec.md) MVP Scope is
aligned with this pin.

## Source Material

This design is normalized from the legacy MVP notes:

- `STRATEGY/Hybrid-Cloud-Optimizer-MVP-legacy.md`
- `STRATEGY/Hybrid-Cloud-Cost-Optimizer-Original.md`

The legacy MVP is the primary design source. The original concept is historical
context and includes deferred scope such as `CloudCostConfig`, migration
policies, Vault integration, and OpenTelemetry tracing.

## HybridWorkload API Contract {#api-contract}

`HybridWorkload` is the only custom resource in the MVP.

Group/version/kind:

- API group: `cost.hybrid.io`
- Version: `v1alpha1`
- Kind: `HybridWorkload`
- Scope: namespaced
- Status subresource: enabled

### Spec

`spec.priority`

- JSON field: `priority`
- Go type: string
- Required: yes
- Validation: enum `low`, `medium`, `high`
- Default: `medium`
- Meaning: scheduling priority. `high` prefers AWS when VPN health passes.

`spec.maxMonthlyCostCents`

- JSON field: `maxMonthlyCostCents`
- Go type: int64
- Required: yes
- Validation: minimum `0`
- Representation: integer cents, for example `$50.00` is stored as `5000`
- Behavior: `0` means no AWS budget is available unless a later task changes
  the budget model.
- Meaning: maximum allowed monthly AWS cost for this workload. The business
  meaning stays USD; the API stores cents to avoid floating-point CRD fields.

`spec.resources`

- JSON field: `resources`
- Go type: `corev1.ResourceRequirements`
- Required: yes
- Validation: CPU and memory requests must be non-zero. CRD markers should
  express the structural schema; webhook validation provides the semantic check.
- Meaning: workload CPU and memory requirements used for placement and AWS
  instance selection.

`spec.workloadTemplate`

- JSON field: `workloadTemplate`
- Go type: `corev1.PodTemplateSpec`
- Required: yes
- Meaning: pod template for the workload that will later be placed on Proxmox
  or AWS.

`spec.capacityType`

- JSON field: `capacityType`
- Go type: string
- Required: yes
- Validation: enum `spot`, `on-demand`
- Default: `spot`
- Meaning: requested AWS capacity type when AWS placement is selected.

### Status

`status.phase`

- JSON field: `phase`
- Go type: string
- Values: `Pending`, `Running`, `Failed`
- Meaning: high-level observed state of the workload placement process.

`status.recommendedPlatform`

- JSON field: `recommendedPlatform`
- Go type: string
- Values: `proxmox`, `aws`, `pending`
- Meaning: current placement recommendation from the decision engine.

`status.estimatedMonthlyCostCents`

- JSON field: `estimatedMonthlyCostCents`
- Go type: int64
- Validation: minimum `0`
- Representation: integer cents
- Meaning: estimated monthly AWS cost for the current recommendation. Proxmox
  placement reports `0`.

`status.karpenterNodePoolName`

- JSON field: `karpenterNodePoolName`
- Go type: string
- Meaning: owned Karpenter `NodePool` name when AWS placement is selected and
  dry-run mode is disabled.

`status.vpnHealth`

- JSON field: `vpnHealth`
- Go type: string
- Values: `Healthy`, `Unhealthy`, `Unknown`
- Meaning: last observed VPN health state used by the AWS placement gate.

`status.dryRun`

- JSON field: `dryRun`
- Go type: bool
- Meaning: records whether the latest decision was computed in dry-run mode.

`status.conditions`

- JSON field: `conditions`
- Go type: `[]metav1.Condition`
- List markers: map list keyed by `type`
- Condition types:
  - `SchedulingDecision`
  - `NodePoolReady`
  - `VPNHealthy`

`status.lastReconcileTime`

- JSON field: `lastReconcileTime`
- Go type: `*metav1.Time`
- Meaning: timestamp of the most recent completed reconciliation attempt.

### Print Columns

The CRD should expose these `kubectl get` columns:

- `Priority`: `.spec.priority`
- `Platform`: `.status.recommendedPlatform`
- `Cost`: `.status.estimatedMonthlyCostCents`
- `Phase`: `.status.phase`

## Configuration Contract {#config-inventory}

Configuration is loaded once at startup into a typed config struct in
`internal/config`. Defaults below are the **runtime source of truth** for any
value not overridden by an environment variable.

### Configuration Inventory

| Env Var | Type | Default | Validation | Consuming Task |
|---------|------|---------|------------|----------------|
| `AWS_REGION` | string | `us-east-1` | non-empty | T012 |
| `AWS_PRICING_API_REGION` | string | `us-east-1` | non-empty (fixed; AWS pricing API is regional) | T012 |
| `PROXMOX_SCALE_OUT_THRESHOLD` | float64 | `0.85` | `0 < x <= 1` AND `> PROXMOX_SCALE_BACK_THRESHOLD` | T007, T014 |
| `PROXMOX_SCALE_BACK_THRESHOLD` | float64 | `0.70` | `0 < x <= 1` AND `< PROXMOX_SCALE_OUT_THRESHOLD` | T007, T014 |
| `VPN_ENDPOINT` | string | `10.0.1.1:51820` | non-empty when AWS/Karpenter enabled | T013 |
| `KARPENTER_ENABLED` | bool | `true` | n/a | T018a |
| `KARPENTER_NAMESPACE` | string | `karpenter` | RFC 1123 label | T018a |
| `LOG_LEVEL` | string | `info` | one of `debug`, `info`, `warn`, `error` | T007, T023 |

Controller process bootstrap settings are configured via CLI flags in `cmd/main.go`,
not via `internal/config`. This includes metrics bind address, probe bind address,
leader election, metrics TLS settings, webhook TLS settings, and the HTTP/2
enablement toggle.

### Validation Rules

- Scale thresholds must be `> 0` and `<= 1`.
- Scale-out threshold must be strictly greater than scale-back threshold (no
  equality).
- `VPN_ENDPOINT` must be non-empty when `KARPENTER_ENABLED=true`.
- `LOG_LEVEL` matched case-insensitively but stored lower-case.
- Validation runs at startup; the controller MUST refuse to start on invalid
  config.

### Wiring

- `internal/config.LoadConfig()` loads environment-backed domain configuration once at startup.
- `LoadConfig()` returns the typed struct or an aggregated validation error.
- Controller process bootstrap options remain CLI flags in `cmd/main.go` and are applied before
`ctrl.NewManager(...)`.
- Core services receive config through constructors. Dependency injection, if
  used, stays in `provider.go` files and the composition root (see
  [P-001](constitution.md), [P-006](constitution.md)).

## Scheduler Interfaces And Decision Contract {#scheduler-interfaces}

The decision engine owns pure placement policy. It depends on interfaces for
metrics, pricing, and VPN health so business logic can be unit tested without
real AWS, Proxmox, or Kubernetes metrics services.

Inputs:

- `HybridWorkload.Spec`
- current `HybridWorkload.Status.RecommendedPlatform`
- Proxmox CPU and memory utilization
- AWS hourly price for the selected instance type
- VPN health state
- typed config thresholds

Output:

```go
type PlacementDecision struct {
    Platform              string
    Reason                string
    EstimatedMonthlyCostCents int64
    RequiresKarpenterPool bool
    InstanceType          string
    CapacityType          string
}
```

Decision rules:

1. `high` priority workloads prefer AWS after the VPN health check passes.
2. New or Proxmox workloads stay on Proxmox while max CPU/memory utilization is
   below the scale-out threshold.
3. Workloads already on AWS return to Proxmox only when utilization drops below
   the scale-back threshold.
4. When Proxmox is overloaded, AWS requires healthy VPN and available budget.
5. If AWS is blocked by budget or VPN health, the recommendation is `pending`.

AWS cost estimate:

- Convert hourly EC2 price to monthly cost using 730 hours.
- `on-demand` uses the baseline EC2 price.
- `spot` applies the MVP discount model of `0.3` times on-demand cost until a
  dedicated Spot pricing lookup is implemented.
- Keep money calculations in code precise enough to produce integer cents for
  the CRD-facing contract.

CRD compatibility note:

- Do not expose money fields as `float64` in API types.
- `controller-gen` warns that floating-point CRD fields are dangerous and can
  reject validations such as `Minimum` on those fields.
- The public CRD contract therefore uses integer cents even though the business
  meaning remains monthly USD cost.

Initial instance type selection:

- Up to 1 CPU and 2 GiB memory: `t3.small`
- Up to 2 CPU and 4 GiB memory: `t3.medium`
- Up to 4 CPU and 8 GiB memory: `t3.large`
- Above that: `t3.xlarge`

## Controller Reconciliation Design {#controller-flow}

The reconciler watches `HybridWorkload`.

Flow:

1. Fetch the `HybridWorkload`; ignore not-found errors.
2. Read dry-run mode from annotation `hcro.io/dry-run: "true"`.
3. Call the decision engine.
4. On decision errors, set status and conditions according to the error contract.
5. On successful decision, update recommendation, estimated cost, dry-run flag,
   VPN health, conditions, and last reconcile time.
6. Create or update a Karpenter `NodePool` only when the decision requires AWS
   capacity and dry-run is disabled.
7. Update the status subresource separately from spec.
8. Requeue periodically for drift correction.

RBAC must include:

- `hybridworkloads`
- `hybridworkloads/status`
- `hybridworkloads/finalizers`
- Karpenter `nodepools`
- Kubernetes `nodes`
- `metrics.k8s.io` node metrics

Deletion handling:

- Add a finalizer before creating external AWS/Karpenter resources.
- On deletion, delete the owned NodePool before removing the finalizer.
- Dry-run mode must not create or delete external resources.

## Karpenter NodePool Contract {#karpenter-contract}

NodePool name:

- `hcro-<hybridworkload-name>`

Ownership:

- The NodePool is owned by the `HybridWorkload` when API compatibility allows an
  owner reference.
- The controller must be able to delete the NodePool by deterministic name during
  finalizer cleanup.

NodePool requirements:

- capacity type: decision `CapacityType`
- instance type: decision `InstanceType`
- architecture: `amd64`
- node class reference: default EC2 node class for the cluster
- taint: `platform=aws:NoSchedule`

### Karpenter Version Policy

- The Karpenter API version MUST be taken from the dependency declared in
  `go.mod` (governed by [P-003](constitution.md)).
- Legacy design referenced `karpenter.sh/v1beta1`; the project MUST NOT
  hardcode this version in YAML manifests, prose, or fallback constants.
- A dedicated unit test asserts that the imported Karpenter API version
  matches the version registered with the controller-runtime scheme. The test
  fails the build if the two diverge.
- When upgrading Karpenter, update `go.mod` first, regenerate manifests with
  `make manifests`, and rerun the version assertion test.

## Status And Error Semantics {#error-semantics}

Typed errors:

- `ProxmoxUnavailableError`: retry after 10 seconds, set `SchedulingDecision`
  false with reason `ProxmoxUnavailable`.
- `VPNUnhealthyError`: retry after 1 minute, set `vpnHealth=Unhealthy` and
  `VPNHealthy` false.
- `BudgetExceededError`: retry after 5 minutes, set `phase=Pending` and
  `SchedulingDecision` false with reason `BudgetExceeded`.
- `KarpenterTimeoutError`: retry after 30 seconds, set `NodePoolReady` false.
- `PricingAPIUnavailableError`: retry after 30 seconds; use cached price if
  available, otherwise surface the error.

Successful decisions:

- Proxmox recommendation: `phase=Running`, `recommendedPlatform=proxmox`,
  `estimatedMonthlyCostCents=0`.
- AWS recommendation with NodePool created: `phase=Running`,
  `recommendedPlatform=aws`, `karpenterNodePoolName` set.
- Pending recommendation: `phase=Pending`, `recommendedPlatform=pending`, clear
  reason in `SchedulingDecision`.
- Dry-run: set `dryRun=true` and record what would have happened in
  `SchedulingDecision`.

## Validation And Webhook Design {#webhook}

### Phase 1 — CRD Schema Validation Only

CRD schema validation is the first and only line of defense in MVP Phase 1:

- priority enum
- capacity type enum
- non-negative budget (`Minimum=0`)
- required `resources`
- required `workloadTemplate`

### Phase 4 — Validating Webhook (Deferred, T026)

A validating webhook is added in Phase 4 for semantic validation that CRD
schema cannot express well:

- CPU request must be non-zero.
- Memory request must be non-zero.
- Budget must be non-negative (defense in depth; redundant with schema).
- Priority and capacity type checks remain as defensive validation.

The webhook is OUT of scope for Phase 1. [spec.md](spec.md) MVP Scope and
[tasks.md](tasks.md) Phase boundaries reflect this pin.

### Mutating Webhook

No mutating webhook is required for MVP defaults if Kubebuilder default markers
cover the field defaults.

## Testing Contract {#testing-contract}

Unit tests:

- Config defaults and validation.
- Decision matrix for priority, utilization, budget, VPN health, and hysteresis.
- Instance type selection.
- Error classification and retry semantics.

Integration tests with envtest:

- Create a valid `HybridWorkload`.
- Reconcile updates status.
- Dry-run computes status without creating a NodePool.
- AWS placement creates or updates the expected NodePool object.
- Finalizer deletes owned NodePool before removal.

E2E tests with kind:

- Install CRDs and controller manifests.
- Apply sample `HybridWorkload`.
- Observe status recommendation.
- Validate dry-run behavior.

## Deferred From Detailed MVP Scope

Do not add these to the MVP contracts unless requirements are updated first:

- `CloudCostConfig` CRD
- automatic live migration
- preferred location policy
- migration policy fields
- embedded monitoring/cost alert policy fields
- Vault integration
- OpenTelemetry tracing
- Helm chart packaging
