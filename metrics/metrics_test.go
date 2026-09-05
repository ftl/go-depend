package metrics_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ftl/go-depend/metrics"
	"github.com/ftl/go-depend/model"
)

const (
	module    = "example.com/module"
	pathMain  = module
	pathHub   = module + "/hub"
	pathLeaf  = module + "/leaf"
	pathAlone = module + "/alone"
	pathUtil  = module + "/util"

	// maxDistance is the default threshold of go-depend.
	maxDistance = 0.5
)

// testGraph contains one module with five packages:
//
//	main  -> hub                    (the entry point, nothing imports it)
//	hub   -> leaf, util             (imported by main)
//	leaf                            (imported by hub, imports nothing)
//	util  -> fmt, cobra, sibling    (imported by hub, no imports inside the module)
//	alone                           (no imports in either direction)
func testGraph() *model.Graph {
	graph := model.NewGraph()
	graph.AddPackage(model.Package{ImportPath: pathMain, ModulePath: module})
	graph.AddPackage(model.Package{ImportPath: pathHub, ModulePath: module, Types: 2, Funcs: 2, Abstract: 1})
	graph.AddPackage(model.Package{ImportPath: pathLeaf, ModulePath: module, Types: 4, Funcs: 2, Abstract: 3})
	graph.AddPackage(model.Package{ImportPath: pathUtil, ModulePath: module, Types: 0, Funcs: 4, Abstract: 0})
	graph.AddPackage(model.Package{ImportPath: pathAlone, ModulePath: module, Types: 1, Funcs: 0, Abstract: 1})

	graph.AddImport(model.Import{From: pathMain, To: pathHub, Kind: model.SameModule})
	graph.AddImport(model.Import{From: pathHub, To: pathLeaf, Kind: model.SameModule})
	graph.AddImport(model.Import{From: pathHub, To: pathUtil, Kind: model.SameModule})
	graph.AddImport(model.Import{From: pathUtil, To: "fmt", Kind: model.Stdlib})
	graph.AddImport(model.Import{From: pathUtil, To: "github.com/spf13/cobra", Kind: model.External})
	graph.AddImport(model.Import{From: pathUtil, To: "example.com/sibling/lib", Kind: model.WorkspaceSibling})

	return graph
}

func TestCoupling(t *testing.T) {
	testCases := []struct {
		importPath                           string
		afferent, efferent, stdlib, external int
	}{
		{pathMain, 0, 1, 0, 0},
		{pathHub, 1, 2, 0, 0},
		{pathLeaf, 1, 0, 0, 0},
		// The import of the workspace sibling counts as external.
		{pathUtil, 1, 0, 1, 2},
		{pathAlone, 0, 0, 0, 0},
	}
	for _, tc := range testCases {
		t.Run(tc.importPath, func(t *testing.T) {
			m := metricsOf(t, tc.importPath)

			assert.Equal(t, tc.afferent, m.Afferent, "afferent")
			assert.Equal(t, tc.efferent, m.Efferent, "efferent")
			assert.Equal(t, tc.stdlib, m.Stdlib, "stdlib")
			assert.Equal(t, tc.external, m.External, "external")
		})
	}
}

func TestInstability(t *testing.T) {
	testCases := []struct {
		importPath string
		expected   float64
		reason     string
	}{
		{pathMain, 1.0, "no package imports the entry point"},
		{pathHub, 2.0 / 3.0, "Ce=2, Ca=1"},
		{pathLeaf, 0.0, "only incoming imports"},
		{pathUtil, 0.0, "imports outside of the module do not count"},
		{pathAlone, 1.0, "no coupling inside the module at all"},
	}
	for _, tc := range testCases {
		t.Run(tc.importPath, func(t *testing.T) {
			assert.InDelta(t, tc.expected, metricsOf(t, tc.importPath).Instability, 1e-9, tc.reason)
		})
	}
}

func TestAbstractness(t *testing.T) {
	testCases := []struct {
		importPath string
		expected   float64
		reason     string
	}{
		{pathHub, 0.25, "1 of 4 declarations is abstract"},
		{pathLeaf, 0.5, "3 of 6 declarations are abstract"},
		{pathUtil, 0.0, "no abstract declaration"},
		{pathAlone, 1.0, "the only declaration is abstract"},
		{pathMain, 0.0, "no exported declaration at all"},
	}
	for _, tc := range testCases {
		t.Run(tc.importPath, func(t *testing.T) {
			assert.InDelta(t, tc.expected, metricsOf(t, tc.importPath).Abstractness, 1e-9, tc.reason)
		})
	}
}

