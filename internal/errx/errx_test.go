package errx_test

import (
	"errors"
	"testing"

	"github.com/shivamshivanshu/kira/internal/errx"
)

func TestInvalidExtractsFirstHint(t *testing.T) {
	plain := errors.New("plain error")
	hinted := errx.User("bad value").WithHint("try again with a valid value")
	got := errx.Invalid("invalid item", []error{plain, hinted})
	if got.Hint != "try again with a valid value" {
		t.Errorf("Hint = %q, want %q", got.Hint, "try again with a valid value")
	}
	if got.Code != errx.ExitUser {
		t.Errorf("Code = %v, want ExitUser", got.Code)
	}
	if want := "invalid item: plain error; bad value"; got.Error() != want {
		t.Errorf("Error() = %q, want %q", got.Error(), want)
	}
}

func TestInvalidNoHintWhenNoneFound(t *testing.T) {
	got := errx.Invalid("invalid item", []error{errors.New("a"), errors.New("b")})
	if got.Hint != "" {
		t.Errorf("Hint = %q, want empty", got.Hint)
	}
}

func TestWithHintDoesNotMutateReceiver(t *testing.T) {
	base := errx.User("something went wrong")
	hinted := base.WithHint("do X")
	if base.Hint != "" {
		t.Errorf("base.Hint = %q, want unchanged empty", base.Hint)
	}
	if hinted.Hint != "do X" {
		t.Errorf("hinted.Hint = %q, want %q", hinted.Hint, "do X")
	}
	if hinted.Code != base.Code || hinted.Err != base.Err {
		t.Error("WithHint must copy Code/Err from the receiver")
	}
}

func TestWithHintCalledTwiceIsIndependent(t *testing.T) {
	base := errx.User("oops")
	a := base.WithHint("hint a")
	b := base.WithHint("hint b")
	if a.Hint != "hint a" || b.Hint != "hint b" {
		t.Errorf("a.Hint=%q b.Hint=%q, want %q / %q", a.Hint, b.Hint, "hint a", "hint b")
	}
}

func TestUnwrapTraversal(t *testing.T) {
	sentinel := errors.New("sentinel")
	wrapped := errx.User("wrapping: %w", sentinel)
	if !errors.Is(wrapped, sentinel) {
		t.Fatal("errors.Is must traverse through *errx.Error to the wrapped sentinel")
	}
	var target *errx.Error
	if !errors.As(wrapped, &target) {
		t.Fatal("errors.As must recover the *errx.Error itself")
	}
	if target.Code != errx.ExitUser {
		t.Errorf("recovered Code = %v, want ExitUser", target.Code)
	}
}

func TestJoinErrors(t *testing.T) {
	got := errx.JoinErrors("invalid items", []error{errors.New("a: bad"), errors.New("b: bad")})
	if want := "invalid items: a: bad; b: bad"; got != want {
		t.Errorf("JoinErrors() = %q, want %q", got, want)
	}
}
