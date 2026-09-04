package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/ftl/go-depend/graph"
	"github.com/ftl/go-depend/load"
)

// defaultRoot is the package of the current working directory.
const defaultRoot = "."

type graphFlags struct {
	exclude  []string
	incoming bool
	outgoing bool
	depth    int
	output   string
}

func newGraphCmd() *cobra.Command {
	var flags graphFlags

	result := &cobra.Command{
		Use:   "graph [pattern]",
		Short: "Produce a dependency graph for the selected package",
		Long: `Produce a dependency graph for the selected package and write it as a mermaid
graph. The pattern defaults to ., the package of the current working
directory, and it may also address a package outside of the module.

go-depend always analyzes the whole module with the pattern ./..., therefore
run this command in the root directory of the module.`,
		Args: cobra.MaximumNArgs(1),
		// The usage of graph does not help with an error that occurs while
		// the packages are loaded or rendered.
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGraph(cmd, rootOf(args), flags)
		},
	}

	result.Flags().StringArrayVar(&flags.exclude, "exclude", nil,
		"ignore all files with a path that matches this regular expression (repeatable)")
	result.Flags().BoolVar(&flags.incoming, "incoming", false,
		"include the packages that import the selected package")
	result.Flags().BoolVar(&flags.outgoing, "outgoing", false,
		"include the packages that the selected package imports")
	result.Flags().IntVar(&flags.depth, "depth", 1,
		"maximum number of steps from the selected package, 0 for no limit")
	result.Flags().StringVar(&flags.output, "output", "",
		"write the graph to this file instead of stdout")

	return result
}

func rootOf(args []string) string {
	if len(args) == 0 {
		return defaultRoot
	}
	return args[0]
}

func runGraph(cmd *cobra.Command, pattern string, flags graphFlags) error {
	options := load.Options{Exclude: flags.exclude}

	root, err := load.ResolveOne(options, pattern)
	if err != nil {
		return err
	}
	moduleGraph, err := load.Load(options, defaultPattern)
	if err != nil {
		return err
	}

	selection := graph.Select(moduleGraph, graph.Options{
		Incoming: flags.incoming,
		Outgoing: flags.outgoing,
		Depth:    flags.depth,
	}, root)

	return writeGraph(cmd, flags.output, selection)
}

func writeGraph(cmd *cobra.Command, output string, selection graph.Selection) error {
	if output == "" {
		return graph.Mermaid(cmd.OutOrStdout(), selection)
	}

	file, err := os.Create(output)
	if err != nil {
		return err
	}
	if err := graph.Mermaid(file, selection); err != nil {
		file.Close()
		return err
	}

	// The error of Close must not be ignored: it reports a write that did not
	// reach the file.
	return file.Close()
}
