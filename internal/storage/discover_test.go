package storage

import (
	"errors"
	"testing"
)

func TestDiscoverReportsStoreNotFoundSentinel(t *testing.T) {
	_, err := Discover(t.TempDir())
	if err == nil {
		t.Fatal("Discover in a dir without .kira should error")
	}
	if !errors.Is(err, ErrStoreNotFound) {
		t.Fatalf("error should match ErrStoreNotFound sentinel, got %v", err)
	}
}
