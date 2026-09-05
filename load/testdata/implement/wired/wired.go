// Package wired implements a port, and it imports the package port as well.
package wired

import "example.com/impl/port"

// Service implements port.Quitter, and this package knows the port.
type Service struct{}

// Quit stops the service.
func (Service) Quit() {}

// Use takes a port as its parameter, and it names the same interface a second
// time. Every symbol counts one time only.
func Use(corpus port.Corpus) int {
	var other port.Corpus
	if other != nil {
		return other.Len()
	}
	return corpus.Len()
}

// Register takes a function type, which is an abstraction as well.
func Register(handler port.Handler) error {
	return handler("wired")
}

// UseStore names the instantiation port.Store[string]. Only an instantiation
// that the module really uses can be implemented by a type.
func UseStore(store port.Store[string]) {
	store.Put("stored")
}
