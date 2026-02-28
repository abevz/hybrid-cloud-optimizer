# Go Patterns & Conventions

This document defines Go coding conventions for the hybrid-cloud-optimizer project. It covers HOW to write code (patterns, error handling, logging), not WHAT to implement (see `MVP.md` for implementation specs).

## Core Principles

1. **Idiomatic Go** — Simplicity over cleverness, explicit error handling, composition over inheritance
2. **Production Readiness** — Structured logging, context propagation, graceful shutdown, observability
3. **Concurrency Safety** — Share memory by communicating, proper `context.Context` usage, panic recovery in all goroutines
4. **Reliability** — SafeGo pattern, circuit breakers, graceful degradation

## Error Handling

### Wrapping Errors with Context

Always add context at each layer using `%w` for wrappable errors:

```go
func (s *Service) GetUser(ctx context.Context, id int) (*User, error) {
    user, err := s.repo.FindByID(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("finding user %d: %w", id, err)
    }
    return user, nil
}
```

### Sentinel Errors

Define package-level sentinel errors for expected failure cases:

```go
var (
    ErrNotFound     = errors.New("not found")
    ErrUnauthorized = errors.New("unauthorized")
    ErrBudgetExceeded = errors.New("budget exceeded")
)
```

Check with `errors.Is`:

```go
if errors.Is(err, ErrNotFound) {
    // handle not found
}
```

### Custom Error Types

Use for errors that carry structured data (e.g., retry semantics in the controller):

```go
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}

// Check with errors.As
var verr ValidationError
if errors.As(err, &verr) {
    // handle validation error with verr.Field
}
```

### Error Aggregation

For batch operations, collect errors and join:

```go
func ProcessBatch(items []Item) error {
    var errs []error
    for _, item := range items {
        if err := process(item); err != nil {
            errs = append(errs, fmt.Errorf("item %s: %w", item.ID, err))
        }
    }
    return errors.Join(errs...)
}
```

### Anti-Patterns

```go
// BAD: Ignoring errors
_ = file.Close()

// GOOD: Handle or log
if err := file.Close(); err != nil {
    logger.Error("failed to close file", "error", err)
}

// BAD: Generic messages
return errors.New("something went wrong")

// GOOD: Specific, actionable
return fmt.Errorf("failed to parse config file %s: %w", path, err)

// BAD: String comparison
if err.Error() == "not found" { ... }

// GOOD: errors.Is / errors.As
if errors.Is(err, ErrNotFound) { ... }

// BAD: Losing context
return err

// GOOD: Wrap with context
return fmt.Errorf("processing request %s: %w", reqID, err)

// BAD: Panic for expected errors
panic("user not found")

// GOOD: Return error
return nil, ErrUserNotFound
```

## SafeGo Pattern (Panic Recovery)

**All goroutines must have panic recovery.** Never use raw `go func()` in production.

```go
// SafeGo wraps goroutine with panic recovery
func SafeGo(name string, fn func()) {
    go func() {
        defer func() {
            if r := recover(); r != nil {
                slog.Error("goroutine panic recovered",
                    "goroutine", name,
                    "panic", r,
                    "stack", string(debug.Stack()),
                )
            }
        }()
        fn()
    }()
}

// Usage
SafeGo("metrics-collector", func() {
    // panics will be caught and logged
    // application continues running
})
```

## Structured Logging with slog

### Component Logger Pattern

Each component creates a child logger with component label:

```go
// In provider.go
func Provider(i do.Injector) (*Service, error) {
    baseLogger := do.MustInvoke[*slog.Logger](i)
    logger := baseLogger.With("component", "decision-engine")
    return NewService(logger), nil
}
```

### Log Fields Convention

Always include relevant context in structured fields:

```go
logger.InfoContext(ctx, "placement decision made",
    "workload", workload.Name,
    "namespace", workload.Namespace,
    "platform", decision.Platform,
    "cost_usd", decision.EstimatedCost,
    "duration_ms", time.Since(start).Milliseconds(),
)
```

### Error Logging

```go
slog.ErrorContext(ctx, "reconciliation failed",
    "error", err,
    "workload", workload.Name,
    "namespace", workload.Namespace,
    "attempt", attempt,
)
```

## Concurrency Patterns

### Context Cancellation

Respect `context.Context` for all operations:

```go
func ProcessItems(ctx context.Context, items []Item) error {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    for _, item := range items {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
            if err := process(item); err != nil {
                return err
            }
        }
    }
    return nil
}
```

## Project Structure Convention

Each package in `internal/` follows this file layout:

```
internal/mypackage/
├── provider.go    # ONLY file with samber/do import (see samber-do.md)
├── service.go     # Business logic, pure constructor
├── model.go       # Domain models (optional)
└── internal.go    # Unexported helpers (optional)
```

- `provider.go` — DI wiring only, imports `samber/do/v2`
- `service.go` — Pure business logic, no framework dependencies
- Keep files focused; split large files by responsibility

## DevOps CLI Patterns

### Dry-Run Support

Every destructive command must support `--dry-run`:

```go
if opts.DryRun {
    logger.Info("would delete nodepool", "name", name)
    return nil
}
// Actually delete
```

### Exit Codes

- `0` — Success
- `1` — General error
- `2` — Validation error
- `3` — Configuration error
- `130` — Interrupted (Ctrl+C)

### Idempotency

Commands should be safe to run multiple times. Check current state before acting:

```go
// Check if resource already exists before creating
existing, err := client.Get(ctx, name)
if err == nil {
    logger.Info("resource already exists, skipping", "name", name)
    return existing, nil
}
if !errors.Is(err, ErrNotFound) {
    return nil, fmt.Errorf("checking resource: %w", err)
}
// Create new resource
```

## Testing Conventions

### Table-Driven Tests

All unit tests use table-driven pattern:

```go
func TestDecisionEngine(t *testing.T) {
    tests := []struct {
        name     string
        util     float64
        priority string
        want     string
        wantErr  error
    }{
        {
            name:     "low utilization places on proxmox",
            util:     0.50,
            priority: "medium",
            want:     "proxmox",
        },
        {
            name:     "high priority always places on aws",
            util:     0.10,
            priority: "high",
            want:     "aws",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := NewDecisionEngine(/* mock deps */)
            got, err := engine.Decide(tt.util, tt.priority)
            if !errors.Is(err, tt.wantErr) {
                t.Errorf("error = %v, want %v", err, tt.wantErr)
            }
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

### Race Detection

Always run tests with the race detector:

```bash
go test -race -cover ./...
```

### No DI in Tests

Tests create services directly, not through the DI container:

```go
func TestService(t *testing.T) {
    mockDep := &mockDependency{}
    svc := NewService(slog.Default(), mockDep)
    // test svc directly
}
```

## Tooling

| Tool | Command | Purpose |
|------|---------|---------|
| golangci-lint | `golangci-lint run ./...` | Linting (multiple linters) |
| gofumpt | `gofumpt -w .` | Strict formatting |
| govulncheck | `govulncheck ./...` | Vulnerability scanning |
| go test | `go test -race -cover ./...` | Testing with race detection |
| mockgen | `mockgen` | Interface mock generation |

## Critical Rules Summary

1. **Always check errors** — Never ignore `err` returns
2. **Use context.Context** — For cancellation and timeouts
3. **Avoid global state** — Pass dependencies explicitly
4. **Handle panics** — Use SafeGo for all goroutines
5. **Close resources** — Use `defer` for cleanup
6. **Avoid init()** — Prefer explicit initialization
7. **Never hardcode credentials** — Use IRSA for AWS