func TestDistance(t *testing.T) {
	testCases := []struct {
		importPath string
		signed     float64
		distance   float64
		reason     string
	}{
		{pathMain, 0.0, 0.0, "A=0, I=1: on the main sequence"},
		{pathUtil, -1.0, 1.0, "A=0, I=0: the zone of pain"},
		{pathAlone, 1.0, 1.0, "A=1, I=1: the zone of uselessness"},
		{pathLeaf, -0.5, 0.5, "A=0.5, I=0"},
		{pathHub, 2.0/3.0 + 0.25 - 1, 1 - 2.0/3.0 - 0.25, "A=0.25, I=2/3"},
	}
	for _, tc := range testCases {
		t.Run(tc.importPath, func(t *testing.T) {
			m := metricsOf(t, tc.importPath)

			assert.InDelta(t, tc.signed, m.SignedDistance, 1e-9, tc.reason)
			assert.InDelta(t, tc.distance, m.Distance, 1e-9, tc.reason)
		})
	}
}

func TestZone(t *testing.T) {
	testCases := []struct {
		importPath string
		expected   model.Zone
		reason     string
	}{
		{pathMain, model.MainSequence, "A=0, I=1: D=0"},
		{pathHub, model.MainSequence, "A=0.25, I=2/3: D=0.083"},
		{pathLeaf, model.MainSequence, "A=0.5, I=0: D=0.5 is exactly the threshold"},
		{pathUtil, model.ZoneOfPain, "A=0, I=0: concrete, and another package depends on it"},
		{pathAlone, model.ZoneOfUselessness, "A=1, I=1: abstract, and no package depends on it"},
	}
	for _, tc := range testCases {
		t.Run(tc.importPath, func(t *testing.T) {
			assert.Equal(t, tc.expected, metricsOf(t, tc.importPath).Zone, tc.reason)
		})
	}
}

func TestZoneWithASmallerThreshold(t *testing.T) {
	all := metrics.Of(testGraph(), 0.4)

	byPath := make(map[string]model.Metrics, len(all))
	for _, m := range all {
		byPath[m.Package.ImportPath] = m
	}

	assert.Equal(t, model.ZoneOfPain, byPath[pathLeaf].Zone, "D=0.5 is now above the threshold")
	assert.Equal(t, model.MainSequence, byPath[pathHub].Zone, "D=0.083 is still below the threshold")
}

func TestNoUnstableDependencies(t *testing.T) {
	// In the test graph the instability decreases with every import:
	// main (1) -> hub (2/3) -> leaf (0) and util (0).
	for _, m := range metrics.Of(testGraph(), maxDistance) {
		assert.Empty(t, m.UnstableDependencies, m.Package.ImportPath)
	}
}

func TestUnstableDependencies(t *testing.T) {
	all := metrics.Of(violationGraph(), maxDistance)

	byPath := make(map[string]model.Metrics, len(all))
	for _, m := range all {
		byPath[m.Package.ImportPath] = m
	}

	require.InDelta(t, 1.0/3.0, byPath[violationCore].Instability, 1e-9)
	require.InDelta(t, 2.0/3.0, byPath[violationHelper].Instability, 1e-9)

	assert.Equal(t, []string{violationHelper}, byPath[violationCore].UnstableDependencies,
		"core is more stable than the package it imports")
	assert.Empty(t, byPath[violationHelper].UnstableDependencies,
		"helper only imports packages that are more stable")
	assert.Empty(t, byPath[violationA].UnstableDependencies,
		"a is completely unstable, therefore every dependency is more stable")
}

func TestUnstableDependenciesOutsideOfTheModuleAreIgnored(t *testing.T) {
	graph := model.NewGraph()
	graph.AddPackage(model.Package{ImportPath: pathLeaf, ModulePath: module})
	graph.AddImport(model.Import{From: pathLeaf, To: "example.com/sibling/lib", Kind: model.WorkspaceSibling})
	graph.AddImport(model.Import{From: pathLeaf, To: "github.com/spf13/cobra", Kind: model.External})
	graph.AddImport(model.Import{From: pathLeaf, To: "fmt", Kind: model.Stdlib})

	all := metrics.Of(graph, maxDistance)

	require.Len(t, all, 1)
	assert.Empty(t, all[0].UnstableDependencies)
}

