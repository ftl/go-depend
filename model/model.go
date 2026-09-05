// Package model holds the vocabulary that all other packages of go-depend
// share: the packages of a module, the imports between them, and the graph
// that connects both.
package model

import (
	"cmp"
	"slices"
	"strings"
)

// ImportKind classifies an import in relation to the module of the importing
// package.
type ImportKind int

const (
	// SameModule marks an import of a package from the same module. Only
	// imports of this kind contribute to the coupling metrics.
	SameModule ImportKind = iota
	// WorkspaceSibling marks an import of a package from another module of
	// the same workspace.
	WorkspaceSibling
	// External marks an import of a package from another module.
	External
	// Stdlib marks an import of a package from the Go standard library.
	Stdlib
)

// Package is one Go package of a module.
type Package struct {
	// ImportPath identifies the package, it is unique within a workspace.
	ImportPath string
	// ModulePath is the path of the module that contains this package.
	ModulePath string
	// Dir is the directory of the package, relative to the root of its
	// module and separated by slashes. The root itself is ".".
	Dir string
	// Types is the number of exported named types of the package.
	Types int
	// Funcs is the number of exported package-level functions of the package.
	Funcs int
	// Abstract is the number of exported named types and package-level
	// functions of the package that are abstract: interfaces, and types and
	// functions with type parameters.
	Abstract int
}

// RelativePath returns the import path of the package, relative to its
// module. The root package of a module is ".".
func (p Package) RelativePath() string {
	if p.ImportPath == p.ModulePath {
		return "."
	}
	return strings.TrimPrefix(p.ImportPath, p.ModulePath+"/")
}

// Import is one import of the package From to the package To.
type Import struct {
	From string
	To   string
	Kind ImportKind

	// AbstractRefs is the number of distinct exported symbols of the imported
	// package that the importing package uses and that are abstract: an
	// interface, or a named function type.
	AbstractRefs int
	// ConcreteRefs is the number of distinct exported symbols of the imported
	// package that the importing package uses and that are concrete.
	ConcreteRefs int
}

// Implementation is a type of one package that implements an interface of
// another package. Go satisfies an interface implicitly, therefore this
// dependency exists without any import that records it.
type Implementation struct {
	// FromPkg is the import path of the package that contains the type.
	FromPkg string
	// Type is the name of the type that implements the interface.
	Type string
	// ToPkg is the import path of the package that contains the interface.
	ToPkg string
	// Interface is the name of the implemented interface.
	Interface string
}

// Graph contains the packages of one or more modules and the imports between
// them. Only packages of the analyzed modules are contained as packages, while
// imports may also point to packages outside of those modules.
type Graph struct {
	packages map[string]Package
	outgoing map[string][]Import
	incoming map[string][]Import

	implementations []Implementation
	outgoingImpl    map[string][]Implementation
	incomingImpl    map[string][]Implementation
}

// NewGraph returns an empty graph.
func NewGraph() *Graph {
	return &Graph{
		packages:     make(map[string]Package),
		outgoing:     make(map[string][]Import),
		incoming:     make(map[string][]Import),
		outgoingImpl: make(map[string][]Implementation),
		incomingImpl: make(map[string][]Implementation),
	}
}

// AddPackage adds a package to the graph and replaces any package that was
// added before with the same import path.
func (g *Graph) AddPackage(pkg Package) {
	g.packages[pkg.ImportPath] = pkg
}

// AddImport adds an import to the graph. Neither the importing nor the
// imported package needs to be part of the graph.
func (g *Graph) AddImport(imp Import) {
	g.outgoing[imp.From] = insertSorted(g.outgoing[imp.From], imp, func(imp Import) string { return imp.To })
	g.incoming[imp.To] = insertSorted(g.incoming[imp.To], imp, func(imp Import) string { return imp.From })
}

func insertSorted(imports []Import, imp Import, key func(Import) string) []Import {
	imports = append(imports, imp)
	slices.SortFunc(imports, func(a, b Import) int { return cmp.Compare(key(a), key(b)) })
	return imports
}

// AddImplementation adds an implementation edge to the graph. Neither of the
// two packages needs to be part of the graph.
func (g *Graph) AddImplementation(impl Implementation) {
	g.implementations = insertSortedImplementation(g.implementations, impl)
	g.outgoingImpl[impl.FromPkg] = insertSortedImplementation(g.outgoingImpl[impl.FromPkg], impl)
	g.incomingImpl[impl.ToPkg] = insertSortedImplementation(g.incomingImpl[impl.ToPkg], impl)
}

func insertSortedImplementation(implementations []Implementation, impl Implementation) []Implementation {
	implementations = append(implementations, impl)
	slices.SortFunc(implementations, func(a, b Implementation) int {
		return cmp.Or(
			cmp.Compare(a.FromPkg, b.FromPkg),
			cmp.Compare(a.Type, b.Type),
			cmp.Compare(a.ToPkg, b.ToPkg),
			cmp.Compare(a.Interface, b.Interface),
		)
	})
	return implementations
}

// Implementations returns all implementation edges of the graph, sorted by
// the implementing package and type. The result must not be modified.
func (g *Graph) Implementations() []Implementation {
	return g.implementations
}

// OutgoingImplementations returns the interfaces of other packages that the
// types of the given package implement. The result must not be modified.
func (g *Graph) OutgoingImplementations(importPath string) []Implementation {
	return g.outgoingImpl[importPath]
}

// IncomingImplementations returns the types of other packages that implement
// an interface of the given package. The result must not be modified.
func (g *Graph) IncomingImplementations(importPath string) []Implementation {
	return g.incomingImpl[importPath]
}

// Package returns the package with the given import path.
func (g *Graph) Package(importPath string) (Package, bool) {
	pkg, ok := g.packages[importPath]
	return pkg, ok
}

// Packages returns all packages of the graph, sorted by their import path.
func (g *Graph) Packages() []Package {
	packages := make([]Package, 0, len(g.packages))
	for _, pkg := range g.packages {
		packages = append(packages, pkg)
	}
	slices.SortFunc(packages, func(a, b Package) int { return cmp.Compare(a.ImportPath, b.ImportPath) })
	return packages
}

// Outgoing returns the imports of the package with the given import path,
// sorted by the imported package. The result must not be modified.
func (g *Graph) Outgoing(importPath string) []Import {
	return g.outgoing[importPath]
}

// Incoming returns the imports that point to the package with the given
// import path, sorted by the importing package. The result must not be
// modified.
func (g *Graph) Incoming(importPath string) []Import {
	return g.incoming[importPath]
}
