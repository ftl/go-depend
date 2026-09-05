package load

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ftl/go-depend/model"
)

const (
	implPort    = "example.com/impl/port"
	implAdapter = "example.com/impl/adapter"
	implWired   = "example.com/impl/wired"
)

func TestImplementations(t *testing.T) {
	graph := loadImplementFixture(t)

	assert.Equal(t, []model.Implementation{
		// Box implements the generic port port.Store[string], which the
		// package wired names.
		{FromPkg: implAdapter, Type: "Box", ToPkg: implPort, Interface: "Store"},
		// A generic type implements the port for every instantiation.
		{FromPkg: implAdapter, Type: "Generic", ToPkg: implPort, Interface: "Corpus"},
		// The accidental match: Printer knows nothing about the port.
		{FromPkg: implAdapter, Type: "Printer", ToPkg: implPort, Interface: "Quitter"},
		// A pointer receiver: only *Random implements the port.
		{FromPkg: implAdapter, Type: "Random", ToPkg: implPort, Interface: "Corpus"},
		// A value receiver.
		{FromPkg: implAdapter, Type: "Ring", ToPkg: implPort, Interface: "Corpus"},
		// The implementing package imports the port as well.
		{FromPkg: implWired, Type: "Service", ToPkg: implPort, Interface: "Quitter"},
	}, graph.Implementations())
}

func TestImplementationInTheSamePackageIsNoEdge(t *testing.T) {
	graph := loadImplementFixture(t)

	for _, impl := range graph.Implementations() {
		assert.NotEqual(t, impl.FromPkg, impl.ToPkg, "%s implements %s", impl.Type, impl.Interface)
		assert.NotEqual(t, "Local", impl.Type)
	}
}

func TestEmptyInterfaceIsNoPort(t *testing.T) {
	graph := loadImplementFixture(t)

	for _, impl := range graph.Implementations() {
		assert.NotEqual(t, "Empty", impl.Interface, "every type implements the empty interface")
	}
}

func TestAccidentalMatchIsReported(t *testing.T) {
	graph := loadImplementFixture(t)

	// Printer has a method Quit for its own reason and knows nothing about
	// the package port. go-depend reports the edge nevertheless: in Go this
	// dependency is real, and a change of the method breaks the other
	// package. A filter for such a match removes more real ports than
	// accidents, see baseline.md.
	assert.Contains(t, graph.Implementations(), model.Implementation{
		FromPkg: implAdapter, Type: "Printer", ToPkg: implPort, Interface: "Quitter",
	})
}

func TestGenericPortIsFoundThroughItsInstantiation(t *testing.T) {
	graph := loadImplementFixture(t)

	// wired names port.Store[string], therefore this instantiation is a port
	// of the module, and Box implements it. The name of the edge carries no
	// type argument: the instantiation is the way to find the edge.
	assert.Contains(t, graph.Implementations(), model.Implementation{
		FromPkg: implAdapter, Type: "Box", ToPkg: implPort, Interface: "Store",
	})
}

func TestGenericImplementerNeedsItsOwnInstantiation(t *testing.T) {
	graph := loadImplementFixture(t)

	// GenBox[T] implements port.Store[T] for every instantiation, but no code
	// of the module names GenBox[string]. Without an instantiation there is
	// no method set that can be compared, and the edge stays invisible. This
	// is the rest of the gap: a generic type that implements a generic port.
	for _, impl := range graph.Implementations() {
		assert.NotEqual(t, "GenBox", impl.Type)
	}
}

func TestIncomingImplementations(t *testing.T) {
	graph := loadImplementFixture(t)

	incoming := graph.IncomingImplementations(implPort)

	require.Len(t, incoming, 6, "all six implementations point at the port")
	assert.Empty(t, graph.IncomingImplementations(implAdapter))
}

func TestOutgoingImplementations(t *testing.T) {
	graph := loadImplementFixture(t)

	assert.Equal(t, []model.Implementation{
		{FromPkg: implWired, Type: "Service", ToPkg: implPort, Interface: "Quitter"},
	}, graph.OutgoingImplementations(implWired))
	assert.Empty(t, graph.OutgoingImplementations(implPort))
}

func TestImplementationsWithoutAnyImport(t *testing.T) {
	graph := loadImplementFixture(t)

	// The whole point of the metric: adapter implements the port, and no
	// import of the graph records this dependency.
	assert.Empty(t, graph.Outgoing(implAdapter))
	assert.Empty(t, graph.Incoming(implAdapter))
	assert.Len(t, graph.OutgoingImplementations(implAdapter), 5)
}

func TestFixtureModuleHasNoImplementations(t *testing.T) {
	graph := loadFixture(t)

	assert.Empty(t, graph.Implementations(), "no type of the fixture implements decl.Reader")
}

func loadImplementFixture(t *testing.T) *model.Graph {
	t.Helper()
	graph, err := Load(Options{Dir: "testdata/implement"}, "./...")
	require.NoError(t, err)
	return graph
}
