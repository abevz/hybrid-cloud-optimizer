# Glossary

Domain terms used across the spec. Every term used in
[spec.md](spec.md), [requirements.md](requirements.md), or
[detailed-design.md](detailed-design.md) is defined here.

## Capacity Type

The pricing model requested for AWS placement: `spot` (interruptible, cheaper)
or `on-demand` (uninterruptible, baseline price). See
[detailed-design.md](detailed-design.md) `spec.capacityType` for the API
contract.

## CRD

Custom Resource Definition. Kubernetes mechanism for adding new typed objects
to the API server. The project's CRD is `HybridWorkload`.

## Decision Engine

Pure-Go component in `internal/scheduler` that produces a `PlacementDecision`
from utilization, pricing, VPN health, and budget inputs. See
[design.md](design.md) Scheduler section and
[detailed-design.md](detailed-design.md) Scheduler Interfaces.

## Dry-Run

Mode triggered by the annotation `hcro.io/dry-run: "true"`. The controller
computes and publishes a placement decision without creating, updating, or
deleting external resources (Karpenter NodePools, AWS calls). See FR-007 in
[requirements.md](requirements.md).

## envtest

Kubebuilder test harness that runs a real `kube-apiserver` and `etcd` locally
without a full cluster. Used for controller integration tests. See
[testing.md](testing.md).

## Finalizer

Kubernetes mechanism that delays object deletion until cleanup logic runs. The
controller adds a finalizer before creating Karpenter NodePools and removes it
after the NodePools are deleted. See FR-006 in
[requirements.md](requirements.md).

## hcro.io

Annotation and condition prefix for project-owned keys. Currently used for
`hcro.io/dry-run`. Future custom annotations and conditions use the same
prefix.

## HybridWorkload

The single custom resource the operator watches. API group `cost.hybrid.io`,
version `v1alpha1`. Defined in `api/v1alpha1/hybridworkload_types.go`. Schema
contract in [detailed-design.md](detailed-design.md) API Contract section.

## Hysteresis

Two thresholds (`scale-out` 0.85, `scale-back` 0.70 by default) that prevent
placement flapping when Proxmox utilization sits near a single boundary. See
FR-003 in [requirements.md](requirements.md).

## IaC

Infrastructure as Code. The project uses Terraform for cluster bootstrap; the
operator itself is the runtime equivalent for workload placement.

## IRSA

IAM Roles for Service Accounts. AWS mechanism that maps a Kubernetes
ServiceAccount to an IAM role through OIDC. Required for production AWS
credentials per P-005 in [constitution.md](constitution.md) and NFR-003 in
[requirements.md](requirements.md).

## Karpenter

AWS-supported Kubernetes node provisioner. Watches unscheduled pods and creates
EC2 capacity to match. The operator manages `NodePool` objects for Karpenter to
honor. See [design.md](design.md) Karpenter section and
[detailed-design.md](detailed-design.md) Karpenter NodePool Contract.

## NodePool

Karpenter custom resource that describes a class of nodes Karpenter is allowed
to provision. The operator owns one NodePool per AWS-bound HybridWorkload, named
`hcro-<workload-name>`.

## PlacementDecision

Output of the decision engine. Carries platform (`proxmox` / `aws` /
`pending`), reason, estimated cost, instance type, and capacity type. See
[detailed-design.md](detailed-design.md) Scheduler Interfaces.

## Proxmox

On-premises hypervisor that hosts the local Kubernetes nodes. The operator
treats Proxmox-backed nodes as regular Kubernetes nodes labeled with platform
metadata. Free-of-cost capacity from the operator's accounting perspective.

## Reconciler

Controller-runtime component that observes `HybridWorkload` events and runs the
reconciliation loop in `internal/controller/hybridworkload_controller.go`.

## VPN Gate

Health check performed before any AWS placement. If the VPN tunnel between
Proxmox and AWS is unhealthy, AWS placement is blocked and the workload status
is set to `pending`. See FR-005 in [requirements.md](requirements.md).
