// Package graph selects a part of a model graph around one or more packages
// and renders it for visualization.
package graph

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/ftl/go-depend/model"
)

// EdgeKind selects which kind of edge the selection follows and contains.
type EdgeKind string

const (
	// ImportEdges follows the imports between the packages.
	ImportEdges EdgeKind = "imports"
	// ImplementationEdges follows the types that implement an interface of
	// another package. Go satisfies an interface implicitly, therefore these
	// edges exist without any import.
	ImplementationEdges EdgeKind = "implements"
	// AllEdges follows both kinds.
	AllEdges EdgeKind = "both"
)

// EdgeKinds returns all supported kinds of edges.
func EdgeKinds() []EdgeKind {
	return []EdgeKind{ImportEdges, ImplementationEdges, AllEdges}
}

// EdgeKindNames returns the names of all supported kinds, for the help of a
// command line flag.
func EdgeKindNames() []string {
	result := make([]string, 0, len(EdgeKinds()))
	for _, kind := range EdgeKinds() {
		result = append(result, string(kind))
	}
	return result
}

// ParseEdgeKind returns the kind of edge with the given name.
func ParseEdgeKind(value string) (EdgeKind, error) {
	for _, kind := range EdgeKinds() {
		if string(kind) == value {
			return kind, nil
		}
	}
	return "", fmt.Errorf("unknown kind of edge %q, use one of %s", value, strings.Join(EdgeKindNames(), ", "))
}

// followsImports reports whether this kind contains the imports. The empty
// kind is the default of go-depend.
func (k EdgeKind) followsImports() bool {
	return k == "" || k == ImportEdges || k == AllEdges
}

// followsImplementations reports whether this kind contains the
// implementation edges.
func (k EdgeKind) followsImplementations() bool {
	return k == ImplementationEdges || k == AllEdges
}

// Options control which part of the graph is selected.
type Options struct {
	// Incoming includes the packages that import a selected package.
	Incoming bool
	// Outgoing includes the packages that a selected package imports.
	Outgoing bool
	// Depth is the maximum number of steps from a root. The value 0 means
	// that the number of steps is not limited.
	Depth int
	// Edges selects the kind of edge that the selection follows. The empty
	// value means the imports.
	Edges EdgeKind
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
	// importing and then by the imported package. It is empty if the options
	// do not contain the imports.
	Imports []model.Import
	// Implementations are all implementation edges between the selected
	// packages. It is empty if the options do not contain them.
	Implementations []model.Implementation
}

// Select returns the part of the graph around the given roots. It follows the
// imports in the directions of the options, and it stops after the number of
// steps that the options allow. Only packages of the analyzed modules are
// selected, a root is always selected. If the options contain no direction at
// all, Select follows the imports in both directions.
func Select(g *model.Graph, options Options, roots ...string) Selection {
	selected := selectNodes(g, options, roots)

	return Selection{
		Packages:        packagesOf(g, selected),
		Externals:       externalsOf(g, selected),
		Imports:         importsBetween(g, selected, options.Edges),
		Implementations: implementationsBetween(g, selected, options.Edges),
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

		for _, next := range neighboursOf(g, current.importPath, incoming, outgoing, options.Edges) {
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

func neighboursOf(g *model.Graph, importPath string, incoming bool, outgoing bool, edges EdgeKind) []string {
	var result []string
	if edges.followsImports() {
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
	}
	if edges.followsImplementations() {
		if outgoing {
			for _, impl := range g.OutgoingImplementations(importPath) {
				result = append(result, impl.ToPkg)
			}
		}
		if incoming {
			for _, impl := range g.IncomingImplementations(importPath) {
				result = append(result, impl.FromPkg)
			}
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
func importsBetween(g *model.Graph, selected map[string]bool, edges EdgeKind) []model.Import {
	if !edges.followsImports() {
		return nil
	}

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

// implementationsBetween returns all implementation edges between the
// selected packages.
func implementationsBetween(g *model.Graph, selected map[string]bool, edges EdgeKind) []model.Implementation {
	if !edges.followsImplementations() {
		return nil
	}

	var result []model.Implementation
	for _, impl := range g.Implementations() {
		if selected[impl.FromPkg] && selected[impl.ToPkg] {
			result = append(result, impl)
		}
	}
	return result
}
