// Package metrics calculates the software package metrics of the packages of a
// model graph. It only does arithmetic and has no side effects.
package metrics

import (
	"math"

	"github.com/ftl/go-depend/model"
)

// Of returns the metrics of all packages of the graph, in the same order as
// model.Graph.Packages. A package with a distance from the main sequence of
// more than maxDistance is in the zone of pain or in the zone of uselessness.
//
// The metrics of a package are only complete if the graph contains its whole
// module: the afferent coupling and the invariant about stable dependencies
// both depend on packages that a narrow pattern may leave out.
func Of(graph *model.Graph, maxDistance float64) []model.Metrics {
	result := couplingMetrics(graph)
	instabilities := instabilitiesOf(result)

	for i := range result {
		result[i].Zone = zoneOf(result[i], maxDistance)
		result[i].UnstableDependencies = unstableDependencies(graph, result[i], instabilities)
	}

	return result
}

func couplingMetrics(graph *model.Graph) []model.Metrics {
	packages := graph.Packages()
	result := make([]model.Metrics, 0, len(packages))
	for _, pkg := range packages {
		result = append(result, ofPackage(graph, pkg))
	}
	return result
}

func instabilitiesOf(all []model.Metrics) map[string]float64 {
	result := make(map[string]float64, len(all))
	for _, m := range all {
		result[m.Package.ImportPath] = m.Instability
	}
	return result
}

// zoneOf returns the zone of the package. The absolute distance decides
// whether the package violates the invariant, and the sign of the distance
// decides which of the two zones it is in.
func zoneOf(m model.Metrics, maxDistance float64) model.Zone {
	switch {
	case m.Distance <= maxDistance:
		return model.MainSequence
	case m.SignedDistance < 0:
		return model.ZoneOfPain
	default:
		return model.ZoneOfUselessness
	}
}

// unstableDependencies returns the imported packages of the same module that
// are less stable than the importing package. Stable dependencies means that
// the instability decreases in the direction of the imports.
func unstableDependencies(graph *model.Graph, m model.Metrics, instabilities map[string]float64) []string {
	var result []string
	for _, imp := range graph.Outgoing(m.Package.ImportPath) {
		if imp.Kind != model.SameModule {
			continue
		}
		if instabilities[imp.To] > m.Instability {
			result = append(result, imp.To)
		}
	}
	return result
}

func ofPackage(graph *model.Graph, pkg model.Package) model.Metrics {
	result := model.Metrics{
		Package:  pkg,
		Afferent: countSameModule(graph.Incoming(pkg.ImportPath)),
	}
	result.Efferent, result.Stdlib, result.External = countOutgoing(graph.Outgoing(pkg.ImportPath))

	result.Instability = instability(result.Efferent, result.Afferent)
	result.Abstractness = abstractness(pkg)
	result.SignedDistance = result.Abstractness + result.Instability - 1
	result.Distance = math.Abs(result.SignedDistance)

	return result
}

// countSameModule counts the imports that stay inside the module. Only these
// imports contribute to the coupling metrics.
func countSameModule(imports []model.Import) int {
	result := 0
	for _, imp := range imports {
		if imp.Kind == model.SameModule {
			result++
		}
	}
	return result
}

// countOutgoing counts the outgoing imports per group: the packages of the
// same module, the packages of the standard library, and the packages of all
// other modules.
func countOutgoing(imports []model.Import) (sameModule int, stdlib int, external int) {
	for _, imp := range imports {
		switch imp.Kind {
		case model.SameModule:
			sameModule++
		case model.Stdlib:
			stdlib++
		case model.External, model.WorkspaceSibling:
			external++
		}
	}
	return sameModule, stdlib, external
}

// instability returns I = Ce / (Ce + Ca). A package without any coupling
// inside its module is completely unstable: no other package of the module
// depends on it, therefore it can change freely.
func instability(efferent int, afferent int) float64 {
	total := efferent + afferent
	if total == 0 {
		return 1
	}
	return float64(efferent) / float64(total)
}

// abstractness returns A = abstractions / declarations. A package without any
// exported declarations is completely concrete: it provides no abstraction
// that another package can depend on.
func abstractness(pkg model.Package) float64 {
	declarations := pkg.Types + pkg.Funcs
	if declarations == 0 {
		return 0
	}
	return float64(pkg.Abstract) / float64(declarations)
}
