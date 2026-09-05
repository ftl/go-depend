package model

// Zone is the position of a package in relation to the main sequence.
type Zone int

const (
	// MainSequence means that the package is close enough to the main
	// sequence, and violates no invariant.
	MainSequence Zone = iota
	// ZoneOfPain means that the package is too concrete for the number of
	// packages that depend on it.
	ZoneOfPain
	// ZoneOfUselessness means that the package is abstract, but almost no
	// other package depends on it.
	ZoneOfUselessness
)

// Metrics are the software package metrics of one package.
type Metrics struct {
	// Package is the package that these metrics belong to.
	Package Package

	// Afferent is the afferent coupling Ca: the number of packages from the
	// same module that import this package.
	Afferent int
	// Efferent is the efferent coupling Ce: the number of packages from the
	// same module that this package imports.
	Efferent int
	// Stdlib is the number of packages from the Go standard library that this
	// package imports.
	Stdlib int
	// External is the number of packages from other modules that this package
	// imports, including the modules of the same workspace.
	External int

	// Instability is I = Ce / (Ce + Ca), in the range 0 to 1.
	Instability float64
	// Abstractness is A = abstractions / declarations, in the range 0 to 1.
	Abstractness float64
	// SignedDistance is A + I - 1, in the range -1 to 1. A negative value
	// means that the package is on the side of the zone of pain, a positive
	// value that it is on the side of the zone of uselessness.
	SignedDistance float64
	// Distance is D = | A + I - 1 |, in the range 0 to 1.
	Distance float64

	// AbstractCoupling is the part of the dependent packages of the same
	// module that use this package through an abstraction: they implement one
	// of its interfaces, or they use one of its interfaces or function types.
	// The range is 0 to 1, and a package without any dependent has the value
	// 0.
	//
	// Martin's abstractness A asks a package how abstract it is. In Go an
	// interface belongs to the package that uses it, and a type implements it
	// without a reference, therefore A cannot see the answer. This value asks
	// the dependent packages instead.
	AbstractCoupling float64

	// PerceivedInstability is the ratio of the months in which the package
	// was changed to all months of the analyzed git history. It is nil if the
	// history was not read.
	//
	// It is deliberately not compared with Instability by a calculation: the
	// instability counts imports, the perceived instability counts months.
	// The reader compares the two values.
	PerceivedInstability *float64

	// Zone is the zone of the package. Any zone except the main sequence
	// violates the invariant about the zones of pain and uselessness.
	Zone Zone
	// UnstableDependencies contains the import paths of the packages from the
	// same module that this package imports although they are less stable
	// than this package itself. Each of them violates the invariant about
	// stable dependencies.
	UnstableDependencies []string
}

// HasViolation reports whether the package violates one of the invariants.
func (m Metrics) HasViolation() bool {
	return m.Zone != MainSequence || len(m.UnstableDependencies) > 0
}
