# GetFullError duplicates every layer's message in its output

## Hypothesis

`GetFullError` renders the same message text multiple times for nested wrapped
errors, because two independent recursive renderings are composed:

1. `_ErrorWithCause.Error()` (error_cause.go:12) ALREADY concatenates the whole
   chain recursively: `fmt.Sprintf("%s: %s", s.msg, s.cause.Error())`. So the
   top-level `.Error()` is already the full `"topmsg: midmsg: DEEPESTTOKEN"`.

2. `GetFullError` (fault.go:70-73) THEN ALSO walks `errors.Unwrap` and appends
   each layer's `.Error()` — and each of those layer `.Error()` calls is itself
   the recursive concatenation of everything below it.

The result is that the deepest messages repeat once per level of nesting.

## Observed vs expected

Chain: `From(From(Errorf("DEEPESTTOKEN"), "midmsg"), "topmsg")`

Observed `GetFullError` output (after the stack trace block):

```
topmsg: midmsg: DEEPESTTOKEN
midmsg: DEEPESTTOKEN
DEEPESTTOKEN
```

- `DEEPESTTOKEN` appears 3 times
- `midmsg` appears 2 times
- `topmsg` appears 1 time

Expected (clean rendering): each token appears exactly once, regardless of which
de-duplicated layout is chosen.

### Duplication factor

For a chain N levels deep, the deepest message appears N times, the next N-1
times, etc. — i.e. O(N^2) total message text. The factor for the deepest token
equals the nesting depth.

## Why I bailed (FILE, not FIX)

The correct output format for `GetFullError` is a product / presentation
judgement call:

- One line per layer with only that layer's own `msg` (no recursion)?
- Just the stack trace plus the single full `err.Error()`?
- A de-duplicated layered rendering?

Any of these changes the user-visible diagnostic string that logging and
diagnostics code may already depend on / pattern-match against. That is not an
unambiguous crash, so per the dig bail criteria this is filed rather than fixed.

## Repro

The failing test is saved alongside this ticket as
`getfullerror_duplication_test.go.txt`. To run it:

1. Copy it into the package root as a `_test.go` file:
   `cp "docs/issues/getfullerror-message-duplication_[PICKUP]/getfullerror_duplication_test.go.txt" getfullerror_duplication_test.go`
2. `go test ./... -run TestGetFullErrorNoMessageDuplication -v`
3. It asserts each unique token appears exactly once; currently FAILS with
   counts 3 and 2.
4. Delete the copied test afterwards.

## Affected code

- `fault.go:58-76` — `GetFullError`
- `error_cause.go:12-14` — `_ErrorWithCause.Error()` (the recursive concatenation)
