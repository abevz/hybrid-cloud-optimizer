# Detailed Design

This document is the contract-level design for the MVP. It turns the product
requirements and architecture into implementation-ready contracts. Implementation
tasks should follow this document instead of making API, interface, status, or
retry-policy decisions in code.

## Source Material

This design is normalized from the legacy MVP notes:

- `STRATEGY/Hybrid-Cloud-Optimizer-MVP-legacy.md`
- `STRATEGY/Hybrid-Cloud-Cost-Optimizer-Original.md`

The legacy MVP is the primary design source. The original concept is historical
context and includes deferred scope such as `CloudCostConfig`, migration
policies, Vault integration, and OpenTelemetry tracing.

## HybridWorkload API Contract

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

## Configuration Contract

Configuration is loaded once at startup into a typed config struct.

Fields:

- `AWSRegion`: env `AWS_REGION`, default `us-east-1`
- `AWSPricingAPIRegion`: fixed default `us-east-1`
- `ProxmoxScaleOutThreshold`: env `PROXMOX_SCALE_OUT_THRESHOLD`, default `0.85`
- `ProxmoxScaleBackThreshold`: env `PROXMOX_SCALE_BACK_THRESHOLD`, default `0.70`
- `VPNEndpoint`: env `VPN_ENDPOINT`, default `10.0.1.1:51820`
- `KarpenterEnabled`: env `KARPENTER_ENABLED`, default `true`
- `KarpenterNamespace`: env `KARPENTER_NAMESPACE`, default `karpenter`
- `LogLevel`: env `LOG_LEVEL`, default `info`
- `MetricsAddr`: env `METRICS_ADDR`, default `:8080`
- `ProbeAddr`: env `PROBE_ADDR`, default `:8081`

Validation rules:

- Scale thresholds must be greater than `0` and less than or equal to `1`.
- Scale-out threshold must be greater than scale-back threshold.
- `VPNEndpoint` must be non-empty when Karpenter/AWS placement is enabled.
- Core services receive config through constructors. Dependency injection, if
  used, stays in `provider.go` files and the composition root.

## Scheduler Interfaces And Decision Contract

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

## Controller Reconciliation Design

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

## Karpenter NodePool Contract

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

Implementation note:

- The exact Karpenter API version must follow the dependency chosen during
  implementation. The legacy design used `karpenter.sh/v1beta1`; do not hardcode
  a stale version if the project dependency uses a newer API.

## Status And Error Semantics

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

## Validation And Webhook Design

CRD schema validation is the first line of defense:

- priority enum
- capacity type enum
- non-negative budget
- required `resources`
- required `workloadTemplate`

Validating webhook is added later for semantic validation that CRD schema cannot
express well:

- CPU request must be non-zero.
- Memory request must be non-zero.
- Budget must be non-negative.
- Priority and capacity type checks remain as defensive validation.

No mutating webhook is required for MVP defaults if Kubebuilder default markers
cover the field defaults.

## Testing Contract

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
