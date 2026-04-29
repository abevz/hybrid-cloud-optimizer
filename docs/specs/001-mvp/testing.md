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
