# Tasks

Each task carries three mandatory slots:

- **Requirement** — the FR/NFR (and constitution principle, if relevant) the
  task closes. Always cross-reference [requirements.md](requirements.md).
- **Acceptance** — a concrete artifact and the verification command that
  proves the task is done.
- **Blocks / Blocked by** — explicit dependency edges. Tasks marked `[P]`
  carry no blocking dependency and may run in parallel.

When a task is split (e.g. T018 → T018a/b/c), the parent ID is removed and
each child gets its own row.

## Phase 1: Foundation

Goal: establish a compilable operator skeleton and API shape.

- [x] T001 Initialize or normalize Go module layout.
      Requirement: scaffolding precondition for FR-001
      Acceptance: `go build ./...` succeeds; `go.mod` declares module path.
      Blocks: [T002]
      Blocked by: []

- [x] T002 Add Kubebuilder/controller-runtime project skeleton.
      Requirement: scaffolding precondition for FR-001, FR-008
      Acceptance: `make manifests` runs without error; `PROJECT` file present.
      Blocks: [T003, T005]
      Blocked by: [T001]

- [x] T003 Implement `HybridWorkload` API types from [detailed-design.md#api-contract](detailed-design.md#api-contract).
      Requirement: FR-001 (and [P-002](constitution.md))
      Acceptance: `api/v1alpha1/hybridworkload_types.go` defines all spec/status fields with kubebuilder markers; `make manifests` produces a CRD whose schema matches the contract.
      Blocks: [T004, T006, T026]
      Blocked by: [T002]

- [x] T004 Add CRD generation target.
      Requirement: FR-001
      Acceptance: `make manifests` regenerates `config/crd/bases/cost.hybrid.io_hybridworkloads.yaml`; `git diff` is empty after a clean run.
      Blocks: [T028]
      Blocked by: [T003]

- [x] T005 Add controller manager entrypoint.
      Requirement: FR-008, NFR-001
      Acceptance: `make build` produces a binary that boots, registers the scheme, and serves `/healthz` and `/metrics`.
      Blocks: [T006, T024, T025]
      Blocked by: [T002]

- [x] T006 Add basic reconciler that watches `HybridWorkload`.
      Requirement: FR-008
      Acceptance: `internal/controller/hybridworkload_controller.go` reconciles on Create/Update events; logs include workload namespace/name; the reconciler returns nil error for the empty-status path.
      Blocks: [T017, T019]
      Blocked by: [T003, T005]

- [ ] T007 [P] Implement typed config loading from [detailed-design.md#config-inventory](detailed-design.md#config-inventory).
      Requirement: NFR-005, FR-003 (defaults for thresholds)
      Acceptance: `internal/config.LoadConfig()` returns a typed struct with all 10 fields populated; invalid env input returns an aggregated error; `cmd/main.go` calls `LoadConfig()` before scheme registration.
      Blocks: [T010, T014, T023]
      Blocked by: []

- [ ] T008 [P] Add unit tests for config loading and validation.
      Requirement: NFR-004, NFR-005
      Acceptance: `go test ./internal/config/...` passes ≥ 90% coverage; cases include defaults, valid override, invalid threshold pair, missing VPN endpoint when Karpenter enabled.
      Blocks: []
      Blocked by: [T007]

- [x] T009 Add lint, format, and test targets.
      Requirement: NFR-004 (toolchain)
      Acceptance: `make lint`, `make fmt`, `make test` all defined in `Makefile` and runnable.
      Blocks: []
      Blocked by: [T002]

Deliverable: controller builds, CRD manifests can be generated, and the
reconciler observes `HybridWorkload` events **and writes a non-empty status**
with `phase=Pending` and a placeholder `SchedulingDecision` condition. Phase 1
is not done until the reconciler updates status at least once per CR.

## Phase 2: Decision Logic

Goal: implement placement decisions without creating AWS resources.

- [ ] T010 Implement scheduler interfaces for metrics, pricing, and health checks.
      Requirement: NFR-004, NFR-005
      Acceptance: `internal/scheduler/interfaces.go` defines `MetricsClient`, `PricingClient`, `HealthChecker` interfaces; constructor takes them as parameters; no `samber/do` import in the package.
      Blocks: [T014, T015, T016]
      Blocked by: [T007]

- [ ] T011 [P] Implement Proxmox metrics client.
      Requirement: FR-002 (utilization input)
      Acceptance: `internal/metrics` package implements `MetricsClient`; reads node metrics via Kubernetes Metrics API; unit tests use a fake metrics client.
      Blocks: [T014]
      Blocked by: [T010]

- [ ] T012 [P] Implement AWS pricing client with cache.
      Requirement: FR-004, NFR-003
      Acceptance: `internal/cost` package implements `PricingClient`; cache TTL configurable; mock pricing client returns deterministic prices in unit tests; no static AWS credentials in code.
      Blocks: [T014, T021]
      Blocked by: [T010]

- [ ] T013 [P] Implement VPN health checker.
      Requirement: FR-005
      Acceptance: `internal/healthcheck` package implements `HealthChecker`; returns `Healthy/Unhealthy/Unknown`; unit tests cover all three outcomes.
      Blocks: [T014]
      Blocked by: [T010]

- [ ] T014 Implement decision engine from [detailed-design.md#scheduler-interfaces](detailed-design.md#scheduler-interfaces).
      Requirement: FR-002, FR-003, FR-004, FR-005 (and [P-004](constitution.md))
      Acceptance: `internal/scheduler/decision.go` returns a `PlacementDecision`; consumes config thresholds and the three interfaces; pure function with no side effects.
      Blocks: [T016, T017]
      Blocked by: [T010, T011, T012, T013]

- [ ] T015 Add typed retryable errors from [detailed-design.md#error-semantics](detailed-design.md#error-semantics).
      Requirement: FR-005, FR-008
      Acceptance: `internal/scheduler/errors.go` defines `ProxmoxUnavailableError`, `VPNUnhealthyError`, `BudgetExceededError`, `KarpenterTimeoutError`, `PricingAPIUnavailableError`; each carries the retry interval declared in the contract.
      Blocks: [T017]
      Blocked by: [T010]

- [ ] T016 Add table-driven decision engine tests.
      Requirement: FR-002, FR-003, NFR-004 (and [P-004](constitution.md))
      Acceptance: `internal/scheduler/decision_test.go` covers all 5 acceptance scenarios from FR-002 plus the 3 hysteresis scenarios from FR-003 in a table-driven test; package coverage ≥ 80%.
      Blocks: []
      Blocked by: [T014]

- [ ] T017 Update reconciler to write placement decision to status.
      Requirement: FR-002, FR-008
      Acceptance: `internal/controller/hybridworkload_controller.go` writes `recommendedPlatform`, `phase`, `vpnHealth`, `dryRun`, conditions, and `lastReconcileTime` on every successful reconciliation; the integration test asserts a non-empty status.
      Blocks: [T021]
      Blocked by: [T006, T014, T015]

Deliverable: controller logs and publishes correct placement decisions for all
acceptance scenarios in FR-002 and FR-003.

## Phase 3: AWS Burst

Goal: create AWS burst capacity through Karpenter.

- [ ] T018a Implement Karpenter NodePool create/update from [detailed-design.md#karpenter-contract](detailed-design.md#karpenter-contract).
      Requirement: FR-006 (and [P-003](constitution.md))
      Acceptance: `internal/karpenter` package can create or update a NodePool named `hcro-<workload>` with capacity type, instance type, taint, and node class from the contract; unit tests use a fake controller-runtime client.
      Blocks: [T018b, T021, T022]
      Blocked by: [T014]

- [ ] T018b Set owner references on owned NodePools.
      Requirement: FR-006
      Acceptance: created NodePools carry an `ownerReference` to the parent HybridWorkload when the API allows it; integration test verifies the reference appears.
      Blocks: [T020]
      Blocked by: [T018a]

- [ ] T018c Delete owned NodePool via finalizer.
      Requirement: FR-006
      Acceptance: deleting a HybridWorkload deletes the owned NodePool before the project finalizer is removed; integration test reproduces the sequence.
      Blocks: []
      Blocked by: [T018b, T020]

- [ ] T019 Add dry-run behavior.
      Requirement: FR-007
      Acceptance: when annotation `hcro.io/dry-run=true` is present, no NodePool is created/updated/deleted by the reconciler; `status.dryRun=true`.
      Blocks: [T022]
      Blocked by: [T017, T018a]

- [ ] T020 Add finalizer cleanup for owned NodePools.
      Requirement: FR-006
      Acceptance: the controller adds the project finalizer before NodePool creation and removes it only after NodePool deletion; integration test asserts both transitions.
      Blocks: [T018c]
      Blocked by: [T018a]

- [ ] T021 Wire NodePool and cost estimate status fields.
      Requirement: FR-004, FR-006, FR-008
      Acceptance: `status.karpenterNodePoolName` and `status.estimatedMonthlyCostCents` populated whenever a NodePool exists; integration test asserts both fields after reconcile.
      Blocks: []
      Blocked by: [T012, T017, T018a]

- [ ] T022 Add integration tests for AWS placement path.
      Requirement: FR-006, NFR-004
      Acceptance: envtest suite covers create, update, delete, and dry-run paths; `make test` includes the suite.
      Blocks: []
      Blocked by: [T018a, T019, T020]

Deliverable: AWS placement creates or updates the expected Karpenter NodePool;
deletion cleans up; dry-run path skips external mutations.

## Phase 4: Production Readiness

Goal: make the MVP deployable and observable.

- [ ] T023 Add Prometheus metrics.
      Requirement: NFR-002
      Acceptance: `/metrics` exposes `hcro_placement_total{platform=}`, `hcro_decision_duration_seconds`, `hcro_estimated_cost_cents{workload=}`; assertions in a metrics unit test.
      Blocks: []
      Blocked by: [T007, T017]

- [ ] T024 Add health and readiness probes.
      Requirement: NFR-001, NFR-002
      Acceptance: `/healthz` and `/readyz` configured via controller-runtime healthz; deployment manifest references both.
      Blocks: []
      Blocked by: [T005]

- [ ] T025 Enable leader election by default.
      Requirement: NFR-001
      Acceptance: `cmd/main.go` defaults `-leader-elect=true` for in-cluster runs; lease ID matches the deployment manifest; e2e on kind verifies leader takeover within 30s.
      Blocks: []
      Blocked by: [T005]

- [ ] T026 Add validating webhook.
      Requirement: FR-001 (semantic checks beyond CRD schema), [detailed-design.md#webhook](detailed-design.md#webhook)
      Acceptance: webhook rejects requests with zero CPU/memory requests, negative budget; admission test exercises rejection and acceptance paths.
      Blocks: []
      Blocked by: [T003]

- [ ] T027 Add RBAC manifests.
      Requirement: FR-006, FR-008, NFR-001
      Acceptance: `config/rbac/role.yaml` includes `hybridworkloads`, `hybridworkloads/status`, `hybridworkloads/finalizers`, Karpenter `nodepools`, Kubernetes `nodes`, and `metrics.k8s.io` node metrics; generated by markers, not hand-edited.
      Blocks: [T028]
      Blocked by: [T003, T018a]

- [ ] T028 Add E2E test with kind.
      Requirement: NFR-004
      Acceptance: `make test-e2e` brings up a kind cluster, installs CRDs and the controller, applies a sample HybridWorkload, asserts non-empty status and dry-run behavior.
      Blocks: []
      Blocked by: [T004, T027]

Deliverable: production-ready MVP with admission validation and basic
observability.

## Phase 5: Documentation and CLI

Goal: make the project understandable and demoable.

- [ ] T029 Add sample `HybridWorkload` manifests.
      Requirement: FR-001 (usability)
      Acceptance: `config/samples/cost_v1alpha1_hybridworkload.yaml` contains at least three samples (low/medium/high priority) with valid resources; sample applies cleanly via `kubectl apply`.
      Blocks: [T028]
      Blocked by: [T003]

- [ ] T030 Add cost savings CLI command.
      Requirement: FR-004 (reporting)
      Acceptance: a CLI subcommand prints monthly Proxmox-vs-AWS savings using current HybridWorkload status; integration test runs the command against a fake client.
      Blocks: []
      Blocked by: [T021]

- [ ] T031 Add README quickstart.
      Requirement: usability
      Acceptance: top-level `phase-1-foundation/README.md` (currently 86 bytes) covers prerequisites, `make install`, `make run`, sample apply, and inspect commands.
      Blocks: []
      Blocked by: [T029]

- [ ] T031.1 Top-level README quickstart.
      Requirement: usability
      Acceptance: `phase-1-foundation/README.md` includes a 5-step "from zero to first decision" walkthrough.
      Blocks: []
      Blocked by: [T031]

- [ ] T032 Add troubleshooting notes.
      Requirement: FR-008 (operational visibility)
      Acceptance: `docs/runbook.md` enumerates the 5 typed errors with their retry intervals, observable symptoms, and remediation steps.
      Blocks: []
      Blocked by: [T015]

- [ ] T032.1 Document failure modes for typed errors.
      Requirement: FR-005, FR-008
      Acceptance: `docs/runbook.md` covers `VPNUnhealthy`, `BudgetExceeded`, `ProxmoxUnavailable`, `KarpenterTimeout`, `PricingAPIUnavailable` with retry intervals from the contract.
      Blocks: []
      Blocked by: [T032]

Deliverable: a user can run a local demo and understand placement decisions.
