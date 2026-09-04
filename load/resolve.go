package load

import (
	"fmt"

	"golang.org/x/tools/go/packages"
)

// resolveMode reads only the identity of a package, and neither its files nor
// its source code.
const resolveMode = packages.NeedName | packages.NeedModule

// ResolveOne resolves a pattern to the import path of exactly one package.
// The pattern may also address a package outside of the module, as long as the
// module depends on it.
func ResolveOne(options Options, pattern string) (string, error) {
	config := &packages.Config{Mode: resolveMode, Dir: options.Dir, Tests: false}
	pkgs, err := packages.Load(config, pattern)
	if err != nil {
		return "", fmt.Errorf("cannot resolve %s: %w", pattern, err)
	}
	if err := loadErrors(pkgs); err != nil {
		return "", err
	}
	if len(pkgs) != 1 {
		return "", fmt.Errorf("the pattern %s matches %d packages, but it must match exactly one", pattern, len(pkgs))
	}

	return pkgs[0].PkgPath, nil
}