func TestDependencyWithTheSameInstabilityIsNoViolation(t *testing.T) {
	// w -> x -> y -> z: x and y both have Ca=1 and Ce=1, therefore both have
	// the instability 0.5. Stable dependencies is only violated if the
	// imported package is less stable, not if it is equally stable.
	graph := model.NewGraph()
	paths := []string{module + "/w", module + "/x", module + "/y", module + "/z"}
	for _, importPath := range paths {
		graph.AddPackage(model.Package{ImportPath: importPath, ModulePath: module})
	}
	for i := 0; i < len(paths)-1; i++ {
		graph.AddImport(model.Import{From: paths[i], To: paths[i+1], Kind: model.SameModule})
	}

	all := metrics.Of(graph, maxDistance)

	byPath := make(map[string]model.Metrics, len(all))
	for _, m := range all {
		byPath[m.Package.ImportPath] = m
	}
	require.InDelta(t, 0.5, byPath[module+"/x"].Instability, 1e-9)
	require.InDelta(t, 0.5, byPath[module+"/y"].Instability, 1e-9)
	assert.Empty(t, byPath[module+"/x"].UnstableDependencies)
}

func TestUnstableDependencyInAnotherModuleIsNoViolation(t *testing.T) {
	// The workspace sibling is part of the graph and less stable than the
	// importing package, but it belongs to another module and therefore
	// violates nothing.
	const (
		moduleB = "example.com/sibling"
		pathLib = moduleB + "/lib"
		pathOwn = moduleB + "/own"
		pathA2  = module + "/a2"
	)
	graph := model.NewGraph()
	graph.AddPackage(model.Package{ImportPath: pathLeaf, ModulePath: module})
	graph.AddPackage(model.Package{ImportPath: pathA2, ModulePath: module})
	graph.AddPackage(model.Package{ImportPath: pathLib, ModulePath: moduleB})
	graph.AddPackage(model.Package{ImportPath: pathOwn, ModulePath: moduleB})

	graph.AddImport(model.Import{From: pathA2, To: pathLeaf, Kind: model.SameModule})
	graph.AddImport(model.Import{From: pathLeaf, To: pathLib, Kind: model.WorkspaceSibling})
	graph.AddImport(model.Import{From: pathLib, To: pathOwn, Kind: model.SameModule})

	all := metrics.Of(graph, maxDistance)

	byPath := make(map[string]model.Metrics, len(all))
	for _, m := range all {
		byPath[m.Package.ImportPath] = m
	}
	require.InDelta(t, 0.0, byPath[pathLeaf].Instability, 1e-9, "Ca=1, Ce=0")
	require.InDelta(t, 1.0, byPath[pathLib].Instability, 1e-9, "Ca=0, Ce=1")
	assert.Empty(t, byPath[pathLeaf].UnstableDependencies)
}

func TestAbstractCoupling(t *testing.T) {
	const (
		port     = module + "/port"
		adapterA = module + "/adaptera"
		adapterB = module + "/adapterb"
		concrete = module + "/concrete"
		unused   = module + "/unused"
		user     = module + "/user"
	)
	graph := model.NewGraph()
	for _, importPath := range []string{port, adapterA, adapterB, concrete, unused, user} {
		graph.AddPackage(model.Package{ImportPath: importPath, ModulePath: module})
	}
	// Two adapters implement the port without importing it.
	graph.AddImplementation(model.Implementation{FromPkg: adapterA, Type: "A", ToPkg: port, Interface: "P"})
	graph.AddImplementation(model.Implementation{FromPkg: adapterB, Type: "B", ToPkg: port, Interface: "P"})
	// user imports the port and uses its interface.
	graph.AddImport(model.Import{From: user, To: port, Kind: model.SameModule, AbstractRefs: 1})
	// user also uses concrete, and only its structures.
	graph.AddImport(model.Import{From: user, To: concrete, Kind: model.SameModule, ConcreteRefs: 3})
	// An import from another module does not count.
	graph.AddImport(model.Import{From: "example.com/other/pkg", To: concrete, Kind: model.WorkspaceSibling, AbstractRefs: 5})

	all := metrics.Of(graph, maxDistance)
	byPath := make(map[string]model.Metrics, len(all))
	for _, m := range all {
		byPath[m.Package.ImportPath] = m
	}

	assert.InDelta(t, 1.0, byPath[port].AbstractCoupling, 1e-9, "two implementations and one abstract use")
	assert.InDelta(t, 0.0, byPath[concrete].AbstractCoupling, 1e-9, "only concrete uses, the other module does not count")
	assert.InDelta(t, 0.0, byPath[unused].AbstractCoupling, 1e-9, "no package depends on it")
	assert.InDelta(t, 0.0, byPath[adapterA].AbstractCoupling, 1e-9, "an adapter has no dependent")
}

