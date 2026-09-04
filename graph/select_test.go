package graph_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ftl/go-depend/graph"
	"github.com/ftl/go-depend/model"
)

const (
	module    = "example.com/module"
	pathMain  = module
	pathHub   = module + "/hub"
	pathLeft  = module + "/left"
	pathRight = module + "/right"
	pathDeep  = module + "/deep"
	pathAlone = module + "/alone"
	pathCobra = "github.com/spf13/cobra"
	pathFmt   = "fmt"
)

// testGraph contains one module:
//
//	main  -> hub
//	hub   -> left, right
//	left  -> deep, cobra
//	right -> deep, fmt
//	deep  -> cobra
//	alone                    (no imports in either direction)
func testGraph() *model.Graph {
	g := model.NewGraph()
	for _, importPath := range []string{pathMain, pathHub, pathLeft, pathRight, pathDeep, pathAlone} {
		g.AddPackage(model.Package{ImportPath: importPath, ModulePath: module})
	}

	addImport(g, pathMain, pathHub, model.SameModule)
	addImport(g, pathHub, pathLeft, model.SameModule)
	addImport(g, pathHub, pathRight, model.SameModule)
	addImport(g, pathLeft, pathDeep, model.SameModule)
	addImport(g, pathRight, pathDeep, model.SameModule)
	addImport(g, pathLeft, pathCobra, model.External)
	addImport(g, pathDeep, pathCobra, model.External)
	addImport(g, pathRight, pathFmt, model.Stdlib)

	return g
}

func addImport(g *model.Graph, from string, to string, kind model.ImportKind) {
	g.AddImport(model.Import{From: from, To: to, Kind: kind})
}

func TestSelectOutgoingUnlimited(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Outgoing: true}, pathMain)

	assert.Equal(t, []string{pathMain, pathDeep, pathHub, pathLeft, pathRight}, importPaths(selection))
	assert.Empty(t, selection.Externals, "packages outside of the module are not selected")
}

func TestSelectOutgoingWithDepth(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Outgoing: true, Depth: 2}, pathMain)

	assert.Equal(t, []string{pathMain, pathHub, pathLeft, pathRight}, importPaths(selection),
		"deep needs three steps from main")
}

func TestSelectIncoming(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Incoming: true}, pathDeep)

	assert.Equal(t, []string{pathMain, pathDeep, pathHub, pathLeft, pathRight}, importPaths(selection))
}

func TestSelectIncomingWithDepth(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Incoming: true, Depth: 1}, pathDeep)

	assert.Equal(t, []string{pathDeep, pathLeft, pathRight}, importPaths(selection))
}

func TestSelectBothDirections(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Incoming: true, Outgoing: true, Depth: 1}, pathHub)

	assert.Equal(t, []string{pathMain, pathHub, pathLeft, pathRight}, importPaths(selection))
}

func TestSelectWithoutDirectionFollowsBoth(t *testing.T) {
	withoutDirection := graph.Select(testGraph(), graph.Options{Depth: 1}, pathHub)
	bothDirections := graph.Select(testGraph(), graph.Options{Incoming: true, Outgoing: true, Depth: 1}, pathHub)

	assert.Equal(t, importPaths(bothDirections), importPaths(withoutDirection))
}

func TestSelectContainsAllImportsBetweenTheSelectedPackages(t *testing.T) {
	// The import of deep by right is part of the picture, although the
	// traversal reaches deep through left.
	selection := graph.Select(testGraph(), graph.Options{Outgoing: true}, pathMain)

	assert.Equal(t, []model.Import{
		{From: pathMain, To: pathHub, Kind: model.SameModule},
		{From: pathHub, To: pathLeft, Kind: model.SameModule},
		{From: pathHub, To: pathRight, Kind: model.SameModule},
		{From: pathLeft, To: pathDeep, Kind: model.SameModule},
		{From: pathRight, To: pathDeep, Kind: model.SameModule},
	}, selection.Imports)
}

