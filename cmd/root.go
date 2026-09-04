// Package cmd implements the command line interface of go-depend.
package cmd

import (
	"github.com/spf13/cobra"
)

const longDescription = `go-depend calculates software package metrics for Go packages:
the number of outgoing dependencies, the instability (I), the abstractness (A),
and the distance from the main sequence (D = | A + I - 1 |).

It also checks the packages of a module for violated invariants and produces
dependency graphs for visualization with mermaid.`

// Execute runs go-depend. The returned error is already fully described and
// only needs to be reported to the user.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	result := &cobra.Command{
		Use:   "go-depend",
		Short: "Calculate software package metrics for Go packages",
		Long:  longDescription,
		// Errors are reported by the caller of Execute, together with the
		// exit code that belongs to them.
		SilenceErrors: true,
	}
	result.AddCommand(newScanCmd())
	result.AddCommand(newGraphCmd())
	return result
}