func TestAbstractCouplingWithMixedUse(t *testing.T) {
	const (
		port = module + "/port"
		user = module + "/user"
	)
	graph := model.NewGraph()
	graph.AddPackage(model.Package{ImportPath: port, ModulePath: module})
	graph.AddPackage(model.Package{ImportPath: user, ModulePath: module})
	graph.AddImplementation(model.Implementation{FromPkg: user, Type: "U", ToPkg: port, Interface: "P"})
	graph.AddImport(model.Import{From: user, To: port, Kind: model.SameModule, AbstractRefs: 1, ConcreteRefs: 2})

	all := metrics.Of(graph, maxDistance)

	for _, m := range all {
		if m.Package.ImportPath == port {
			assert.InDelta(t, 0.5, m.AbstractCoupling, 1e-9, "one implementation and one abstract use against two concrete uses")
			return
		}
	}
	assert.Fail(t, "no metrics for the port")
}

func TestAbstractCouplingIgnoresImplementationsOfOtherModules(t *testing.T) {
	const port = module + "/port"
	graph := model.NewGraph()
	graph.AddPackage(model.Package{ImportPath: port, ModulePath: module})
	graph.AddPackage(model.Package{ImportPath: "example.com/other/adapter", ModulePath: "example.com/other"})
	graph.AddImplementation(model.Implementation{FromPkg: "example.com/other/adapter", Type: "A", ToPkg: port, Interface: "P"})
	graph.AddImport(model.Import{From: "example.com/other/adapter", To: port, Kind: model.WorkspaceSibling, ConcreteRefs: 1})

	all := metrics.Of(graph, maxDistance)

	for _, m := range all {
		if m.Package.ImportPath == port {
			assert.InDelta(t, 0.0, m.AbstractCoupling, 1e-9, "only the same module counts")
			return
		}
	}
	assert.Fail(t, "no metrics for the port")
}

func TestOfKeepsTheOrderOfThePackages(t *testing.T) {
	all := metrics.Of(testGraph(), maxDistance)

	require.Len(t, all, 5)
	assert.Equal(t, []string{pathMain, pathAlone, pathHub, pathLeaf, pathUtil}, importPaths(all))
}

func TestOfEmptyGraph(t *testing.T) {
	assert.Empty(t, metrics.Of(model.NewGraph(), maxDistance))
}

const (
	violationA      = module + "/a"
	violationB      = module + "/b"
	violationCore   = module + "/core"
	violationHelper = module + "/helper"
	violationSink1  = module + "/sink1"
	violationSink2  = module + "/sink2"
)

// violationGraph contains a violation of the stable dependencies invariant:
//
//	a, b   -> core            (a and b are completely unstable)
//	core   -> helper          (Ca=2, Ce=1, I=1/3)
//	helper -> sink1, sink2    (Ca=1, Ce=2, I=2/3)
//	sink1, sink2              (Ca=1, Ce=0, I=0)
//
// core is more stable than helper, therefore the import of helper by core
// violates the invariant.
func violationGraph() *model.Graph {
	graph := model.NewGraph()
	for _, importPath := range []string{violationA, violationB, violationCore, violationHelper, violationSink1, violationSink2} {
		graph.AddPackage(model.Package{ImportPath: importPath, ModulePath: module})
	}

	for _, imp := range []model.Import{
		{From: violationA, To: violationCore},
		{From: violationB, To: violationCore},
		{From: violationCore, To: violationHelper},
		{From: violationHelper, To: violationSink1},
		{From: violationHelper, To: violationSink2},
	} {
		imp.Kind = model.SameModule
		graph.AddImport(imp)
	}

	return graph
}

func metricsOf(t *testing.T, importPath string) model.Metrics {
	t.Helper()
	for _, m := range metrics.Of(testGraph(), maxDistance) {
		if m.Package.ImportPath == importPath {
			return m
		}
	}
	require.FailNowf(t, "no metrics", "no metrics for package %s", importPath)
	return model.Metrics{}
}

func importPaths(all []model.Metrics) []string {
	result := make([]string, len(all))
	for i, m := range all {
		result[i] = m.Package.ImportPath
	}
	return result
}
