// Command go-depend calculates software package metrics for Go packages.
package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/ftl/go-depend/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "go-depend:", err)
		os.Exit(exitCode(err))
	}
}

// exitCode maps the result of the command to the exit code of the process: 0
// if everything is fine, 1 if a package violates an invariant, and 2 if
// go-depend cannot do its work.
func exitCode(err error) int {
	switch {
	case err == nil:
		return 0
	case errors.Is(err, cmd.ErrViolations):
		return 1
	default:
		return 2
	}
}
