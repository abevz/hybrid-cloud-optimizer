# Testing Strategy

## Test Pyramid

The MVP should start with fast unit tests and add Kubernetes integration tests
only where the API server behavior matters.

```text
E2E: kind cluster, full controller path
Integration: envtest, fake Kubernetes API server
Unit: pure Go logic with mocks
```

## Unit Tests

Run with:

```bash
go test -race ./...
```

Unit tests should cover:

- config defaults and validation
- decision engine placement matrix
- hysteresis behavior
- budget blocking
- VPN health blocking
- AWS price parsing
- structured error classification

Use table-driven tests with named subtests.

## Integration Tests

Integration tests should use `envtest` when controller-runtime reconciliation or
Kubernetes API behavior matters.

Run with:

```bash
go test -v ./test/integration/...
```

Integration tests should cover:

- creating a `HybridWorkload`
- status update after reconciliation
- dry-run behavior
- finalizer behavior
- Karpenter NodePool create/update/delete behavior with fake API objects

## E2E Tests

E2E tests should use a local `kind` cluster.

Run with:

```bash
go test -v ./test/e2e/...
```

E2E tests should cover:

- install CRD and controller
- apply sample `HybridWorkload`
- observe status decision
- validate dry-run does not create external resources

## Manual Verification

Until the controller skeleton exists, the only meaningful local verification is:

```bash
go test ./...
go build -o /tmp/hcro-controller ./cmd/controller
```

After Phase 1, manual verification should include applying the CRD to a local
cluster and watching reconciler logs.
