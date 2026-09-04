// Package graph selects a part of a model graph around one or more packages
// and renders it for visualization.
package graph

import (
	"cmp"
	"slices"

	"github.com/ftl/go-depend/model"
)

// Options control which part of the graph is selected.
type Options struct {
	// Incoming includes the packages that import a selected package.
	Incoming bool
	// Outgoing includes the packages that a selected package imports.
	Outgoing bool
	// Depth is the maximum number of steps from a root. The value 0 means
	// that the number of steps is not limited.
	Depth int
}

// Selection is a part of a model graph.
type Selection struct {
	// Packages are the selected packages of the analyzed modules, sorted by
	// their import path.
	Packages []model.Package
	// Externals are the import paths of the selected packages that do not
	// belong to the analyzed modules. Only a root can be external.
	Externals []string
	// Imports are all imports between the selected packages, sorted by the
	// importing and then by the imported package.
	Imports []model.Import
}

// Select returns the part of the graph around the given roots. It follows the
// imports in the directions of the options, and it stops after the number of
// steps that the options allow. Only packages of the analyzed modules are
// selected, a root is always selected. If the options contain no direction at
// all, Select follows the imports in both directions.
func Select(g *model.Graph, options Options, roots ...string) Selection {
	selected := selectNodes(g, options, roots)

	return Selection{
		Packages:  packagesOf(g, selected),
		Externals: externalsOf(g, selected),
		Imports:   importsBetween(g, selected),
	}
}

// step is one package on the way from a root through the graph.
type step struct {
	importPath string
	depth      int
}

func selectNodes(g *model.Graph, options Options, roots []string) map[string]bool {
	incoming, outgoing := directionsOf(options)

	selected := make(map[string]bool, len(roots))
	queue := make([]step, 0, len(roots))
	for _, root := range roots {
		if selected[root] {
			continue
		}
		selected[root] = true
		queue = append(queue, step{importPath: root})
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if options.Depth > 0 && current.depth >= options.Depth {
			continue
		}

		for _, next := range neighboursOf(g, current.importPath, incoming, outgoing) {
			if selected[next] {
				continue
			}
			if _, ok := g.Package(next); !ok {
				// Only the packages of the analyzed modules are selected.
				continue
			}
			selected[next] = true
			queue = append(queue, step{importPath: next, depth: current.depth + 1})
		}
	}

	return selected
}

// directionsOf returns the directions to follow. No direction at all means
// both directions: a selection without any direction would contain nothing
// but the roots.
func directionsOf(options Options) (incoming bool, outgoing bool) {
	if !options.Incoming && !options.Outgoing {
		return true, true
	}
	return options.Incoming, options.Outgoing
}

func neighboursOf(g *model.Graph, importPath string, incoming bool, outgoing bool) []string {
	var result []string
	if outgoing {
		for _, imp := range g.Outgoing(importPath) {
			result = append(result, imp.To)
		}
	}
	if incoming {
		for _, imp := range g.Incoming(importPath) {
			result = append(result, imp.From)
		}
	}
	return result
}

func packagesOf(g *model.Graph, selected map[string]bool) []model.Package {
	var result []model.Package
	for _, pkg := range g.Packages() {
		if selected[pkg.ImportPath] {
			result = append(result, pkg)
		}
	}
	return result
}

func externalsOf(g *model.Graph, selected map[string]bool) []string {
	var result []string
	for importPath := range selected {
		if _, ok := g.Package(importPath); !ok {
			result = append(result, importPath)
		}
	}
	slices.Sort(result)
	return result
}

// importsBetween returns all imports between the selected packages, also the
// imports that the traversal did not follow. Every import between two
// selected packages is part of the picture.
func importsBetween(g *model.Graph, selected map[string]bool) []model.Import {
	var result []model.Import
	for importPath := range selected {
		for _, imp := range g.Outgoing(importPath) {
			if selected[imp.To] {
				result = append(result, imp)
			}
		}
	}

	slices.SortFunc(result, func(a, b model.Import) int {
		return cmp.Or(cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To))
	})
	return result
}
