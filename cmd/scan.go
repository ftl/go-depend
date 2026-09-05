package cmd

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ftl/go-depend/history"
	"github.com/ftl/go-depend/load"
	"github.com/ftl/go-depend/metrics"
	"github.com/ftl/go-depend/model"
	"github.com/ftl/go-depend/report"
)

// defaultPattern covers the whole module.
const defaultPattern = "./..."

// ErrViolations indicates that at least one package violates an invariant and
// that the user asked for it with --fail-on-violation.
var ErrViolations = errors.New("at least one package violates an invariant")

type scanFlags struct {
	exclude         []string
	maxDistance     float64
	format          string
	failOnViolation bool
	edge            bool
	history         bool
	since           string
}

func newScanCmd() *cobra.Command {
	var flags scanFlags

	result := &cobra.Command{
		Use:   "scan [pattern ...]",
		Short: "Calculate the metrics of the matching packages",
		Long: `Scan the Go packages that match the given patterns and write a table with
their metrics and the violated invariants. The patterns default to ./..., the
whole module.`,
		// The usage of scan does not help with an error that occurs while
		// the packages are loaded or rendered.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runScan(cmd, patternsOf(args), flags)
		},
	}

	result.Flags().StringArrayVar(&flags.exclude, "exclude", nil,
		"ignore all files with a path that matches this regular expression (repeatable)")
	result.Flags().Float64Var(&flags.maxDistance, "max-distance", 0.5,
		"maximum allowed distance from the main sequence")
	result.Flags().StringVar(&flags.format, "format", string(report.TableFormat),
		fmt.Sprintf("output format: %s", strings.Join(report.FormatNames(), "|")))
	result.Flags().BoolVar(&flags.failOnViolation, "fail-on-violation", false,
		"exit with the code 1 if a package violates an invariant")
	result.Flags().BoolVar(&flags.edge, "edge", false,
		"use the abstract coupling instead of the abstractness for the distance and the zone")
	result.Flags().BoolVar(&flags.history, "history", false,
		"read the git history and compare the perceived with the calculated instability")
	result.Flags().StringVar(&flags.since, "since", "",
		"read only the git history after this point in time, for example 2026-01-01 (only with --history)")

	return result
}

func patternsOf(args []string) []string {
	if len(args) == 0 {
		return []string{defaultPattern}
	}
	return args
}

func runScan(cmd *cobra.Command, patterns []string, flags scanFlags) error {
	graph, err := load.Load(load.Options{Exclude: flags.exclude}, patterns...)
	if err != nil {
		return err
	}

	all := metrics.Of(graph, metrics.Options{
		MaxDistance:         flags.maxDistance,
		UseAbstractCoupling: flags.edge,
	})
	if flags.history {
		if err := addHistory(all, graph, flags.since); err != nil {
			return err
		}
	}

	if err := report.Write(cmd.OutOrStdout(), report.Format(flags.format), all); err != nil {
		return err
	}

	if flags.failOnViolation && hasViolation(all) {
		return ErrViolations
	}
	return nil
}

// addHistory reads the git history of the module and adds the perceived
// instability to the metrics.
func addHistory(all []model.Metrics, graph *model.Graph, since string) error {
	changes, err := history.Read(history.Options{Since: since})
	if err != nil {
		return err
	}

	perceived := history.PerceivedInstability(changes, graph.Packages())
	for i := range all {
		if value, ok := perceived[all[i].Package.ImportPath]; ok {
			all[i].PerceivedInstability = &value
		}
	}
	return nil
}

func hasViolation(all []model.Metrics) bool {
	return slices.ContainsFunc(all, model.Metrics.HasViolation)
}
