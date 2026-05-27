# Coalesce `Errors()` leaks the internal slice (encapsulation break)

## Summary
`(*_CoalesceError).Errors()` returns `s.errs` directly. A caller that mutates an
element of the returned slice corrupts the coalesce error's internal state,
silently changing later `Error()` / `Is` / `As` results.

## Location
`coalesce.go`:
```go
func (s *_CoalesceError) Errors() []error {
	return s.errs
}
```

## Observed vs expected
Repro (in `coalesce_leak_test.go` at time of filing):

```go
ce := Coalesce(errors.New("alpha"), errors.New("beta"))
ce.Error()                       // "alpha; beta"
ce.(*_CoalesceError).Errors()[0] = errors.New("HIJACKED")
ce.Error()                       // "HIJACKED; beta"  <-- internal state mutated
```

- Observed: after mutating the returned slice, `Error()` returns `"HIJACKED; beta"`.
- Expected: `Errors()` exposes a read-only view; mutating it must not affect the
  coalesce error. `Error()` stays `"alpha; beta"`.

## Fix
Return a defensive copy:

```go
func (s *_CoalesceError) Errors() []error {
	out := make([]error, len(s.errs))
	copy(out, s.errs)
	return out
}
```

Behaviour-preserving: no production caller depends on mutating the returned
slice (only read in `coalesce_test.go`). `Unwrap()` shares a sub-slice of the
same backing array, but that view is internal and not handed to callers.

## Repro
`go test ./... -run TestCoalesceErrorsEncapsulationLeak` — RED before fix.
