# Tasks

## Phase 1: Foundation

Goal: establish a compilable operator skeleton and API shape.

- [x] T001 Initialize or normalize Go module layout.
- [x] T002 Add Kubebuilder/controller-runtime project skeleton.
- [ ] T003 Define `HybridWorkload` API types.
- [x] T004 Add CRD generation target.
- [x] T005 Add controller manager entrypoint.
- [x] T006 Add basic reconciler that watches `HybridWorkload`.
- [ ] T007 Add typed config loading.
- [ ] T008 Add unit tests for config loading and validation.
- [x] T009 Add lint, format, and test targets.

Deliverable: controller builds, CRD manifests can be generated, and reconciler can
observe `HybridWorkload` events.

## Phase 2: Decision Logic

Goal: implement placement decisions without creating AWS resources.

- [ ] T010 Define scheduler interfaces for metrics, pricing, and health checks.
- [ ] T011 Implement Proxmox metrics client.
- [ ] T012 Implement AWS pricing client with cache.
- [ ] T013 Implement VPN health checker.
- [ ] T014 Implement decision engine.
- [ ] T015 Add typed retryable errors.
- [ ] T016 Add table-driven decision engine tests.
- [ ] T017 Update reconciler to write placement decision to status.

Deliverable: controller logs and publishes correct placement decisions.

## Phase 3: AWS Burst

Goal: create AWS burst capacity through Karpenter.

- [ ] T018 Implement Karpenter NodePool manager.
- [ ] T019 Add dry-run behavior.
- [ ] T020 Add finalizer cleanup for owned NodePools.
- [ ] T021 Add status fields for NodePool and cost estimate.
- [ ] T022 Add integration tests for AWS placement path.

Deliverable: AWS placement creates or updates the expected Karpenter NodePool.

## Phase 4: Production Readiness

Goal: make the MVP deployable and observable.

- [ ] T023 Add Prometheus metrics.
- [ ] T024 Add health and readiness probes.
- [ ] T025 Enable leader election by default.
- [ ] T026 Add validating webhook.
- [ ] T027 Add RBAC manifests.
- [ ] T028 Add E2E test with kind.

Deliverable: production-ready MVP with admission validation and basic
observability.

## Phase 5: Documentation and CLI

Goal: make the project understandable and demoable.

- [ ] T029 Add sample `HybridWorkload` manifests.
- [ ] T030 Add cost savings CLI command.
- [ ] T031 Add README quickstart.
- [ ] T032 Add troubleshooting notes.

Deliverable: a user can run a local demo and understand placement decisions.
