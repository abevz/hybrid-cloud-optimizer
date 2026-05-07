# Constitution

These principles are immutable for the lifetime of this project.
Changing any principle requires an explicit decision recorded under
[Amendment Log](#amendment-log).

## Principles

### P-001 Provider wiring stays in `provider.go`

The project MUST keep all `samber/do` (or any other dependency injection
container) imports inside `provider.go` files only. Core business code MUST NOT
import the DI container.

Rationale: prevents the DI container from leaking into testable units; keeps
service code constructable with explicit dependencies.

Source: NFR-005 in [requirements.md](requirements.md), Dependency Injection
section in [design.md](design.md).

### P-002 Money values are integer cents

The CRD-facing API MUST represent monetary values as `int64` cents. The project
MUST NOT introduce `float64` fields into any API type.

Rationale: `controller-gen` warns that floating-point CRD fields are dangerous
and can reject `Minimum` validations. Integer cents avoid both the validation
problem and rounding drift.

Source: FR-001, FR-004 in [requirements.md](requirements.md), API Contract in
[detailed-design.md](detailed-design.md).

### P-003 Karpenter version comes from `go.mod`

The project MUST take the Karpenter API version from the dependency declared in
`go.mod`. The project MUST NOT hardcode a Karpenter API version in YAML
manifests, prose, or fallback constants.

Rationale: Karpenter has churned across `v1alpha5`, `v1beta1`, and `v1`. Pinning
to the resolved dependency keeps manifests, generated code, and tests in sync.

Source: Karpenter NodePool Contract in
[detailed-design.md](detailed-design.md).

### P-004 Placement decisions are covered by table-driven tests

Every placement decision branch in `internal/scheduler` MUST have a
table-driven unit test. New decision branches MUST land in the same commit as
their test row.

Rationale: the decision engine is the riskiest piece of business logic; without
a table-driven matrix, behavior drifts silently across refactors.

Source: NFR-004 in [requirements.md](requirements.md), Testing Contract in
[detailed-design.md](detailed-design.md).

### P-005 AWS credentials use IRSA in production

Production deployments MUST use IRSA (IAM Roles for Service Accounts) for AWS
credentials. The project MUST NOT rely on hardcoded credentials, static access
keys, or credential files outside of local development.

Rationale: IRSA is the only credential mechanism that scales without a
long-lived secret; static keys are an audit and rotation hazard.

Source: NFR-003 in [requirements.md](requirements.md).

### P-006 Service code accepts dependencies through constructors

Core services MUST receive their dependencies (clients, config, clock) through
explicit constructor parameters. Services MUST NOT pull dependencies from a
global registry or service locator at call time.

Rationale: constructor injection makes business logic mockable without the DI
container and forces dependency surfaces to be visible at review time.

Source: NFR-004, NFR-005 in [requirements.md](requirements.md).

### P-007 Contracts are written before code

Every implementation task MUST cite a requirement (`FR-` or `NFR-`) and a
detailed-design section. If a contract gap is discovered during implementation,
work stops and `detailed-design.md` is updated first.

Rationale: SDD requires the spec to be the source of truth. Inventing a field
or behavior in code makes the spec a fiction.

Source: working rule in [README.md](README.md).

## Amendment Log

- 2026-05-07: initial constitution drafted from existing requirements and
  design decisions.
