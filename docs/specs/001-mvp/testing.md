# Testing Strategy

## Test Pyramid

The MVP should start with fast unit tests and add Kubernetes integration tests
only where Kubernetes API server behavior matters.

```text
E2E: kind cluster, full controller deployment path
Integration: envtest, local API server and etcd
Unit: pure Go logic with mocks
```

## Canonical Commands

Use Kubebuilder Makefile targets as the project test entrypoints.

```bash
make test
```

**make test** runs manifest generation, deepcopy generation, formatting, vet,
envtest setup, and Go tests excluding E2E packages.

Do not use go test ./... as the default project-wide command after Kubebuilder
scaffolding. Controller tests use envtest and require Kubernetes test binaries
through KUBEBUILDER_ASSETS or make setup-envtest.

For a fast pure-Go package check that skips envtest and E2E packages:

```bash
go test $(go list ./... | grep -v /internal/controller | grep -v /test/e2e)
```

## Unit Tests

Unit tests should cover pure business logic and avoid Kubernetes API server
startup when possible.

Unit test targets include:

- config defaults and validation
- decision engine placement matrix
- hysteresis behavior
- budget blocking
- VPN health blocking
- AWS price parsing
- structured error classification

Use table-driven tests with named subtests.

## Coverage Targets

| Package | Target | Rationale |
|---------|--------|-----------|
| `internal/config` | ≥ 90% | small surface; high blast radius (start-time validation) |
| `internal/scheduler` | ≥ 80% | pure business logic; table-driven tests are cheap |
| `internal/cost` | ≥ 75% | external API; mock-driven testing |
| `internal/healthcheck` | ≥ 75% | network-bound; covers Healthy/Unhealthy/Unknown states |
| `internal/karpenter` | ≥ 70% | Kubernetes API interactions; envtest-driven |
| `internal/controller` | ≥ 70% | reconciler glue; envtest-driven |

Coverage is measured by `go test -cover`. Falling below the target on `main`
should fail CI.

## Test → Requirement Map

| Test File | Tests | Requirement | Detailed-Design Anchor |
|-----------|-------|-------------|------------------------|
| `internal/config/config_test.go` | defaults, threshold pair validation, env override, missing VPN endpoint | NFR-004, NFR-005, FR-003 | [#config-inventory](detailed-design.md#config-inventory) |
| `internal/scheduler/decision_test.go` | placement matrix, hysteresis, budget blocking, VPN gate | FR-002, FR-003, FR-005 | [#scheduler-interfaces](detailed-design.md#scheduler-interfaces) |
| `internal/scheduler/errors_test.go` | typed error classification and retry intervals | FR-005, FR-008 | [#error-semantics](detailed-design.md#error-semantics) |
| `internal/cost/aws_pricing_client_test.go` | on-demand parse, spot discount, cache hit/miss | FR-004 | [#scheduler-interfaces](detailed-design.md#scheduler-interfaces) |
| `internal/healthcheck/vpn_test.go` | Healthy/Unhealthy/Unknown outcomes | FR-005 | [#error-semantics](detailed-design.md#error-semantics) |
| `internal/karpenter/nodepool_test.go` | create, update, delete, owner reference, version assertion | FR-006 ([P-003](constitution.md)) | [#karpenter-contract](detailed-design.md#karpenter-contract) |
| `internal/controller/hybridworkload_controller_test.go` | first reconcile writes status, dry-run path, finalizer | FR-007, FR-008 | [#controller-flow](detailed-design.md#controller-flow) |
| `test/e2e/e2e_test.go` | kind: install CRDs, apply sample, observe status | NFR-004 | [#testing-contract](detailed-design.md#testing-contract) |

## Integration Tests

Integration tests use envtest for controller-runtime reconciliation and
Kubernetes API behavior.

Run with:
```bash
make test
```

Integration tests should cover:

- creating a HybridWorkload
- status update after reconciliation
- dry-run behavior
- finalizer behavior
- Karpenter NodePool create/update/delete behavior with fake API objects

## E2E Tests

E2E tests use a local kind cluster and the Kubebuilder scaffolded E2E target.

Run with:
```bash
make test-e2e
```

E2E tests should cover:

- installing CRDs and controller manifests
- applying a sample HybridWorkload
- observing status decisions
- validating dry-run behavior

## CI Plan

| Workflow File | Job | Gate |
|---------------|-----|------|
| `.github/workflows/test.yml` | `make test` (unit + envtest) | merge blocker on PRs to `main` |
| `.github/workflows/lint.yml` | `golangci-lint`, `make manifests` drift check | merge blocker |
| `.github/workflows/test-e2e.yml` | `make test-e2e` on kind | merge blocker on PRs touching `internal/controller`, `config/`, `cmd/` |

CI rules:

- `make manifests` MUST be a no-op on a clean checkout. Drift between markers
  and generated YAML fails the build.
- Lint configuration lives in `.golangci.yml`. Adding a new linter requires a
  PR against the constitution (review the Amendment Log).
- The Karpenter version assertion test (see
  [detailed-design.md#karpenter-contract](detailed-design.md#karpenter-contract))
  runs in `make test` and fails the build on version skew.

## Manual Verification

For local development against the current Kubernetes context:
```bash
make build
make install
make run
```
**make install** installs CRDs into the current kubectl context. make run
starts the controller locally while it talks to that cluster through kubeconfig.

For the Proxmox lab cluster, first confirm the context:
```bash
kubectl config current-context
```

Then install the CRD and run the controller locally:
```bash
make install
make run
```

In another terminal, apply a sample resource and inspect it:
```bash
kubectl apply -f config/samples/cost_v1alpha1_hybridworkload.yaml
kubectl get hybridworkloads.cost.hybrid.io
kubectl describe hybridworkload <name>
```


## Verification

After editing:

```bash
git diff -- docs/specs/001-mvp/testing.md
make test
```
If envtest binaries are missing, run:
```bash
make setup-envtest
make test
```

## Assumptions

- Kubebuilder scaffold remains the canonical project structure.
- E2E tests should use make test-e2e, not direct go test.
- Proxmox/k8s-lab is used for manual smoke testing, not as a replacement for envtest.
