package model_test

import (
	"testing"

	"github.com/ftl/go-depend/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	pathApp   = "github.com/ftl/example"
	pathScan  = "github.com/ftl/example/scan"
	pathAlone = "github.com/ftl/example/alone"
	pathCobra = "github.com/spf13/cobra"
	pathFmt   = "fmt"
)

// testGraph contains three packages of one module: app imports scan, cobra and
// fmt, scan imports fmt, and alone has no imports in either direction.
func testGraph() *model.Graph {
	graph := model.NewGraph()
	graph.AddPackage(model.Package{ImportPath: pathApp, ModulePath: pathApp, Dir: "."})
	graph.AddPackage(model.Package{ImportPath: pathScan, ModulePath: pathApp, Dir: "scan", Types: 3, Funcs: 1, Abstract: 2})
	graph.AddPackage(model.Package{ImportPath: pathAlone, ModulePath: pathApp, Dir: "alone"})

	graph.AddImport(model.Import{From: pathApp, To: pathFmt, Kind: model.Stdlib})
	graph.AddImport(model.Import{From: pathApp, To: pathCobra, Kind: model.External})
	graph.AddImport(model.Import{From: pathApp, To: pathScan, Kind: model.SameModule})
	graph.AddImport(model.Import{From: pathScan, To: pathFmt, Kind: model.Stdlib})

	return graph
}

func TestPackageLookup(t *testing.T) {
	graph := testGraph()

	pkg, ok := graph.Package(pathScan)
	require.True(t, ok)
	assert.Equal(t, "scan", pkg.Dir)
	assert.Equal(t, 3, pkg.Types)

	_, ok = graph.Package(pathCobra)
	assert.False(t, ok, "an imported package outside of the module is no package of the graph")
}

func TestPackagesAreSortedByImportPath(t *testing.T) {
	graph := testGraph()

	packages := graph.Packages()

	require.Len(t, packages, 3)
	assert.Equal(t, []string{pathApp, pathAlone, pathScan}, importPaths(packages))
}

func TestOutgoingImports(t *testing.T) {
	graph := testGraph()

	imports := graph.Outgoing(pathApp)

	require.Len(t, imports, 3)
	assert.Equal(t, []string{pathFmt, pathScan, pathCobra}, targets(imports), "sorted by the imported package")
	assert.Equal(t, model.Stdlib, imports[0].Kind)
	assert.Equal(t, model.SameModule, imports[1].Kind)
}

func TestIncomingImports(t *testing.T) {
	graph := testGraph()

	assert.Equal(t, []string{pathApp}, sources(graph.Incoming(pathScan)))
	assert.Equal(t, []string{pathApp, pathScan}, sources(graph.Incoming(pathFmt)), "also for a package outside of the module")
}

func TestPackageWithoutImports(t *testing.T) {
	graph := testGraph()

	assert.Empty(t, graph.Outgoing(pathAlone))
	assert.Empty(t, graph.Incoming(pathAlone))
}

func TestUnknownPackageHasNoImports(t *testing.T) {
	graph := testGraph()

	assert.Empty(t, graph.Outgoing("does/not/exist"))
	assert.Empty(t, graph.Incoming("does/not/exist"))
}

func TestAddPackageReplacesPackageWithSameImportPath(t *testing.T) {
	graph := testGraph()

	graph.AddPackage(model.Package{ImportPath: pathScan, ModulePath: pathApp, Dir: "scan", Types: 7})

	pkg, ok := graph.Package(pathScan)
	require.True(t, ok)
	assert.Equal(t, 7, pkg.Types)
	assert.Len(t, graph.Packages(), 3)
}

func TestHasViolation(t *testing.T) {
	testCases := []struct {
		name     string
		metrics  model.Metrics
		expected bool
	}{
		{"on the main sequence", model.Metrics{Zone: model.MainSequence}, false},
		{"zone of pain", model.Metrics{Zone: model.ZoneOfPain}, true},
		{"zone of uselessness", model.Metrics{Zone: model.ZoneOfUselessness}, true},
		{"unstable dependency", model.Metrics{UnstableDependencies: []string{pathScan}}, true},
		{"empty list of unstable dependencies", model.Metrics{UnstableDependencies: []string{}}, false},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.metrics.HasViolation())
		})
	}
}

func importPaths(packages []model.Package) []string {
	result := make([]string, len(packages))
	for i, pkg := range packages {
		result[i] = pkg.ImportPath
	}
	return result
}

func targets(imports []model.Import) []string {
	result := make([]string, len(imports))
	for i, imp := range imports {
		result[i] = imp.To
	}
	return result
}

func sources(imports []model.Import) []string {
	result := make([]string, len(imports))
	for i, imp := range imports {
		result[i] = imp.From
	}
	return result
}
