// Package load builds the model graph of the Go packages that match a set of
// patterns. It is the only package of go-depend that reads Go source code.
package load

import (
	"errors"
	"fmt"
	"go/ast"
	"go/types"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"golang.org/x/tools/go/packages"

	"github.com/ftl/go-depend/model"
)

// loadMode contains everything that is needed to build the graph: the package
// and module identity, the file names, the syntax trees to find the imports of
// the individual files, and the type information to count the declarations.
const loadMode = packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
	packages.NeedModule | packages.NeedSyntax | packages.NeedTypes

// Options control how the patterns are resolved.
type Options struct {
	// Dir is the directory the patterns are resolved in. The empty string
	// means the current working directory.
	Dir string
	// Exclude contains regular expressions. A file is ignored completely if
	// its module-relative path matches one of them.
	Exclude []string
}

// Load resolves the given patterns and returns the graph of all matching
// packages and their imports.
func Load(options Options, patterns ...string) (*model.Graph, error) {
	exclude, err := compileExclude(options.Exclude)
	if err != nil {
		return nil, err
	}

	config := &packages.Config{Mode: loadMode, Dir: options.Dir, Tests: false}
	pkgs, err := packages.Load(config, patterns...)
	if err != nil {
		return nil, fmt.Errorf("cannot load packages: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages match %s", strings.Join(patterns, " "))
	}
	if err := loadErrors(pkgs); err != nil {
		return nil, err
	}

	return buildGraph(pkgs, exclude)
}

func compileExclude(expressions []string) ([]*regexp.Regexp, error) {
	result := make([]*regexp.Regexp, 0, len(expressions))
	for _, expression := range expressions {
		compiled, err := regexp.Compile(expression)
		if err != nil {
			return nil, fmt.Errorf("invalid exclude expression %q: %w", expression, err)
		}
		result = append(result, compiled)
	}
	return result, nil
}

// loadErrors returns all errors that occurred while the packages were loaded.
// Metrics of a package that does not compile would be wrong without any
// indication, therefore go-depend refuses to analyze it.
func loadErrors(pkgs []*packages.Package) error {
	var result []error
	for _, pkg := range pkgs {
		for _, err := range pkg.Errors {
			result = append(result, fmt.Errorf("%s: %s", pkg.PkgPath, err))
		}
	}
	return errors.Join(result...)
}

func buildGraph(pkgs []*packages.Package, exclude []*regexp.Regexp) (*model.Graph, error) {
	modulePaths := modulePathsOf(pkgs)
	graph := model.NewGraph()

	for _, pkg := range pkgs {
		if pkg.Module == nil {
			return nil, fmt.Errorf("package %s belongs to no module", pkg.PkgPath)
		}

		declarations := declarationsOf(pkg, exclude)
		graph.AddPackage(model.Package{
			ImportPath: pkg.PkgPath,
			ModulePath: pkg.Module.Path,
			Dir:        relativeDir(pkg),
			Types:      declarations.types,
			Funcs:      declarations.funcs,
			Abstract:   declarations.abstract,
		})

		for _, importPath := range importsOf(pkg, exclude) {
			graph.AddImport(model.Import{
				From: pkg.PkgPath,
				To:   importPath,
				Kind: classify(importPath, pkg.Module.Path, modulePaths),
			})
		}
	}

	return graph, nil
}

// modulePathsOf returns the paths of all modules that contain at least one of
// the given packages. In a workspace this is the set of the modules in scope.
func modulePathsOf(pkgs []*packages.Package) []string {
	var result []string
	for _, pkg := range pkgs {
		if pkg.Module != nil && !slices.Contains(result, pkg.Module.Path) {
			result = append(result, pkg.Module.Path)
		}
	}
	return result
}

// relativeDir returns the directory of the package, relative to the root of
// its module and separated by slashes.
func relativeDir(pkg *packages.Package) string {
	if pkg.Dir == "" || pkg.Module.Dir == "" {
		return "."
	}
	relative, err := filepath.Rel(pkg.Module.Dir, pkg.Dir)
	if err != nil {
		return "."
	}
	return filepath.ToSlash(relative)
}

