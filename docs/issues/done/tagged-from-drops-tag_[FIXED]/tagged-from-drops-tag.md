# `.From()` on a tagged fault silently drops the tag

## Hypothesis

`_ErrorWithTag.From(err, message)` (tagged_error.go:46) returns a plain
`&_ErrorWithCause{_Error: s._Error, cause: Wrap(err, message)}`. The new error's
chain is `_ErrorWithCause -> Wrap(err) -> err`; the original tagged `s` is **not**
in the chain, and no `tag` field is carried over. So `extractTag` (which walks the
chain via `errors.As` looking for `*_ErrorWithTag`) finds nothing and falls back to
`InternalFault`.

Result: calling `.From(...)` on a `RetryableFault` (or any tagged) fault produces an
error that `IsRetryable`/`IsUserFault`/etc. report as `InternalFault`.

## Observed vs expected

- Observed: `TagAsRetryable(...).From(downstream, "msg")` → `extractTag == 500` (InternalFault). Tag lost.
- Expected (intuitive): the derived error should retain the parent's classification, i.e. stay retryable, so retry/HTTP-status logic downstream still works.

Compare with `_ErrorWithCause.From` (error_cause.go:27) and `_Error.From` (error.go:40):
both *also* return a plain `_ErrorWithCause` and discard `s` from the chain — so the
same loss applies to chaining `.From()` after any builder. The tagged case is the
sharpest because the lost information is the tag itself.

Note the existing test `TestTaggedErrorFrom` (tagged_error_test.go:128) only asserts
the *message* contains the wrapper/cause strings; it never checks tag survival, which
is why this slipped through.

## Why filed, not fixed

This is a product/design judgement call, not an unambiguous crash:

1. **Judgement:** "Should `From` carry the parent's tag forward, or is `From` meant
   to mint a fresh, untagged error derived from a new cause?" Both are defensible
   designs. `From` semantically replaces the cause, so an argument exists that the old
   tag shouldn't ride along.
2. **User-visible contract change:** any fix flips `IsRetryable`/`IsUserFault`/
   `IsAuthFault`/`IsNotFoundFault` results, which callers use for retry and HTTP-status
   decisions. Changing classification output is a behaviour-contract change.

Per the dig bail criteria (judgement call + changed user-visible behaviour → file),
this is filed for a human decision rather than auto-fixed.

## Decision needed

- Should `.From()` preserve the receiver's tag? If yes, fix is to make
  `_ErrorWithTag.From` keep `s` in the chain (e.g. wrap so the tagged node is still
  reachable by `errors.As`), or re-apply the tag to the result.
- If "From mints a fresh untagged error" is intended, document it explicitly and add a
  regression test asserting the tag is *not* preserved, so the behaviour is locked.

## Repro

Save the test below as a live `*_test.go` in package `fault` and run
`go test ./... -run TestTaggedErrorFromPreservesTag -v`. It currently FAILS
(`extractTag=500`).

See `repro_test.go.txt` in this folder.
