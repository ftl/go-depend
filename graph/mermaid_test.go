package graph_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ftl/go-depend/graph"
	"github.com/ftl/go-depend/model"
)

func TestMermaidOneModule(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Outgoing: true, Depth: 1}, pathHub)

	// The imports of cobra and fmt are missing: a package outside of the
	// module is only part of the selection if it is the root.
	assert.Equal(t, `graph LR
  p0["hub"]
  p1["left"]
  p2["right"]
  p0 --> p1
  p0 --> p2
`, mermaidOf(t, selection))
}

func TestMermaidWithoutSubgraphForOneModule(t *testing.T) {
	output := mermaidOf(t, graph.Select(testGraph(), graph.Options{Outgoing: true}, pathMain))

	assert.NotContains(t, output, "subgraph")
	assert.NotContains(t, output, "end")
}

func TestMermaidWithSeveralModules(t *testing.T) {
	const sibling = "example.com/sibling"
	g := model.NewGraph()
	g.AddPackage(model.Package{ImportPath: pathHub, ModulePath: module})
	g.AddPackage(model.Package{ImportPath: sibling + "/lib", ModulePath: sibling})
	addImport(g, pathHub, sibling+"/lib", model.WorkspaceSibling)

	selection := graph.Select(g, graph.Options{Outgoing: true}, pathHub)

	assert.Equal(t, `graph LR
  subgraph p0m["example.com/module"]
    p0["hub"]
  end
  subgraph p1m["example.com/sibling"]
    p1["lib"]
  end
  p0 -.-> p1
`, mermaidOf(t, selection))
}

func TestMermaidWithAnExternalRoot(t *testing.T) {
	selection := graph.Select(testGraph(), graph.Options{Incoming: true, Depth: 1}, pathCobra)

	assert.Equal(t, `graph LR
  p0["deep"]
  p1["left"]
  p2["github.com/spf13/cobra"]
  p0 -.-> p2
  p1 --> p0
  p1 -.-> p2
`, mermaidOf(t, selection))
}

func TestMermaidWithTheModuleRoot(t *testing.T) {
	output := mermaidOf(t, graph.Select(testGraph(), graph.Options{Outgoing: true, Depth: 1}, pathMain))

	assert.Contains(t, output, `p0["."]`, "the root package of the module")
}

func TestMermaidWithImplementations(t *testing.T) {
	selection := graph.Select(testGraph(),
		graph.Options{Incoming: true, Depth: 1, Edges: graph.ImplementationEdges}, pathHub)

	assert.Equal(t, `graph LR
  p0["alone"]
  p1["hub"]
  p0 -. implements 2 .-> p1
`, mermaidOf(t, selection), "two interfaces between the same packages are one arrow")
}

func TestMermaidWithBothKindsOfEdge(t *testing.T) {
	selection := graph.Select(testGraph(),
		graph.Options{Incoming: true, Depth: 1, Edges: graph.AllEdges}, pathHub)

	output := mermaidOf(t, selection)

	assert.Contains(t, output, "p0 --> p2", "main imports hub")
	assert.Contains(t, output, "p1 -. implements 2 .-> p2", "alone implements hub")
}

func TestMermaidWithoutPackages(t *testing.T) {
	assert.Equal(t, "graph LR\n", mermaidOf(t, graph.Selection{}))
}

func mermaidOf(t *testing.T, selection graph.Selection) string {
	t.Helper()
	out := &bytes.Buffer{}
	require.NoError(t, graph.Mermaid(out, selection))
	return out.String()
}
