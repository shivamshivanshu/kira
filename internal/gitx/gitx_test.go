package gitx

import "testing"

// A malformed cat-file --batch stream whose last-declared object's content
// runs exactly to the end of the buffer, with the mandatory trailing LF
// missing, used to advance pos one byte past len(buf). The next iteration's
// bytes.IndexByte(buf[pos:], '\n') then panicked instead of erroring.
func TestParseCatFileBatchTruncatedContentDoesNotPanic(t *testing.T) {
	buf := []byte("aaaa blob 5\nhello")
	_, err := parseCatFileBatch(buf, 2)
	if err == nil {
		t.Fatalf("want error for truncated content, got nil")
	}
}
