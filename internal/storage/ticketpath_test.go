package storage

import "testing"

const validULID = "01HXAGA0000000000000000000"

func TestIsItemFilename(t *testing.T) {
	cases := []struct {
		name string
		base string
		want bool
	}{
		{"valid ulid", validULID + ".md", true},
		{"non-item file", "README.md", false},
		{"wrong-length stem", "01HXAGA000.md", false},
		{"dotfile", "." + validULID + ".md", false},
		{"dotfile masquerading as valid length", "." + validULID[1:] + ".md", false},
		{"wrong extension", validULID + ".txt", false},
		{"no extension", validULID, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isItemFilename(c.base); got != c.want {
				t.Errorf("isItemFilename(%q) = %v, want %v", c.base, got, c.want)
			}
		})
	}
}

func TestIsItemPath(t *testing.T) {
	cases := []struct {
		name string
		rel  string
		want bool
	}{
		{"direct child", TicketsPrefix + "/" + validULID + ".md", true},
		{"non-item file", TicketsPrefix + "/README.md", false},
		{"nested path", TicketsPrefix + "/sub/" + validULID + ".md", false},
		{"outside tickets dir", validULID + ".md", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsItemPath(c.rel); got != c.want {
				t.Errorf("IsItemPath(%q) = %v, want %v", c.rel, got, c.want)
			}
		})
	}
}
