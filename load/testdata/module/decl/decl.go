package decl

import "errors"

// Reader is an abstract interface.
type Reader interface {
	Read() string
}

// Cache is an abstract type: it has type parameters.
type Cache[K comparable, V any] struct {
	entries map[K]V
}

// Celsius is a concrete named type without a struct.
type Celsius float64

// Client is a concrete struct.
type Client struct {
	Name string
}

// Alias is a type alias and does not count.
type Alias = Client

// hidden is not exported and does not count.
type hidden struct {
	name string
}

// DefaultName is a constant and does not count.
const DefaultName = "client"

// ErrMissing is a variable and does not count.
var ErrMissing = errors.New("missing")

// New is a concrete function.
func New() *Client {
	return &Client{Name: DefaultName}
}

// Map is an abstract function: it has type parameters.
func Map[T any, U any](in []T, convert func(T) U) []U {
	result := make([]U, len(in))
	for i, value := range in {
		result[i] = convert(value)
	}
	return result
}

// Get is a method and does not count.
func (c *Client) Get() string {
	return c.Name
}

// Put is a method of a generic type and does not count.
func (c *Cache[K, V]) Put(key K, value V) {
	if c.entries == nil {
		c.entries = make(map[K]V)
	}
	c.entries[key] = value
}

func hiddenFunc(h hidden) string {
	return h.name
}
