package load

import (
	"go/types"

	"golang.org/x/tools/go/packages"

	"github.com/ftl/go-depend/model"
)

// declaration is an exported named type of one of the analyzed packages.
type declaration struct {
	importPath string
	name       string
	typ        types.Type
}

// implementationsOf returns the implementation edges between the analyzed
// packages: which exported type implements which exported interface of
// another package. Go satisfies an interface implicitly, therefore these
// edges exist without any import.
//
// A generic type takes part in an edge: it implements an interface for every
// instantiation. A generic interface takes part in no edge, because its
// methods carry the type parameters and no concrete type can implement it
// without an instantiation. go-depend does not find the implementation of a
// generic port.
func implementationsOf(pkgs []*packages.Package) []model.Implementation {
	interfaces, namedTypes := declarationsOfAll(pkgs)

	var result []model.Implementation
	added := make(map[model.Implementation]bool)
	for _, t := range namedTypes {
		for _, i := range interfaces {
			if t.importPath == i.importPath {
				// An implementation inside one package is no edge between
				// two packages.
				continue
			}
			if !implements(t.typ, i.typ) {
				continue
			}

			// The name of a declaration carries no type arguments: the
			// instantiation is the way to find the edge, and not the edge
			// itself. Therefore the same edge can be found more than one
			// time.
			edge := model.Implementation{
				FromPkg:   t.importPath,
				Type:      t.name,
				ToPkg:     i.importPath,
				Interface: i.name,
			}
			if !added[edge] {
				added[edge] = true
				result = append(result, edge)
			}
		}
	}

	return result
}

func declarationsOfAll(pkgs []*packages.Package) (interfaces []declaration, named []declaration) {
	analyzed := analyzedPackages(pkgs)

	for _, pkg := range pkgs {
		if pkg.Types == nil {
			continue
		}
		scope := pkg.Types.Scope()
		for _, name := range scope.Names() {
			object, ok := scope.Lookup(name).(*types.TypeName)
			if !ok || object.IsAlias() || !object.Exported() {
				continue
			}

			decl := declaration{importPath: pkg.PkgPath, name: name, typ: object.Type()}

			// An interface is never an implementation of another interface,
			// also if it has all of its methods.
			if types.IsInterface(object.Type()) {
				if isInterfaceWithMethods(object.Type()) {
					interfaces = append(interfaces, decl)
				}
				continue
			}
			named = append(named, decl)
		}

		i, n := instantiationsOf(pkg, analyzed)
		interfaces = append(interfaces, i...)
		named = append(named, n...)
	}
	return interfaces, named
}

func analyzedPackages(pkgs []*packages.Package) map[string]bool {
	result := make(map[string]bool, len(pkgs))
	for _, pkg := range pkgs {
		result[pkg.PkgPath] = true
	}
	return result
}

// instantiationsOf returns the instantiations of generic declarations that
// this package uses. An interface with type parameters has no method set that
// a type can implement, therefore only its instantiations take part in an
// edge. go-depend finds a generic port exactly if the module uses it.
func instantiationsOf(pkg *packages.Package, analyzed map[string]bool) (interfaces []declaration, named []declaration) {
	if pkg.TypesInfo == nil {
		return nil, nil
	}

	for _, instance := range pkg.TypesInfo.Instances {
		instantiated, ok := instance.Type.(*types.Named)
		if !ok || !isConcreteInstantiation(instantiated) {
			continue
		}
		object := instantiated.Obj()
		if object.Pkg() == nil || !object.Exported() || !analyzed[object.Pkg().Path()] {
			continue
		}

		decl := declaration{importPath: object.Pkg().Path(), name: object.Name(), typ: instantiated}
		if types.IsInterface(instantiated) {
			if isInterfaceWithMethods(instantiated) {
				interfaces = append(interfaces, decl)
			}
			continue
		}
		named = append(named, decl)
	}
	return interfaces, named
}

// isConcreteInstantiation reports whether all type arguments are types and no
// type parameter. Inside generic code a declaration is instantiated with the
// type parameters of that code, and such an instantiation has no method set
// that can be compared either.
func isConcreteInstantiation(instantiated *types.Named) bool {
	args := instantiated.TypeArgs()
	if args.Len() == 0 {
		return false
	}
	for i := range args.Len() {
		if _, ok := args.At(i).(*types.TypeParam); ok {
			return false
		}
	}
	return true
}

// isInterfaceWithMethods reports whether the type is a real interface. A
// constraint for a type parameter has no method set, and an empty interface
// is implemented by everything.
func isInterfaceWithMethods(t types.Type) bool {
	iface, ok := t.Underlying().(*types.Interface)
	return ok && iface.IsMethodSet() && iface.NumMethods() > 0
}

// implements reports whether the type or a pointer to it implements the
// interface. A method with a pointer receiver belongs only to the method set
// of the pointer type.
func implements(t types.Type, i types.Type) bool {
	iface, ok := i.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	return types.Implements(t, iface) || types.Implements(types.NewPointer(t), iface)
}
