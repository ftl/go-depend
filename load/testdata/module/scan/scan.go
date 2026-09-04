package scan

import (
	"fmt"

	"example.com/fixture/model"
)

// Name describes the fixture.
const Name = "fixture"

// Describe returns a description of the given package.
func Describe(pkg model.Package) string {
	return fmt.Sprintf("%s", pkg.Name)
}
