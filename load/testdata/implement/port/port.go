// Package port declares the interfaces that other packages implement. It
// imports none of them.
package port

// Corpus is a port with two methods.
type Corpus interface {
	Next() string
	Len() int
}

// Quitter has only one method, therefore another type can implement it by
// accident.
type Quitter interface {
	Quit()
}

// Internal is implemented inside this package.
type Internal interface {
	Do()
}

// Local implements Internal. Both are in the same package, which is no edge
// between two packages.
type Local struct{}

// Do does nothing.
func (Local) Do() {}

// Empty is implemented by every type and takes part in no edge.
type Empty interface{}

// Store is a generic port.
type Store[T any] interface {
	Put(value T)
}

// Handler is a function type: another package can put its own behaviour here.
type Handler func(name string) error
