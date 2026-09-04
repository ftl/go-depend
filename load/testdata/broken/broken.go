package broken

import "example.com/does/not/exist"

// Broken uses a package that cannot be resolved.
func Broken() string {
	return exist.Name
}
