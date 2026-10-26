package fault

import (
	"errors"
	"runtime"
	"strings"
	"testing"
)

func assertStackOrigin(t *testing.T, err error, file string, line int) {
	t.Helper()
	trace := GetStackTrace(err)
	if len(trace) == 0 {
		t.Fatal("expected non-empty stack trace")
	}
	_, gotFile, gotLine := trace[0].location()
	if gotFile != file || gotLine != line {
		t.Errorf("want stack origin %s:%d, got %+v", file, line, trace[0])
	}
}

func TestConstructorsCaptureExactCaller(t *testing.T) {
	_, file, line, _ := runtime.Caller(0)
	err := Errorf("root")
	assertStackOrigin(t, err, file, line+1)

	_, file, line, _ = runtime.Caller(0)
	err = Wrap(errors.New("root"), "context")
	assertStackOrigin(t, err, file, line+1)

	_, file, line, _ = runtime.Caller(0)
	err = Wrapf(errors.New("root"), "context %d", 1)
	assertStackOrigin(t, err, file, line+1)

	_, file, line, _ = runtime.Caller(0)
	err = From(errors.New("root"), "context")
	assertStackOrigin(t, err, file, line+1)

	_, file, line, _ = runtime.Caller(0)
	err = Sentinel("base").From(errors.New("root"), "context")
	assertStackOrigin(t, err, file, line+1)

	_, file, line, _ = runtime.Caller(0)
	err = TagAsRetryable(errors.New("root"), "retry")
	assertStackOrigin(t, err, file, line+1)

	_, file, line, _ = runtime.Caller(0)
	err = MakeRetryableErrorf("retry %d", 1)
	assertStackOrigin(t, err, file, line+1)
}

func recoverAtSite(value any) (err error, file string, line int) {
	defer func() { RecoverPanic(recover(), &err) }()
	_, file, line, _ = runtime.Caller(0)
	panic(value)
}

func recoverRuntimePanic() (err error, file string, line int) {
	defer func() { RecoverPanic(recover(), &err) }()
	var empty []int
	_, file, line, _ = runtime.Caller(0)
	_ = empty[0]
	return
}

func TestRecoverPanicCapturesExactPanicSite(t *testing.T) {
	cause := errors.New("root")
	for _, value := range []any{"root", cause} {
		err, file, line := recoverAtSite(value)
		assertStackOrigin(t, err, file, line+1)
		if got := err.Error(); got != "caught panic: root" {
			t.Errorf("unexpected recovered message %q", got)
		}
		if value == cause && !errors.Is(err, cause) {
			t.Error("recovery lost the panic error's identity")
		}
	}

	err, file, line := recoverRuntimePanic()
	assertStackOrigin(t, err, file, line+1)
	if !strings.Contains(err.Error(), "index out of range") {
		t.Errorf("runtime panic lost its message: %v", err)
	}
}

func TestRecoverPanicDirectCallCapturesCaller(t *testing.T) {
	var err error
	_, file, line, _ := runtime.Caller(0)
	RecoverPanic("root", &err)
	assertStackOrigin(t, err, file, line+1)
}

func TestRecoverPanicWithoutPanicPreservesError(t *testing.T) {
	cause := errors.New("existing")
	err := cause
	RecoverPanic(nil, &err)
	if err != cause {
		t.Errorf("no panic changed the existing error: %v", err)
	}
}

func TestStackTraceSnapshotCannotMutateError(t *testing.T) {
	err := Errorf("root")
	trace := GetStackTrace(err)
	origin := trace[0]
	trace[0] = 0
	if got := GetStackTrace(err)[0]; got != origin {
		t.Errorf("mutating returned stack changed the error origin: %+v", got)
	}
}

//go:noinline
func recordRecoveredPanic(value any, err *error) {
	RecoverPanic(value, err)
}

func recoverThroughHelper() (err error, file string, line int) {
	defer func() { recordRecoveredPanic(recover(), &err) }()
	_, file, line, _ = runtime.Caller(0)
	panic("root")
}

func recoverNilPointerPanic() (err error, file string, line int) {
	defer func() { RecoverPanic(recover(), &err) }()
	var ptr *int
	_, file, line, _ = runtime.Caller(0)
	_ = *ptr
	return
}

func recoverNestedPanic() (err error, file string, line int) {
	defer func() { RecoverPanic(recover(), &err) }()
	defer func() {
		_, file, line, _ = runtime.Caller(0)
		panic("newest")
	}()
	panic("older")
}

func TestRecoverPanicFindsBoundaryThroughHelpers(t *testing.T) {
	err, file, line := recoverThroughHelper()
	assertStackOrigin(t, err, file, line+1)
}

func TestRecoverPanicFindsNilPointerSite(t *testing.T) {
	err, file, line := recoverNilPointerPanic()
	assertStackOrigin(t, err, file, line+1)
}

func TestRecoverPanicUsesInnermostPanic(t *testing.T) {
	err, file, line := recoverNestedPanic()
	assertStackOrigin(t, err, file, line+1)
	if got := err.Error(); got != "caught panic: newest" {
		t.Errorf("recovery lost the newest panic: %q", got)
	}
}
