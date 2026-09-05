package load

import (
	"go/types"
	"regexp"

	"golang.org/x/tools/go/packages"
)

// references counts the symbols of one imported package.
type references struct {
	abstract int
	concrete int
}

// referencesOf counts the exported symbols of other packages that this
// package uses, by the import path of the package that declares them. Every
// symbol counts one time, however often the code uses it: a structure in a
// loop must not control the result.
func referencesOf(pkg *packages.Package, exclude []*regexp.Regexp) map[string]references {
	result := make(map[string]references)
	if pkg.TypesInfo == nil {
		return result
	}

	counted := make(map[types.Object]bool)
	for identifier, object := range pkg.TypesInfo.Uses {
		if object.Pkg() == nil || object.Pkg().Path() == pkg.PkgPath || counted[object] {
			// A builtin, a symbol of this package, or a symbol that was
			// already counted. The check for this package changes no result,
			// because a package never imports itself. It is only there
			// because most of the used symbols belong to this package.
			continue
		}
		if isExcluded(pkg.Fset.Position(identifier.Pos()).Filename, pkg.Module.Dir, exclude) {
			continue
		}
		counted[object] = true

		importPath := object.Pkg().Path()
		count := result[importPath]
		if isAbstractSymbol(object) {
			count.abstract++
		} else {
			count.concrete++
		}
		result[importPath] = count
	}

	return result
}

// isAbstractSymbol reports whether the used symbol is a point where another
// implementation can take the place of this one: an interface, a named
// function type, or a method that is called through an interface. Everything
// else is concrete, also a package-level function and a method of a concrete
// type.
func isAbstractSymbol(object types.Object) bool {
	switch object := object.(type) {
	case *types.TypeName:
		if isInterfaceWithMethods(object.Type()) {
			return true
		}
		_, isFunc := object.Type().Underlying().(*types.Signature)
		return isFunc
	case *types.Func:
		return isInterfaceMethod(object)
	default:
		return false
	}
}

// isInterfaceMethod reports whether the function is a method of an interface.
// The caller of such a method uses the abstraction, and not one of its
// implementations.
func isInterfaceMethod(object *types.Func) bool {
	signature, ok := object.Type().(*types.Signature)
	if !ok || signature.Recv() == nil {
		return false
	}
	return types.IsInterface(signature.Recv().Type())
}
