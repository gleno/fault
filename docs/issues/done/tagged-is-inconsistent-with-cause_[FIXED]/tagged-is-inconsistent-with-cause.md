# `_ErrorWithTag` inherits `_ErrorWithCause.Is`, breaking same-msg equality for tagged errors

## Hypothesis

`_ErrorWithTag` (tagged_error.go:19) embeds `_ErrorWithCause` and defines **no `Is`
method of its own**. It therefore inherits `_ErrorWithCause.Is` (error_cause.go:20):

```go
func (s *_ErrorWithCause) Is(target error) bool {
	if t, ok := target.(*_ErrorWithCause); ok {
		return s.msg == t.msg
	}
	return s._Error.Is(target)
}
```

The fast-path does `target.(*_ErrorWithCause)`. When the *target* is itself a
`*_ErrorWithTag`, that type assertion **fails** — Go embedding does not make
`*_ErrorWithTag` assertable to `*_ErrorWithCause`. So it falls through to
`_Error.Is`, whose assertion `target.(*_Error)` also fails. Result: `false`.

Consequence: two **distinct** `_ErrorWithCause` values with identical wrapper
message compare equal under `errors.Is` (message-equality semantics), but the
analogous pair of **distinct `_ErrorWithTag` values with identical message and
identical tag** compare *unequal*. The same-msg matching semantic silently
disappears the moment the error is tagged.

NOTE: this is DISTINCT from `is-message-equality-false-positive` (which argues the
message-equality matching is itself wrong / a false-positive). This ticket is about
the *inconsistency*: whatever the intended semantic is, it is not applied uniformly
across the two embedding types. If the resolution of the false-positive ticket is
"remove message-equality matching entirely", this ticket folds into that. If the
resolution keeps message-equality, then `_ErrorWithTag` needs its own `Is` (or the
assertion must cover the tagged type) so tagged errors behave like their cause base.

## Observed vs expected

```go
causeA := From(Sentinel("inner"), "same msg")
causeB := From(Sentinel("inner"), "same msg")
errors.Is(causeA, causeB)                       // OBSERVED: true

taggedA := TagAsUserFault(Sentinel("inner"), "same msg")
taggedB := TagAsUserFault(Sentinel("inner"), "same msg")
errors.Is(taggedA, taggedB)                     // OBSERVED: false   (inconsistent with cause-pair)
```

Expected: the two pairs should agree. Either both `true` (message-equality applies
through the tag) or both `false` (message-equality removed) — but not split by type.

## Why bailed (filed, not fixed)

`Is` semantics are a judgement call with user-visible behavior, and the correct
resolution is coupled to the still-open `is-message-equality-false-positive`
decision. Picking a fix here unilaterally (add `_ErrorWithTag.Is`, or broaden the
assertion) could contradict that ticket's resolution. Per the dig directive
(Is-semantics = lean FILE), filing.

## Possible fixes (for the picker)

- If message-equality is kept: give `_ErrorWithTag` its own `Is` that compares
  `s.msg` (and possibly `s.tag`) against a `*_ErrorWithTag` target, falling back to
  the embedded `_ErrorWithCause.Is`. Decide whether tag must also match.
- If message-equality is dropped (per the other ticket): no change needed here; this
  ticket is subsumed.

## Repro

`repro_test.go.txt` in this folder is the failing test (kept out of the live suite
so `go test ./...` stays green). To run:

```bash
cp "docs/issues/tagged-is-inconsistent-with-cause_[PICKUP]/repro_test.go.txt" tag_is_consistency_test.go
go test ./... -run TestTagIsConsistencyWithCause -v
rm tag_is_consistency_test.go
```

It FAILS while the hypothesis holds (asserts the two `errors.Is` results agree).
