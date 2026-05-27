# `From(nil, msg)` produces an error whose `.Error()` panics (nil cause deref)

## Area
fault.go / error_cause.go — package-level `From` + `_ErrorWithCause.Error()`

## Hypothesis
The package-level constructor `From(err, message)` (fault.go:34) stores its
`err` argument directly as the `cause` field with **no nil guard**:

```go
func From(err error, message string) Fault {
	return &_ErrorWithCause{
		_Error: _Error{msg: message},
		cause:  err,
	}
}
```

`_ErrorWithCause.Error()` (error_cause.go:13) unconditionally dereferences the
cause:

```go
func (s *_ErrorWithCause) Error() string {
	return fmt.Sprintf("%s: %s", s.msg, s.cause.Error())
}
```

So `From(nil, "context").Error()` calls `nil.Error()` → nil-pointer panic.

## Observed
`go test` panics:

```
panic: runtime error: invalid memory address or nil pointer dereference
github.com/gleno/fault.(*_ErrorWithCause).Error(...)
	error_cause.go:13
```

A nil dereference panic, not a returned error string.

## Expected
Calling `.Error()` on any constructed `Fault` must never panic. A sensible
contract is one of:
- `From(nil, msg)` returns `nil` (mirroring `Wrap(nil, _) == nil`), or
- `From(nil, msg)` returns a plain message-only fault equivalent to
  `Sentinel(msg)` (cause omitted), or
- `_ErrorWithCause.Error()` is made nil-cause-safe (render just `s.msg` when
  `cause == nil`).

## Related surface (same defect class)
The builder methods that route through `Wrap` also leave a nil cause when given
nil, because `Wrap(nil, _)` returns nil:
- `_Error.From` (error.go:40)
- `_ErrorWithCause.From` (error_cause.go:27)
- `_ErrorWithTag.From` (tagged_error.go:46)

e.g. `Sentinel("base").From(nil, "ctx").Error()` panics the same way. A robust
fix should make `_ErrorWithCause.Error()` nil-safe (catches the whole class at
the render site) AND/OR guard the constructors. Patch the class, not just the
package-level `From`.

## Why filed (not fixed)
The correct behavior is a genuine API/semantics judgement call — return nil vs.
degrade to a message-only fault vs. nil-safe render all change observable
behavior for callers. Multiple defensible choices, so leaving the decision to
the fixer.

## Repro
Add `repro_test.txt` (in this folder) as `from_nil_test.go` in package `fault`
at repo root, then `go test ./...`. It panics at error_cause.go:13.

---

## Class extension (dig 2026-05-27-1128): the whole builder family panics

A class-level nil-input audit confirms this is not one defect — **every builder
that takes an `error`/cause and later renders it panics on a nil argument.**
All `*.From` methods build `cause: Wrap(err, message)`, and `Wrap(nil, _)`
returns `nil` (stack.go:118), so the `cause` field becomes nil and
`_ErrorWithCause.Error()` (error_cause.go:13) dereferences it.

Verified RED — all 6 subtests panic with
`runtime error: invalid memory address or nil pointer dereference`:

| Variant | Call | Panic site |
|---|---|---|
| package-level (already filed) | `From(nil, "msg").Error()` | error_cause.go:13 |
| `_Error.From` | `Sentinel("base").From(nil, "ctx").Error()` | error_cause.go:13 (via Wrap nil cause) |
| `_ErrorWithCause.From` | `From(Sentinel("x"),"y").From(nil,"ctx").Error()` | error_cause.go:13 |
| `_ErrorWithTag.From` | `Sentinel("b").AsRetryable("r").From(nil,"ctx").Error()` | error_cause.go:13 |
| `_CoalesceError.From` | `Coalesce(a,b).From(nil,"ctx").Error()` | error_cause.go:13 |
| `_CoalesceError` nil member | `(&_CoalesceError{errs: []error{nil}}).Error()` | coalesce.go:18 (nil member `.Error()`) |

Affected methods (all five `From` impls plus the coalesce render):
- `From` (fault.go:34) — package-level, stores nil directly
- `_Error.From` (error.go:40)
- `_ErrorWithCause.From` (error_cause.go:27)
- `_ErrorWithTag.From` (tagged_error.go:46)
- `_CoalesceError.From` (coalesce.go:49)
- `_CoalesceError.Error` (coalesce.go:14) — `msgs[i] = err.Error()` derefs a nil
  member (defensive; `Coalesce(...)` filters nils today, but the field is
  reachable and the From paths can seed it indirectly).

### Recommended fix shape
The single highest-leverage guard is at the **render site**:
`_ErrorWithCause.Error()` should render just `s.msg` when `s.cause == nil`.
That catches all five `From` paths at once (they all funnel into
`_ErrorWithCause`). Separately, `_CoalesceError.Error()` should skip/guard nil
members. Optionally also guard the package-level `From` to mirror
`Wrap(nil,_) == nil` — but that's the API-semantics judgement call noted above,
so it stays filed rather than auto-fixed.

### Class repro
`builders_nil_cause_repro.txt` (this folder) — drop into package `fault` at repo
root as `builders_nil_cause_test.go`, then `go test -run TestBuildersNilCauseClass`.
All 6 subtests panic on the unpatched code.
