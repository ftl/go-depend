package a

import (
	"fmt"

	"example.com/b/lib"
)

// Greet returns a greeting.
func Greet() string {
	return fmt.Sprintf("hello %s", lib.Name)
}
