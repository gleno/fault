# _CoalesceError.From discards members from errors.Is/As traversal

## Hypothesis
`(*_CoalesceError).From(err, message)` flattens the coalesced member errors into a
plain string (`s.Error()`) and wraps the *new* `err` with it:

```go
func (s *_CoalesceError) From(err error, message string) Fault {
	return &_ErrorWithCause{_Error: _Error{msg: message}, cause: Wrap(err, s.Error())}
}
```

Because the members become a message string and only the passed-in `err` survives as a
real error value, `errors.Is` / `errors.As` against the result can no longer find any of
the original coalesced members.

## Observed vs expected
Observed: after `coalesced.From(rootCause, "batch context")`, `errors.Is(derived, member)`
returns false for every original member.

Expected (the open question): `WithMessage` on the same type *does* preserve member
traversal (`cause: s`), and there is an explicit test for that
(`TestCoalesceWithMessage`, lines 217-222 of coalesce_test.go). One might reasonably
expect `From` to preserve members the same way. But `From` semantics differ — it
introduces a brand-new root cause `err` and uses the receiver only as contextual text;
this mirrors `(*_Error).From` (error.go:40) which also turns the receiver into a wrapper,
not a traversable cause set.

## Why I bailed (did not fix)
Judgement call + library-contract / user-visible behaviour:
- Whether `From` should keep the coalesced members `errors.Is`-discoverable is a design
  decision, not an unambiguous bug. `From`'s whole point may be to re-root the error on a
  new cause and demote the old members to a message.
- Any fix changes user-visible `errors.Is`/`errors.As` behaviour someone could depend on.
- The failing test below may encode a wrong assumption (that `From` behaves like
  `WithMessage`).

## Repro
`repro_test.go.txt` in this folder reproduces it (kept as `.txt` so the bracketed dir
name does not break `go test ./...`). To run, copy it into the package root and:

```bash
cp "docs/issues/coalesce-from-discards-members_[PICKUP]/repro_test.go.txt" ./dig_repro_test.go
go test -run TestReproCoalesceFromPreservesMembers ./...
rm ./dig_repro_test.go
```

It fails (RED) on current `main`: both `errors.Is` assertions fail.

## Resolution options for the picker
1. Decide `From` *should* re-root and discard members → this is "working as intended";
   close as NOREPRO/WONTFIX and optionally document the difference vs `WithMessage`.
2. Decide `From` *should* preserve members → make the result also wrap `s` as a cause
   alongside the new `err` (a multi-cause wrapper), or change `From` to keep `s` in the
   Unwrap chain. This is a contract change and needs sign-off.
