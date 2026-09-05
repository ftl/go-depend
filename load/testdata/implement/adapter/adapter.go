// Package adapter implements the ports. It imports the package port nowhere:
// in Go a type satisfies an interface without a reference to it.
package adapter

// Ring implements port.Corpus with value receivers.
type Ring struct{}

// Next returns the next entry.
func (Ring) Next() string { return "ring" }

// Len returns the number of entries.
func (Ring) Len() int { return 1 }

// Random implements port.Corpus with pointer receivers.
type Random struct {
	seed int
}

// Next returns the next entry.
func (r *Random) Next() string { return "random" }

// Len returns the number of entries.
func (r *Random) Len() int { return r.seed }

// Printer implements port.Quitter by accident: it quits its own output, and
// it knows nothing about the package port.
type Printer struct{}

// Quit stops the printer.
func (Printer) Quit() {}

// hidden implements port.Corpus, but it is not exported and is no part of
// the contract of this package.
type hidden struct{}

func (hidden) Next() string { return "hidden" }

func (hidden) Len() int { return 0 }

// Generic implements port.Corpus, but it has a type parameter. Only an
// instantiated type has a method set that can be compared with an interface.
type Generic[T any] struct {
	value T
}

// Next returns the next entry.
func (Generic[T]) Next() string { return "generic" }

// Len returns the number of entries.
func (Generic[T]) Len() int { return 0 }

// Box implements port.Store[string].
type Box struct{}

// Put stores a value.
func (Box) Put(value string) {}

// GenBox implements port.Store[T] for every instantiation.
type GenBox[T any] struct{}

// Put stores a value.
func (GenBox[T]) Put(value T) {}

// RingAlias is an alias and declares no new type, therefore it is no
// additional implementation of port.Corpus.
type RingAlias = Ring

// Closer has all methods of port.Quitter and one more. An interface is no
// implementation of another interface.
type Closer interface {
	Quit()
	Close() error
}

// hiddenStore implements port.Store[string], but it is not exported and is no
// part of the contract of this package.
type hiddenStore[T any] struct{}

func (hiddenStore[T]) Put(value T) {}

// NewHiddenStore instantiates the unexported generic type.
func NewHiddenStore() any { return hiddenStore[string]{} }
