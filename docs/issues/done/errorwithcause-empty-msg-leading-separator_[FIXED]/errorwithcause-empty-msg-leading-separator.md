# `_ErrorWithCause.Error()` emits malformed separators with an empty message

## Hypothesis

`_ErrorWithCause.Error()` (`error_cause.go:13`) unconditionally formats as:

```go
return fmt.Sprintf("%s: %s", s.msg, s.cause.Error())
```

When `msg` is empty (e.g. `From(cause, "")`, `Wrap(cause, "")`, `Sentinel(x).From(cause, "")`),
the `"%s: %s"` template still emits the `": "` separator, producing malformed output with a
leading or doubled separator.

## Observed vs expected

Repro (see `errorwithcause-empty-msg-leading-separator_test.go.txt` in this folder):

- `From(Errorf("root cause"), "").Error()`
  - observed: `": root cause"` (leading separator, dangling prefix)
  - expected: `"root cause"` (no empty prefix segment)

- `Sentinel("x").From(Errorf("root cause"), "").Error()`
  - observed: `"x: : root cause"` (doubled separator)
  - expected: `"x: root cause"`

## Why bailed (not fixed)

Per the dig directive: output-string FORMAT is a presentation judgement call. Changing the
`.Error()` string is user-visible and logging/parsing may depend on the current shape. The fix
is also not unambiguous — when `msg == ""` should we:

1. return just `s.cause.Error()` (drop the empty prefix + separator), or
2. preserve the prefix slot some other way?

Option 1 is the obvious intent, but it changes a stable string. A second consideration: should
`From`/`Wrap` reject or normalize an empty message at construction time instead of patching the
formatter? That's a broader API decision (the empty message may itself be a caller bug worth
surfacing). Leaning FILE so a human picks the contract.

Note: the same class also lives in `coalesce.go` `From` (`Wrap(err, s.Error())`) and anywhere an
empty `msg` reaches `_ErrorWithCause` — fix the class at the formatter, not just the one call site.

## Repro

```
go test ./... -run EmptyMessage -v
```

Rename `errorwithcause-empty-msg-leading-separator_test.go.txt` to `..._test.go` and move it into
the package root (alongside `error_cause.go`) to run it — it is parked as `.txt` here so the
bracketed ticket folder does not break `go test ./...` (the `[` in the `[PICKUP]` tag is an invalid
Go import-path char). It currently fails on both the leading-separator and doubled-separator
assertions.