func TestSelectExternalRoot(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Incoming: true, Depth: 1}, pathCobra)

	assert.Equal(t, []string{pathDeep, pathLeft}, importPathsOfPackages(selection.Packages))
	assert.Equal(t, []string{pathCobra}, selection.Externals, "the root stays in the selection")
	assert.Equal(t, []model.Import{
		{From: pathDeep, To: pathCobra, Kind: model.External},
		{From: pathLeft, To: pathDeep, Kind: model.SameModule},
		{From: pathLeft, To: pathCobra, Kind: model.External},
	}, selection.Imports)
}

func TestSelectExternalRootUnlimited(t *testing.T) {
	// This is the third focus of the README: every package of the module that
	// depends on an external package, directly or through other packages.
	selection := graph.Select(testGraph(), graph.Options{Incoming: true}, pathCobra)

	assert.Equal(t, []string{pathMain, pathDeep, pathHub, pathLeft, pathRight},
		importPathsOfPackages(selection.Packages))
	assert.Equal(t, []string{pathCobra}, selection.Externals)
}

func TestSelectExternalRootWithTransitiveIncomingImports(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Incoming: true, Depth: 0}, pathFmt)

	assert.Equal(t, []string{pathMain, pathHub, pathRight}, importPathsOfPackages(selection.Packages),
		"fmt is imported by right, which is imported by hub and main")
}

func TestSelectIsolatedPackage(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Incoming: true, Outgoing: true}, pathAlone)

	assert.Equal(t, []string{pathAlone}, importPaths(selection))
	assert.Empty(t, selection.Imports)
}

func TestSelectSeveralRoots(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Outgoing: true, Depth: 1}, pathLeft, pathRight)

	assert.Equal(t, []string{pathDeep, pathLeft, pathRight}, importPaths(selection))
}

func TestSelectUnknownRoot(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Incoming: true, Outgoing: true}, "example.com/other/pkg")

	assert.Empty(t, selection.Packages)
	assert.Equal(t, []string{"example.com/other/pkg"}, selection.Externals)
	assert.Empty(t, selection.Imports)
}

func TestSelectWithoutRoots(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Incoming: true, Outgoing: true})

	assert.Empty(t, selection.Packages)
	assert.Empty(t, selection.Externals)
	assert.Empty(t, selection.Imports)
}

func TestSelectWithACycle(t *testing.T) {
	// Go forbids a cycle between packages, but the model does not: a graph
	// with a cycle must not make the traversal run forever.
	g := model.NewGraph()
	paths := []string{module + "/a", module + "/b", module + "/c"}
	for _, importPath := range paths {
		g.AddPackage(model.Package{ImportPath: importPath, ModulePath: module})
	}
	addImport(g, paths[0], paths[1], model.SameModule)
	addImport(g, paths[1], paths[2], model.SameModule)
	addImport(g, paths[2], paths[0], model.SameModule)

	selection := graph.Select(g, graph.Options{Outgoing: true}, paths[0])

	assert.Equal(t, paths, importPaths(selection))
	assert.Len(t, selection.Imports, 3, "the import that closes the cycle is part of the picture")
}

func TestSelectWithASelfImport(t *testing.T) {
	// A package that imports itself is impossible in Go, but the traversal
	// must not run forever if the model contains one.
	g := model.NewGraph()
	g.AddPackage(model.Package{ImportPath: pathHub, ModulePath: module})
	addImport(g, pathHub, pathHub, model.SameModule)

	selection := graph.Select(g, graph.Options{Incoming: true, Outgoing: true}, pathHub)

	assert.Equal(t, []string{pathHub}, importPaths(selection))
	assert.Equal(t, []model.Import{{From: pathHub, To: pathHub, Kind: model.SameModule}}, selection.Imports)
}

// importPaths returns the import paths of all selected packages, the external
// ones first.
func importPaths(selection graph.Selection) []string {
	return append(selection.Externals, importPathsOfPackages(selection.Packages)...)
}

func importPathsOfPackages(packages []model.Package) []string {
	result := make([]string, len(packages))
	for i, pkg := range packages {
		result[i] = pkg.ImportPath
	}
	return result
}
