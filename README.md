# fault

Error handling for Go: stack-traced errors, semantic tags (user fault, auth, not-found, retryable), cause coalescing, and panic recovery — built on `github.com/pkg/errors`.

## Install

```sh
go get github.com/gleno/fault
```

## What it gives you

- **Stack-traced errors** — `Errorf` and `Wrap` capture a stack trace at the point of creation. `Wrap` is idempotent: it won't add a redundant trace when the cause already carries one originating in your goroutine.
- **Sentinel errors** — `Sentinel` creates a comparable, stack-free error value you can match with `errors.Is`, and decorate with a cause via `.From(...)`.
- **Semantic tags** — tag any error as user fault, auth, not-found, or retryable (`TagAsUserFault`, `TagAsAuthError`, `TagAsNotFound`, `TagAsRetryable`), then check it with `IsUserFault`, `IsAuthFault`, `IsNotFoundFault`, `IsRetryable`. Tags double as HTTP status codes.
- **Cause coalescing** — `Coalesce` joins multiple errors into one that `errors.Is` / `errors.As` can still traverse.
- **Panic recovery** — `RecoverPanic` converts a recovered panic into a stack-traced error, with the trace pointing at the panic site rather than the recovery plumbing.
- **Full error rendering** — `GetFullError` renders the message chain plus stack trace for logging.

The `Fault` interface (returned by most constructors) is a chainable `error` with `From`, `WithMessage`, and the `As*` tagging methods.

## Usage

### Creating and wrapping errors with stack traces

```go
import "github.com/gleno/fault"

func loadUser(id string) error {
	row, err := db.Query(id)
	if err != nil {
		// Decorates the message and attaches a stack trace if one isn't already present.
		return fault.Wrap(err, "loading user")
	}
	_ = row
	return nil
}

func validate(name string) error {
	if name == "" {
		// Errorf starts a fresh stack trace at this line.
		return fault.Errorf("name is required")
	}
	return nil
}
```

`Wrapf` is the formatting variant of `Wrap`. Both return `nil` when the cause is `nil`, so they're safe to use unconditionally.

### Sentinel errors

```go
var ErrNotFound = fault.Sentinel("not found")

func lookup(id string) error {
	row, err := db.Query(id)
	if err != nil {
		return ErrNotFound.From(err, "lookup user")
	}
	_ = row
	return nil
}

// Caller:
if err := lookup("42"); errors.Is(err, ErrNotFound) {
	// handle the not-found case
}
```

Sentinels carry no stack trace. `From` attaches a cause (and its trace); `WithMessage` adds a message layer while still matching the sentinel via `errors.Is`.

### Tagging errors and checking tags

```go
func charge(amount int) error {
	if amount < 0 {
		return fault.TagAsUserFault(
			fault.Errorf("amount %d is negative", amount),
			"amount must be positive",
		)
	}
	if err := gateway.Charge(amount); err != nil {
		return fault.TagAsRetryable(err, "payment gateway unavailable")
	}
	return nil
}

// At an HTTP boundary:
err := charge(-1)
switch {
case fault.IsUserFault(err):
	// 400 — fault.UserFault
case fault.IsAuthFault(err):
	// 401 — fault.AuthFault
case fault.IsNotFoundFault(err):
	// 404 — fault.NotFoundFault
case fault.IsRetryable(err):
	// 504 — fault.RetryableFault
default:
	// 500 — fault.InternalFault (the default for untagged errors)
}

reason := fault.GetTagMessage(err) // the human reason passed when tagging
```

The tag constants are HTTP status codes (`UserFault == http.StatusBadRequest`, etc.), so you can map them directly onto responses. There are also one-shot constructors that create and tag in a single call: `MakeUserErrorf`, `MakeAuthErrorf`, `MakeRetryableErrorf`, and `MissingValue`.

### Coalescing multiple causes

```go
func runAll(tasks []Task) error {
	var errs []error
	for _, t := range tasks {
		errs = append(errs, t.Run())
	}
	// Drops nils; returns nil if all succeeded, the single error if only one failed,
	// or a combined error otherwise.
	return fault.Coalesce(errs...)
}
```

The combined error joins messages with `"; "`, and `errors.Is` / `errors.As` traverse into each underlying error — so a tag on any member is still detectable (e.g. `fault.IsRetryable` on the coalesced result).

### Recovering from panics

```go
func process() (err error) {
	defer func() {
		fault.RecoverPanic(recover(), &err)
	}()

	mightPanic()
	return nil
}
```

If the deferred call catches a panic, `err` is set to a stack-traced error (`"caught panic: ..."`) whose trace points at the line that panicked. With no panic, `err` is left untouched.

### Rendering for logs

```go
log.Print(fault.GetFullError(err))
```

`GetFullError` writes the stack trace (if any) followed by every message in the wrapped chain.

## API reference

Constructors and helpers:

| Function | Signature | Purpose |
| --- | --- | --- |
| `Errorf` | `(msg string, args ...any) error` | New stack-traced error |
| `Wrap` | `(cause error, msg string) error` | Wrap with message, idempotent stack trace |
| `Wrapf` | `(cause error, msg string, args ...any) error` | Formatting variant of `Wrap` |
| `Sentinel` | `(msg string) Fault` | Comparable, stack-free error value |
| `From` | `(err error, message string) Fault` | Wrap a cause behind a message |
| `Coalesce` | `(errs ...error) error` | Combine multiple errors into one |
| `RecoverPanic` | `(r any, errPtr *error)` | Turn a recovered panic into an error |
| `GetStackTrace` | `(err error) StackTrace` | Extract the stack trace, or `nil` |
| `GetFullError` | `(err error) string` | Render message chain + stack trace |
| `MakeUserErrorf` / `MakeAuthErrorf` / `MakeRetryableErrorf` | `(format string, args ...any) Fault` | Create and tag in one call |
| `MissingValue` | `(msg string) Fault` | Sentinel pre-tagged as not-found |

Tagging:

| Function | Signature | Check |
| --- | --- | --- |
| `TagAsUserFault` | `(err error, reason string) Fault` | `IsUserFault(err) bool` |
| `TagAsAuthError` | `(err error, reason string) Fault` | `IsAuthFault(err) bool` |
| `TagAsNotFound` | `(err error, reason string) Fault` | `IsNotFoundFault(err) bool` |
| `TagAsRetryable` | `(err error, reason string) Fault` | `IsRetryable(err) bool` |
| `GetTagMessage` | `(err error) string` | the reason passed when tagging |

Tagging `nil` returns `nil`. Untagged errors report as `InternalFault`.

Types:

- `Fault` — chainable `error` interface: `From(err error, message string) Fault`, `WithMessage(message string) Fault`, `AsUserFault`, `AsValueMissing`, `AsRetryable`, `AsAuthFault`, plus `Is` / `Unwrap`.
- `Tag` — `int` alias over HTTP status codes: `InternalFault` (500), `RetryableFault` (504), `UserFault` (400), `AuthFault` (401), `NotFoundFault` (404).
- `StackTrace` — alias for `errors.StackTrace` from `github.com/pkg/errors`.
- `StackTracer` — `interface { StackTrace() errors.StackTrace }`.
