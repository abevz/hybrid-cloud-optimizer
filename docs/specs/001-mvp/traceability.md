# Traceability Matrix

Bidirectional map from requirements to design sections, tasks, and test files.
Refresh on every change to [requirements.md](requirements.md) or
[tasks.md](tasks.md).

## Functional Requirements

| ID | Title | Design Anchor | Detailed-Design Anchor | Tasks | Test Files |
|----|-------|---------------|------------------------|-------|------------|
| FR-001 | HybridWorkload API | [design.md#api](design.md#api) | [detailed-design.md#api-contract](detailed-design.md#api-contract) | T003, T004, T026 | `api/v1alpha1/hybridworkload_types_test.go` (planned) |
| FR-002 | Placement Decision | [design.md#scheduler](design.md#scheduler) | [detailed-design.md#scheduler-interfaces](detailed-design.md#scheduler-interfaces) | T010, T014, T016, T017 | `internal/scheduler/decision_test.go` (planned) |
| FR-003 | Hysteresis | [design.md#hysteresis](design.md#hysteresis) | [detailed-design.md#config-inventory](detailed-design.md#config-inventory) | T007, T014, T016 | `internal/scheduler/hysteresis_test.go` (planned) |
| FR-004 | AWS Cost Estimate | [design.md#cost](design.md#cost) | [detailed-design.md#scheduler-interfaces](detailed-design.md#scheduler-interfaces) | T012, T014 | `internal/cost/aws_pricing_client_test.go` (planned) |
| FR-005 | VPN Health Gate | [design.md#health-check](design.md#health-check) | [detailed-design.md#error-semantics](detailed-design.md#error-semantics) | T013, T014 | `internal/healthcheck/vpn_test.go` (planned) |
| FR-006 | Karpenter NodePool Management | [design.md#karpenter](design.md#karpenter) | [detailed-design.md#karpenter-contract](detailed-design.md#karpenter-contract) | T018a, T018b, T018c, T020, T022 | `internal/karpenter/nodepool_test.go` (planned), `test/e2e/karpenter_test.go` (planned) |
| FR-007 | Dry Run | [design.md#controller](design.md#controller) | [detailed-design.md#controller-flow](detailed-design.md#controller-flow) | T019 | `internal/controller/hybridworkload_controller_test.go` |
| FR-008 | Status Updates | [design.md#status-as-user-interface](design.md#status-as-user-interface) | [detailed-design.md#error-semantics](detailed-design.md#error-semantics) | T017, T021 | `internal/controller/hybridworkload_controller_test.go` |

## Non-Functional Requirements

| ID | Title | Design Anchor | Detailed-Design Anchor | Tasks | Test Files |
|----|-------|---------------|------------------------|-------|------------|
| NFR-001 | Reliability (leader election) | [design.md#controller](design.md#controller) | [detailed-design.md#controller-flow](detailed-design.md#controller-flow) | T025 | `cmd/main_test.go` (planned) |
| NFR-002 | Observability | [design.md#controller](design.md#controller) | [detailed-design.md#controller-flow](detailed-design.md#controller-flow) | T023, T024 | `internal/controller/metrics_test.go` (planned) |
| NFR-003 | Security (IRSA) | [design.md#cost](design.md#cost) | [detailed-design.md#config-inventory](detailed-design.md#config-inventory) | T012, T018a | manual smoke test in `test/e2e/` |
| NFR-004 | Testability | [design.md#scheduler](design.md#scheduler) | [detailed-design.md#testing-contract](detailed-design.md#testing-contract) | T008, T010, T016 | every `*_test.go` under `internal/` |
| NFR-005 | Maintainability (provider isolation) | [design.md#dependency-injection](design.md#dependency-injection) | [detailed-design.md#config-inventory](detailed-design.md#config-inventory) | T007, T010 | enforced by review; lint rule planned |

## Constitution Principles

| ID | Title | Source Requirement | Enforcement |
|----|-------|--------------------|-------------|
| P-001 | Provider wiring in `provider.go` | NFR-005 | code review, [constitution.md](constitution.md) |
| P-002 | Money in integer cents | FR-001, FR-004 | CRD validation, [constitution.md](constitution.md) |
| P-003 | Karpenter version from `go.mod` | FR-006 | dependency pin, [constitution.md](constitution.md) |
| P-004 | Table-driven decision tests | NFR-004 | T016 acceptance, [constitution.md](constitution.md) |
| P-005 | IRSA in production | NFR-003 | deployment checklist, [constitution.md](constitution.md) |
| P-006 | Constructor injection | NFR-004, NFR-005 | code review, [constitution.md](constitution.md) |
| P-007 | Contracts before code | working rule | T### Acceptance slot, [constitution.md](constitution.md) |

## How to use this matrix

1. Adding a requirement: append a row before tasks land. Empty Tasks/Tests cells
   are flagged in PR review.
2. Closing a task: confirm the task's requirement row references it.
3. Auditing coverage: scan the Test Files column. Empty cells mean the
   requirement is not yet verifiable.
