// Package deps pins pre-declared project dependencies via blank imports until
// their real usages land in later work packages; without this go mod tidy prunes them.
package deps

import (
	_ "github.com/charmbracelet/bubbles/list"
	_ "github.com/charmbracelet/bubbletea"
	_ "github.com/charmbracelet/glamour"
	_ "github.com/charmbracelet/lipgloss"
	_ "github.com/oklog/ulid/v2"
	_ "github.com/rogpeppe/go-internal/testscript"
	_ "github.com/spf13/cobra"
	_ "gopkg.in/yaml.v3"
	_ "modernc.org/sqlite"
)
