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

Required status fields:

- phase
- recommended platform: `proxmox`, `aws`, or `pending`
- estimated monthly cost in USD
- Karpenter NodePool name when AWS is selected
- conditions
- VPN health state

### FR-002 Placement Decision

The decision engine must choose placement using this priority order:

1. High priority workloads prefer AWS after the VPN health check passes.
2. If Proxmox utilization is below the scale-out threshold, place on Proxmox.
3. If the workload is currently on AWS and Proxmox is below the scale-back
   threshold, recommend returning to Proxmox.
4. If Proxmox is full, VPN is healthy, and budget is available, place on AWS.
5. If budget or health checks block AWS placement, mark placement as pending.

### FR-003 Hysteresis

The system must use separate thresholds to prevent flapping:

- scale out to AWS when Proxmox utilization is above `0.85`
- scale back to Proxmox when utilization is below `0.70`

Both thresholds must be configurable.

### FR-004 AWS Cost Estimate

The system must estimate AWS monthly cost before selecting AWS.

The MVP may use on-demand EC2 pricing as the baseline estimate. Spot capacity can
apply a discount model until a dedicated Spot pricing lookup is implemented.

### FR-005 VPN Health Gate

The system must check VPN health before creating or recommending AWS placement.

If VPN health fails, AWS placement must be blocked and the workload should remain
pending with a clear condition.

### FR-006 Karpenter NodePool Management

When AWS is selected and dry-run is disabled, the controller must create or
update the required Karpenter `NodePool`.

When a workload is deleted, the controller must clean up owned NodePools before
removing its finalizer.

### FR-007 Dry Run

When the annotation `hcro.io/dry-run: "true"` is set, the controller must compute
and publish the decision without creating, updating, or deleting external
resources.

### FR-008 Status Updates

The controller must update status separately from spec and include enough
information for users to understand the placement decision.

## Non-Functional Requirements

### NFR-001 Reliability

The controller must use leader election when running in-cluster.

### NFR-002 Observability

The controller must emit structured logs and Prometheus metrics for placement
count, decision duration, and estimated cost.

### NFR-003 Security

AWS credentials must use IRSA. The code must not rely on hardcoded credentials or
local credential files in production.

### NFR-004 Testability

Business logic must be testable with pure constructors and mocks. Service code
must not import the dependency injection container.

### NFR-005 Maintainability

Provider wiring must stay isolated in `provider.go` files. Core services must
accept explicit dependencies through constructors.
