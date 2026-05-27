# `errors.Is` false positive from wrapper-message equality

## Hypothesis

`_Error.Is` (error.go:13) compares two faults purely by message-string equality:

```go
func (s *_Error) Is(target error) bool {
	if t, ok := target.(*_Error); ok {
		return s.msg == t.msg
	}
	return false
}
```

`_ErrorWithCause.Is` (error_cause.go:20) compares its own wrapper `msg` to another
`*_ErrorWithCause`'s wrapper `msg`, and otherwise falls back to `_Error.Is`:

```go
func (s *_ErrorWithCause) Is(target error) bool {
	if t, ok := target.(*_ErrorWithCause); ok {
		return s.msg == t.msg
	}
	return s._Error.Is(target)
}
```

Because matching is by message text only — with no identity / sentinel-pointer
component — `errors.Is(err, sentinel)` returns `true` whenever `err`'s wrapper
message text happens to equal the sentinel's text, even when `err` has no real
relationship to that sentinel.

## Observed vs expected

False-positive scenario (both observed RED):

```go
notFound  := Sentinel("not found")
unrelated := From(Errorf("disk exploded"), "not found") // contextual msg coincidentally == sentinel text
errors.Is(unrelated, notFound)                          // OBSERVED: true   EXPECTED: false
```

```go
notFound  := Sentinel("not found")
unrelated := Sentinel("disk exploded").WithMessage("not found")
errors.Is(unrelated, notFound)                          // OBSERVED: true   EXPECTED: false
```

`unrelated` is a wholly different error that merely carries the human-readable
context string "not found"; it should not be sentinel-equal to `notFound`.

Secondary form (same root cause): two independent `From(...)` errors that share a
wrapper message compare equal under `errors.Is`, conflating unrelated errors.

## Why this is FILE, not FIX

This is a judgement call that changes user-visible `errors.Is` semantics that
existing callers and tests depend on. Message-equality is the library's
*deliberate* sentinel-matching mechanism — there is no identity token to match on,
so the only knob is the string. Existing tests that encode the current behaviour:

- `fault_test.go` `TestSentinelCauseIsMatchesSentinel` (line 51)
- `fault_test.go` `TestSentinelWithMessage` (line 71)
- `fault_test.go` `TestErrorWithCauseIsMatchesSameMessage` (line 166) — explicitly
  asserts that two `_ErrorWithCause` with the same message match via `errors.Is`.

Requiring identity (or not falling through on message text) would break those.
A real fix likely needs a distinct sentinel-identity mechanism (e.g. an interned
pointer/ID carried on the sentinel and propagated through `From` / `WithMessage`),
separate from the display message — a design change, not a one-liner. Hence:
file for a human to decide the contract, do not silently change `Is`.

## Repro

`repro_test.go.txt` in this folder is the failing test (saved out of the live
suite). To run: copy it back to the repo root as `is_falsepositive_test.go`, then
`go test ./...` from `/Users/gleb/published/fault`. Both
`TestIsFalsePositiveFromMessageCollision` and
`TestIsFalsePositiveWithMessageCollision` fail while the bug stands.
