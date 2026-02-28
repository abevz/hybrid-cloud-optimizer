# samber/do v2 Conventions

This document defines dependency injection conventions for the hybrid-cloud-optimizer project using `samber/do/v2`. It covers HOW to structure DI code, not WHAT components to build (see `MVP.md` for implementation specs).

## The Golden Rule

**`samber/do/v2` imports are allowed ONLY in `provider.go` files.**

This is non-negotiable. Service code, models, tests, and internal helpers must never import `samber/do`.

## File Structure

Each package with DI follows this layout:

```
internal/mypackage/
├── provider.go    # ONLY file with do/v2 import
├── service.go     # Business logic, pure constructor
├── model.go       # Domain models (optional)
└── internal.go    # Unexported helpers (optional)
```

## Provider Function Rules

### 1. Single Provider Per Package

```go
// GOOD: One provider per package
func Provider(i do.Injector) (*Service, error)

// BAD: Multiple providers
func PrimaryProvider(i do.Injector) (*Service, error)
func ReplicaProvider(i do.Injector) (*Service, error)
```

Use Config struct for variations (see closure pattern below).

### 2. Closure-Based Configuration

Config is passed from `main.go` via closures. **Never read `os.Getenv()` in providers.**

```go
// provider.go
type Config struct {
    APIKey  string
    BaseURL string
}

func Provider(cfg Config) func(do.Injector) (*Service, error) {
    return func(i do.Injector) (*Service, error) {
        logger := do.MustInvoke[*slog.Logger](i)
        logger = logger.With("component", "myservice")
        return NewService(logger, cfg.APIKey, cfg.BaseURL), nil
    }
}

// main.go — environment variables read HERE
do.Provide(injector, myservice.Provider(myservice.Config{
    APIKey:  os.Getenv("API_KEY"),
    BaseURL: os.Getenv("BASE_URL"),
}))
```

```go
// BAD: Reading env in provider
func Provider(i do.Injector) (*Service, error) {
    apiKey := os.Getenv("API_KEY")  // WRONG! Read in main.go
}
```

### 3. Component Logger Pattern

Always create a child logger with component label:

```go
func Provider(i do.Injector) (*Service, error) {
    baseLogger := do.MustInvoke[*slog.Logger](i)
    logger := baseLogger.With("component", "decision-engine")
    return NewService(logger), nil
}
```

### 4. Dependency Resolution Order

Resolve dependencies at the start of the provider, in consistent order:

```go
func Provider(i do.Injector) (*Service, error) {
    // 1. Logger first
    logger := do.MustInvoke[*slog.Logger](i)
    logger = logger.With("component", "myservice")

    // 2. Other dependencies
    bus := do.MustInvoke[*eventbus.EventBus](i)
    cache := do.MustInvoke[Cache](i)

    // 3. Create service
    return NewService(logger, bus, cache), nil
}
```

### 5. Optional Dependencies

Return `nil, nil` for disabled features. Use pointer types (not interfaces) so nil is safe:

```go
func Provider(cfg Config) func(do.Injector) (*Feature, error) {
    return func(i do.Injector) (*Feature, error) {
        if !cfg.Enabled {
            return nil, nil  // disabled, no error
        }
        logger := do.MustInvoke[*slog.Logger](i)
        return NewFeature(logger), nil
    }
}
```

```go
// GOOD: Pointer type — nil is safe
func Provider(...) (*Upgrader, error)

// BAD: Interface type — nil interface panics on method call!
func Provider(...) (Upgrader, error)
```

### 6. Error Handling in Providers

Always wrap errors with context:

```go
func Provider(i do.Injector) (*Service, error) {
    db, err := do.Invoke[*sql.DB](i)
    if err != nil {
        return nil, fmt.Errorf("resolving database: %w", err)
    }

    svc, err := NewService(db)
    if err != nil {
        return nil, fmt.Errorf("creating service: %w", err)
    }

    return svc, nil
}
```

## Pure Constructors

Service constructors must be pure — no DI framework, no env vars, no side effects:

```go
// service.go — NO do/v2 import!
package myservice

import "log/slog"

type EventPublisher interface {
    Publish(event any) error
}

type Service struct {
    logger *slog.Logger
    events EventPublisher  // interface, not concrete
}

// Pure constructor — explicit dependencies
func NewService(logger *slog.Logger, events EventPublisher) *Service {
    return &Service{
        logger: logger,
        events: events,
    }
}
```

