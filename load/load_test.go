package load

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ftl/go-depend/model"
)

const (
	fixtureModule = "example.com/fixture"
	fixtureMain   = "example.com/fixture"
	fixtureScan   = "example.com/fixture/scan"
	fixtureModel  = "example.com/fixture/model"
	fixtureAlone  = "example.com/fixture/alone"
	fixtureGen    = "example.com/fixture/gen"
	fixtureDecl   = "example.com/fixture/decl"
)

func TestLoadFixtureModule(t *testing.T) {
	graph := loadFixture(t)

	packages := graph.Packages()

	require.Len(t, packages, 6)
	assert.Equal(t, []string{fixtureMain, fixtureAlone, fixtureDecl, fixtureGen, fixtureModel, fixtureScan}, importPaths(packages))
	for _, pkg := range packages {
		assert.Equal(t, fixtureModule, pkg.ModulePath, pkg.ImportPath)
	}
}

func TestPackageDirIsRelativeToTheModuleRoot(t *testing.T) {
	graph := loadFixture(t)

	main, ok := graph.Package(fixtureMain)
	require.True(t, ok)
	assert.Equal(t, ".", main.Dir)

	scan, ok := graph.Package(fixtureScan)
	require.True(t, ok)
	assert.Equal(t, "scan", scan.Dir)
}

func TestImportsAreClassified(t *testing.T) {
	graph := loadFixture(t)

	assert.Equal(t, []model.Import{
		// main uses scan.Name and fmt.Println, both concrete.
		{From: fixtureMain, To: fixtureScan, Kind: model.SameModule, ConcreteRefs: 1},
		{From: fixtureMain, To: "fmt", Kind: model.Stdlib, ConcreteRefs: 1},
	}, graph.Outgoing(fixtureMain))
}

func TestPackageWithoutImports(t *testing.T) {
	graph := loadFixture(t)

	assert.Empty(t, graph.Outgoing(fixtureAlone))
	assert.Empty(t, graph.Incoming(fixtureAlone))
}

func TestIncomingImportsOfAnExternalPackage(t *testing.T) {
	graph := loadFixture(t)

	assert.Equal(t, []string{fixtureMain, fixtureScan}, sources(graph.Incoming("fmt")))
}

func TestExcludedFileContributesNoImports(t *testing.T) {
	withGen := loadFixture(t)
	assert.Equal(t, []string{"encoding/json", fixtureModel}, targets(withGen.Outgoing(fixtureGen)))

	withoutGen, err := Load(Options{Dir: "testdata/module", Exclude: []string{`\.pb\.go$`}}, "./...")
	require.NoError(t, err)

	assert.Equal(t, []string{fixtureModel}, targets(withoutGen.Outgoing(fixtureGen)))
	assert.Len(t, withoutGen.Packages(), 6, "the package itself remains in the graph")
}

func TestLoadWorkspace(t *testing.T) {
	// A workspace root is no module, therefore the patterns address the
	// modules of the workspace one by one.
	graph, err := Load(Options{Dir: "testdata/workspace"}, "./a/...", "./b/...")
	require.NoError(t, err)

	require.Len(t, graph.Packages(), 2)
	assert.Equal(t, []model.Import{
		{From: "example.com/a", To: "example.com/b/lib", Kind: model.WorkspaceSibling, ConcreteRefs: 1},
		{From: "example.com/a", To: "fmt", Kind: model.Stdlib, ConcreteRefs: 1},
	}, graph.Outgoing("example.com/a"))
}

func TestDeclarationsAreCounted(t *testing.T) {
	graph := loadFixture(t)

	testCases := []struct {
		importPath string
		expected   model.Package
	}{
		// Reader, Cache, Celsius and Client count as types, New and Map as
		// functions. Reader, Cache and Map are abstract. The alias, the
		// unexported declarations, the constant, the variable and the two
		// methods do not count.
		{fixtureDecl, model.Package{Types: 4, Funcs: 2, Abstract: 3}},
		{fixtureModel, model.Package{Types: 1, Funcs: 0, Abstract: 0}},
		{fixtureScan, model.Package{Types: 0, Funcs: 1, Abstract: 0}},
		{fixtureAlone, model.Package{Types: 0, Funcs: 1, Abstract: 0}},
		{fixtureGen, model.Package{Types: 0, Funcs: 2, Abstract: 0}},
		// The main function is not exported.
		{fixtureMain, model.Package{Types: 0, Funcs: 0, Abstract: 0}},
	}
	for _, tc := range testCases {
		t.Run(tc.importPath, func(t *testing.T) {
			pkg, ok := graph.Package(tc.importPath)
			require.True(t, ok)
			assert.Equal(t, tc.expected.Types, pkg.Types, "types")
			assert.Equal(t, tc.expected.Funcs, pkg.Funcs, "funcs")
			assert.Equal(t, tc.expected.Abstract, pkg.Abstract, "abstract")
		})
	}
}

