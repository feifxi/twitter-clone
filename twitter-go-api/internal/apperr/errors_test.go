package apperr

import (
	"errors"
	"testing"
)

func TestKindConstructors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		err      *Error
		wantKind Kind
	}{
		{"BadRequest", BadRequest("msg"), KindBadRequest},
		{"Unauthorized", Unauthorized("msg"), KindUnauthorized},
		{"Forbidden", Forbidden("msg"), KindForbidden},
		{"NotFound", NotFound("msg"), KindNotFound},
		{"Conflict", Conflict("msg"), KindConflict},
		{"Internal", Internal("msg", nil), KindInternal},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.err.Kind != tc.wantKind {
				t.Fatalf("expected kind %q, got %q", tc.wantKind, tc.err.Kind)
			}
		})
	}
}

func TestKindOf(t *testing.T) {
	t.Parallel()

	err := NotFound("thing not found")
	kind, ok := KindOf(err)
	if !ok {
		t.Fatal("expected KindOf to return true for *Error")
	}
	if kind != KindNotFound {
		t.Fatalf("expected NOT_FOUND, got %q", kind)
	}
}

func TestKindOf_NonAppError(t *testing.T) {
	t.Parallel()

	_, ok := KindOf(errors.New("plain error"))
	if ok {
		t.Fatal("expected KindOf to return false for non-app error")
	}
}

func TestMessageOf(t *testing.T) {
	t.Parallel()

	err := BadRequest("custom message")
	if MessageOf(err) != "custom message" {
		t.Fatalf("unexpected message: %q", MessageOf(err))
	}
}

func TestMessageOf_NonAppError(t *testing.T) {
	t.Parallel()

	if MessageOf(errors.New("raw")) != "" {
		t.Fatal("expected empty string for non-app error")
	}
}

func TestInternalUsesErrAsRoot(t *testing.T) {
	t.Parallel()

	inner := errors.New("db error")
	appErr := Internal("something went wrong", inner)
	if !errors.Is(appErr, inner) {
		t.Fatal("expected Internal error to wrap provided err")
	}
}

func TestInternalCreatesErrWhenNil(t *testing.T) {
	t.Parallel()

	appErr := Internal("something went wrong", nil)
	if appErr.Err == nil {
		t.Fatal("expected Internal to create a fallback error when nil provided")
	}
}

func TestErrorString(t *testing.T) {
	t.Parallel()

	appErr := BadRequest("bad input")
	if appErr.Error() != "bad input" {
		t.Fatalf("unexpected Error() string: %q", appErr.Error())
	}
}

func TestErrorUnwrap(t *testing.T) {
	t.Parallel()

	inner := errors.New("cause")
	appErr := &Error{Kind: KindInternal, Err: inner}
	if !errors.Is(appErr, inner) {
		t.Fatal("expected Unwrap to return inner error")
	}
}
