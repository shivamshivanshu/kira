package contract

import "testing"

func TestPlainContract(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{"list", []string{"list", "--no-color"}},
		{"board", []string{"board", "--plain", "--no-color"}},
		{"tree", []string{"tree", "--no-color"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			dir := seededRepo(t)
			out, stderr, code := kira(t, dir, c.args...)
			if code != 0 {
				t.Fatalf("exit %d, stderr: %s", code, stderr)
			}
			got := scrub(out, dir)
			checkGolden(t, "plain/"+c.name+".plain", got)
			out2, _, _ := kira(t, dir, c.args...)
			if got != scrub(out2, dir) {
				t.Errorf("%s not stable across runs:\n%s\n---\n%s", c.name, out, out2)
			}
		})
	}
}