func TestExcludedFileContributesNoDeclarations(t *testing.T) {
	graph, err := Load(Options{Dir: "testdata/module", Exclude: []string{`\.pb\.go$`}}, "./...")
	require.NoError(t, err)

	pkg, ok := graph.Package(fixtureGen)
	require.True(t, ok)
	assert.Equal(t, 1, pkg.Funcs, "only the function of gen.go remains")
}

func TestPackageThatDoesNotCompileIsRefused(t *testing.T) {
	_, err := Load(Options{Dir: "testdata/broken"}, "./...")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "example.com/broken")
	assert.Contains(t, err.Error(), "could not import example.com/does/not/exist")
}

func TestReferencesAreClassified(t *testing.T) {
	graph, err := Load(Options{Dir: "testdata/implement"}, "./...")
	require.NoError(t, err)

	// wired uses the interfaces port.Corpus and port.Store, calls their
	// methods Len and Put, and takes the function type port.Handler. All five
	// are abstract: another implementation can take the place of this one.
	// wired names port.Corpus twice, and it counts one time.
	outgoing := graph.Outgoing("example.com/impl/wired")
	require.Len(t, outgoing, 1)
	assert.Equal(t, 5, outgoing[0].AbstractRefs, "Corpus, Corpus.Len, Handler, Store and Store.Put")
	assert.Equal(t, 0, outgoing[0].ConcreteRefs)
}

func TestConcreteReferences(t *testing.T) {
	graph := loadFixture(t)

	// scan uses the structure model.Package and its field Name, both concrete.
	outgoing := graph.Outgoing(fixtureScan)
	for _, imp := range outgoing {
		if imp.To != fixtureModel {
			continue
		}
		assert.Equal(t, 0, imp.AbstractRefs)
		assert.Equal(t, 2, imp.ConcreteRefs, "model.Package and Package.Name")
		return
	}
	assert.Fail(t, "scan does not import model")
}

func TestReferencesOfAnExcludedFileDoNotCount(t *testing.T) {
	withGenerated := loadFixture(t)
	withoutGenerated, err := Load(Options{Dir: "testdata/module", Exclude: []string{`\.pb\.go$`}}, "./...")
	require.NoError(t, err)

	assert.Equal(t, 1, referencesTo(t, withGenerated, fixtureGen, "encoding/json"))
	assert.Equal(t, 0, referencesTo(t, withoutGenerated, fixtureGen, "encoding/json"))
}

func referencesTo(t *testing.T, graph *model.Graph, from string, to string) int {
	t.Helper()
	for _, imp := range graph.Outgoing(from) {
		if imp.To == to {
			return imp.AbstractRefs + imp.ConcreteRefs
		}
	}
	return 0
}

func TestPatternWithoutPackages(t *testing.T) {
	_, err := Load(Options{Dir: "testdata"}, "./empty/...")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no packages match")
}

func TestUnknownPattern(t *testing.T) {
	_, err := Load(Options{Dir: "testdata/module"}, "./does-not-exist/...")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "does-not-exist")
}

func TestInvalidExcludeExpression(t *testing.T) {
	_, err := Load(Options{Dir: "testdata/module", Exclude: []string{"("}}, "./...")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid exclude expression")
}

func TestClassify(t *testing.T) {
	const own = "example.com/own"
	workspace := []string{own, "example.com/sibling", "example.com/own/nested"}

	testCases := []struct {
		importPath string
		expected   model.ImportKind
	}{
		{"fmt", model.Stdlib},
		{"net/http", model.Stdlib},
		{"unsafe", model.Stdlib},
		{own, model.SameModule},
		{own + "/pkg", model.SameModule},
		{"example.com/sibling/pkg", model.WorkspaceSibling},
		{"example.com/own/nested/pkg", model.WorkspaceSibling},
		{"github.com/spf13/cobra", model.External},
		{"example.com/other", model.External},
	}
	for _, tc := range testCases {
		t.Run(tc.importPath, func(t *testing.T) {
			assert.Equal(t, tc.expected, classify(tc.importPath, own, workspace))
		})
	}
}

func loadFixture(t *testing.T) *model.Graph {
	t.Helper()
	graph, err := Load(Options{Dir: "testdata/module"}, "./...")
	require.NoError(t, err)
	return graph
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

func TestResolveOne(t *testing.T) {
	testCases := []struct {
		pattern  string
		expected string
	}{
		{".", fixtureMain},
		{"./model", fixtureModel},
		{"./gen", fixtureGen},
	}
	for _, tc := range testCases {
		t.Run(tc.pattern, func(t *testing.T) {
			importPath, err := ResolveOne(Options{Dir: "testdata/module"}, tc.pattern)

			require.NoError(t, err)
			assert.Equal(t, tc.expected, importPath)
		})
	}
}

func TestResolveOneFailsForSeveralPackages(t *testing.T) {
	_, err := ResolveOne(Options{Dir: "testdata/module"}, "./...")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "matches 6 packages, but it must match exactly one")
}

func TestResolveOneFailsForAnUnknownPackage(t *testing.T) {
	_, err := ResolveOne(Options{Dir: "testdata/module"}, "example.com/does/not/exist")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "example.com/does/not/exist")
}
