// Command kira is the CLI entry point.
package main

import (
	"os"

	"github.com/shivamshivanshu/kira/internal/cli"
)

func main() {
	os.Exit(cli.Main())
}