```go
// BAD: Service using do/v2
func (s *Service) DoWork() {
    cache := do.MustInvoke[Cache](nil)  // WRONG!
}

// GOOD: Dependency injected via constructor
func (s *Service) DoWork() {
    s.cache.Get(...)  // use injected dependency
}
```

## Interface Adapter Pattern

When a consumer needs a different interface than what a concrete service provides, use an adapter with compile-time check:

```go
// adapters/graph.go
type graphAuthProvider struct {
    svc *auth.Service
}

// Compile-time interface check
var _ graph.Auth = (*graphAuthProvider)(nil)

func (p *graphAuthProvider) UserFromContext(ctx context.Context) *model.User {
    return p.svc.UserFromContext(ctx)
}

// Provider returns interface type, not concrete
func AuthAdapterProvider(i do.Injector) (graph.Auth, error) {
    svc := do.MustInvoke[*auth.Service](i)
    return &graphAuthProvider{svc: svc}, nil
}
```

## Service Isolation

**Services must NOT import other service packages directly.** Use interfaces and wire in `provider.go`.

### Import Rules

| File | Can import other services? | Reason |
|------|---------------------------|--------|
| `main.go` | Yes | Composition root |
| `provider.go` | Yes | DI wiring only |
| `service.go` | **No** | Business logic |
| `model.go` | **No** | Domain models |
| `*_test.go` | **No** | Tests use mocks |

### Consumer Defines Interface

```go
// internal/scheduler/service.go — defines what IT needs
type MetricsReader interface {
    GetUtilization(ctx context.Context) (float64, error)
}

type DecisionEngine struct {
    metrics MetricsReader  // interface, not *metrics.Client
}

func NewDecisionEngine(metrics MetricsReader) *DecisionEngine {
    return &DecisionEngine{metrics: metrics}
}
```

```go
// internal/scheduler/provider.go — wires concrete to interface
func Provider(i do.Injector) (*DecisionEngine, error) {
    metricsClient := do.MustInvoke[*metrics.Client](i)  // concrete
    return NewDecisionEngine(metricsClient), nil          // satisfies interface
}
```

## Testing Without DI

Services remain testable without the DI container:

```go
// service_test.go — NO do/v2 import!
func TestDecisionEngine(t *testing.T) {
    mockMetrics := &mockMetricsReader{utilization: 0.50}
    engine := NewDecisionEngine(mockMetrics)

    decision, err := engine.Decide(context.Background())
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if decision.Platform != "proxmox" {
        t.Errorf("got %q, want proxmox", decision.Platform)
    }
}
```

## DI Patterns Reference

### Basic Provider

```go
func Provider(i do.Injector) (*Service, error) {
    logger := do.MustInvoke[*slog.Logger](i)
    return NewService(logger), nil
}
```

### Named Instances

For multiple instances of the same type (e.g., primary and replica databases):

```go
// main.go
do.ProvideNamed(injector, "primary-db", db.Provider(primaryCfg))
do.ProvideNamed(injector, "replica-db", db.Provider(replicaCfg))

// usage
primary := do.MustInvokeNamed[*DB](injector, "primary-db")
```

### Factory Pattern

Internal implementation selection based on config:

```go
func Provider(cfg Config) func(do.Injector) (*Service, error) {
    return func(i do.Injector) (*Service, error) {
        var notifier Notifier
        if cfg.TelegramToken != "" {
            notifier = newTelegramNotifier(cfg.TelegramToken)
        } else {
            notifier = newConsoleNotifier()
        }
        return NewService(notifier), nil
    }
}
```

### Health Check Registration

```go
func (s *Service) HealthCheck(ctx context.Context) error {
    return s.db.PingContext(ctx)
}

func Provider(i do.Injector) (*Service, error) {
    svc, err := NewService(...)
    if err != nil {
        return nil, err
    }
    checker := do.MustInvoke[HealthRegistry](i)
    checker.Register("myservice", svc)
    return svc, nil
}
```

## Provider Checklist

Before committing provider code, verify:

- [ ] `samber/do` imported ONLY in `provider.go`
- [ ] Service has pure constructor with explicit deps
- [ ] Config passed via closure (no `os.Getenv` in provider)
- [ ] Logger has component label
- [ ] One Provider function per package
- [ ] Optional deps return `nil, nil` (pointer types)
- [ ] Errors wrapped with context
- [ ] Interface adapters have compile-time check (`var _ Interface = (*impl)(nil)`)
- [ ] Service has no `do/v2` imports
- [ ] Service depends on interfaces, not concrete types
