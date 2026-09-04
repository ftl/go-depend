package graph

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/ftl/go-depend/model"
)

// mermaidDirection lays the graph out from left to right: the imports of a
// package are to its right.
const mermaidDirection = "LR"

// Mermaid writes the selection as a mermaid graph. The packages of a module
// are grouped in a subgraph if the selection covers more than one module. An
// import that leaves the module of the importing package is dotted.
func Mermaid(w io.Writer, selection Selection) error {
	ids := idsOf(selection)

	var out strings.Builder
	fmt.Fprintf(&out, "graph %s\n", mermaidDirection)
	writeNodes(&out, selection, ids)
	writeImports(&out, selection, ids)

	_, err := io.WriteString(w, out.String())
	return err
}

// idsOf assigns a short and unique node id to every selected package. The ids
// are assigned in the order of the report, so the same selection always
// results in the same graph.
func idsOf(selection Selection) map[string]string {
	result := make(map[string]string, len(selection.Packages)+len(selection.Externals))
	for _, pkg := range sortedByModule(selection.Packages) {
		result[pkg.ImportPath] = fmt.Sprintf("p%d", len(result))
	}
	for _, importPath := range selection.Externals {
		result[importPath] = fmt.Sprintf("p%d", len(result))
	}
	return result
}

func writeNodes(out *strings.Builder, selection Selection, ids map[string]string) {
	packages := sortedByModule(selection.Packages)
	withModules := countModules(packages) > 1

	currentModule := ""
	for _, pkg := range packages {
		if withModules && pkg.ModulePath != currentModule {
			closeSubgraph(out, currentModule)
			currentModule = pkg.ModulePath
			fmt.Fprintf(out, "  subgraph %s[\"%s\"]\n", ids[pkg.ImportPath]+"m", currentModule)
		}
		fmt.Fprintf(out, "%s%s[\"%s\"]\n", indent(withModules), ids[pkg.ImportPath], pkg.RelativePath())
	}
	closeSubgraph(out, currentModule)

	// A package outside of the analyzed modules belongs to no subgraph, and
	// its full import path is its label.
	for _, importPath := range selection.Externals {
		fmt.Fprintf(out, "  %s[\"%s\"]\n", ids[importPath], importPath)
	}
}

func closeSubgraph(out *strings.Builder, currentModule string) {
	if currentModule != "" {
		out.WriteString("  end\n")
	}
}

func indent(withModules bool) string {
	if withModules {
		return "    "
	}
	return "  "
}

func writeImports(out *strings.Builder, selection Selection, ids map[string]string) {
	for _, imp := range selection.Imports {
		fmt.Fprintf(out, "  %s %s %s\n", ids[imp.From], arrowOf(imp.Kind), ids[imp.To])
	}
}

// arrowOf returns a solid arrow for an import inside the module, and a dotted
// arrow for an import that leaves it.
func arrowOf(kind model.ImportKind) string {
	if kind == model.SameModule {
		return "-->"
	}
	return "-.->"
}

func sortedByModule(packages []model.Package) []model.Package {
	result := slices.Clone(packages)
	slices.SortFunc(result, func(a, b model.Package) int {
		return cmp.Or(
			cmp.Compare(a.ModulePath, b.ModulePath),
			cmp.Compare(a.ImportPath, b.ImportPath),
		)
	})
	return result
}

func countModules(packages []model.Package) int {
	var modules []string
	for _, pkg := range packages {
		if !slices.Contains(modules, pkg.ModulePath) {
			modules = append(modules, pkg.ModulePath)
		}
	}
	return len(modules)
}
