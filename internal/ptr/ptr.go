// Package ptr holds small pointer helpers shared across packages.
package ptr

func Deref[T any](p *T) T {
	if p == nil {
		var zero T
		return zero
	}
	return *p
}
