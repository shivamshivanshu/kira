package core

import (
	"errors"
	"testing"

	"github.com/shivamshivanshu/kira/internal/errx"
	"github.com/shivamshivanshu/kira/internal/query"
)

func TestQueryErrorAdaptsQueryError(t *testing.T) {
	err := queryError(&query.Error{Pos: 5, Msg: "unexpected token"})
	var ce *errx.Error
	if !errors.As(err, &ce) {
		t.Fatalf("queryError(*query.Error) = %v, want an *errx.Error", err)
	}
	if ce.Code != errx.ExitUser {
		t.Errorf("Code = %v, want ExitUser", ce.Code)
	}
	if ce.Hint == "" {
		t.Error("expected a hint")
	}
	if _, ok := errors.AsType[*query.Error](err); !ok {
		t.Error("the wrapped error chain must still reach the original *query.Error")
	}
}

func TestQueryErrorPassesThroughOtherErrors(t *testing.T) {
	orig := errx.User("unknown filter %q", "bogus").WithHint("did you mean `x`?")
	got := queryError(orig)
	if !errors.Is(got, orig) {
		t.Errorf("queryError must return non-*query.Error errors unchanged, got %v", got)
	}
}