// importsOf returns the import paths of all files of the package that are not
// excluded, sorted and without duplicates.
func importsOf(pkg *packages.Package, exclude []*regexp.Regexp) []string {
	var result []string
	for _, file := range pkg.Syntax {
		if isExcluded(fileNameOf(pkg, file), pkg.Module.Dir, exclude) {
			continue
		}
		for _, imp := range file.Imports {
			importPath, err := strconv.Unquote(imp.Path.Value)
			if err != nil || importPath == "C" {
				continue
			}
			if !slices.Contains(result, importPath) {
				result = append(result, importPath)
			}
		}
	}
	slices.Sort(result)
	return result
}

// declarations counts the exported declarations of a package that contribute
// to its abstractness.
type declarations struct {
	types    int
	funcs    int
	abstract int
}

// declarationsOf counts the exported named types and package-level functions
// of the package, and how many of them are abstract. Methods, constants,
// variables and type aliases do not count: only named types and functions are
// the Go equivalent of a class, and a method is a member of a type.
func declarationsOf(pkg *packages.Package, exclude []*regexp.Regexp) declarations {
	var result declarations
	if pkg.Types == nil {
		return result
	}

	scope := pkg.Types.Scope()
	for _, name := range scope.Names() {
		object := scope.Lookup(name)
		if !object.Exported() || isExcluded(positionOf(pkg, object), pkg.Module.Dir, exclude) {
			continue
		}

		switch object := object.(type) {
		case *types.TypeName:
			if object.IsAlias() {
				continue
			}
			result.types++
			if isAbstractType(object) {
				result.abstract++
			}
		case *types.Func:
			// Methods are members of their type and not part of the package
			// scope, therefore only package-level functions arrive here.
			result.funcs++
			if isAbstractFunc(object) {
				result.abstract++
			}
		}
	}

	return result
}

// isAbstractType reports whether the named type is an abstraction: an
// interface, or a type with type parameters.
func isAbstractType(object *types.TypeName) bool {
	if types.IsInterface(object.Type()) {
		return true
	}
	named, ok := object.Type().(*types.Named)
	return ok && named.TypeParams().Len() > 0
}

// isAbstractFunc reports whether the function is an abstraction: it has type
// parameters.
func isAbstractFunc(object *types.Func) bool {
	signature, ok := object.Type().(*types.Signature)
	return ok && signature.TypeParams().Len() > 0
}

func positionOf(pkg *packages.Package, object types.Object) string {
	return pkg.Fset.Position(object.Pos()).Filename
}

func fileNameOf(pkg *packages.Package, file *ast.File) string {
	return pkg.Fset.Position(file.Package).Filename
}

// isExcluded matches the exclude expressions against the path of the file,
// relative to the root of the module. A file outside of the module directory,
// for example a file that cgo generated, is never excluded.
func isExcluded(fileName string, moduleDir string, exclude []*regexp.Regexp) bool {
	if len(exclude) == 0 {
		return false
	}
	relative, err := filepath.Rel(moduleDir, fileName)
	if err != nil || strings.HasPrefix(relative, "..") {
		return false
	}
	relative = filepath.ToSlash(relative)

	for _, expression := range exclude {
		if expression.MatchString(relative) {
			return true
		}
	}
	return false
}

// classify determines the relation of the imported package to the module of
// the importing package.
func classify(importPath string, modulePath string, modulePaths []string) model.ImportKind {
	if isStdlib(importPath) {
		return model.Stdlib
	}
	switch longestModulePath(importPath, modulePaths) {
	case modulePath:
		return model.SameModule
	case "":
		return model.External
	default:
		return model.WorkspaceSibling
	}
}

// isStdlib reports whether the import path belongs to the Go standard
// library. Only import paths outside of the standard library have a domain
// name as their first element, and only a domain name contains a dot.
func isStdlib(importPath string) bool {
	firstElement, _, _ := strings.Cut(importPath, "/")
	return !strings.Contains(firstElement, ".")
}

// longestModulePath returns the longest of the given module paths that
// contains the imported package. Nested modules make more than one match
// possible, and the longest match is the module the package belongs to.
func longestModulePath(importPath string, modulePaths []string) string {
	result := ""
	for _, modulePath := range modulePaths {
		if !inModule(importPath, modulePath) {
			continue
		}
		if len(modulePath) > len(result) {
			result = modulePath
		}
	}
	return result
}

func inModule(importPath string, modulePath string) bool {
	return importPath == modulePath || strings.HasPrefix(importPath, modulePath+"/")
}
