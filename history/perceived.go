package history

import (
	"path"
	"strings"
	"time"

	"github.com/ftl/go-depend/model"
)

// PerceivedInstability returns the perceived instability of every given
// package, by import path. The value is the ratio of the months in which the
// package was changed to all months of the analyzed history, in the range 0
// to 1. A package that changes constantly reaches 1, a package that changed
// only during a short period of time stays close to 0.
//
// The analyzed history reaches from the oldest to the newest change of the
// given changes. It does not reach until today: otherwise the result of the
// reality check would depend on the day it runs.
func PerceivedInstability(changes []Change, packages []model.Package) map[string]float64 {
	result := make(map[string]float64, len(packages))
	for _, pkg := range packages {
		result[pkg.ImportPath] = 0
	}

	months := monthsOf(changes)
	if months == 0 {
		return result
	}

	for importPath, active := range activeMonths(changes, packages) {
		result[importPath] = float64(len(active)) / float64(months)
	}
	return result
}

// activeMonths returns the months in which each package was changed.
func activeMonths(changes []Change, packages []model.Package) map[string]map[int]bool {
	byDir := make(map[string]string, len(packages))
	for _, pkg := range packages {
		byDir[pkg.Dir] = pkg.ImportPath
	}

	result := make(map[string]map[int]bool)
	for _, change := range changes {
		if !isSourceFile(change.Path) {
			continue
		}
		importPath, ok := byDir[path.Dir(change.Path)]
		if !ok {
			// The file belongs to no analyzed package, for example because it
			// is part of the documentation.
			continue
		}

		if result[importPath] == nil {
			result[importPath] = make(map[int]bool)
		}
		result[importPath][monthOf(change.Time)] = true
	}
	return result
}

// isSourceFile reports whether a change of this file is a change of the
// package. Test files do not count, just as they do not count for the
// calculated metrics.
func isSourceFile(filePath string) bool {
	return strings.HasSuffix(filePath, ".go") && !strings.HasSuffix(filePath, "_test.go")
}

// monthsOf returns the number of months from the oldest to the newest change,
// both included.
func monthsOf(changes []Change) int {
	if len(changes) == 0 {
		return 0
	}

	oldest, newest := monthOf(changes[0].Time), monthOf(changes[0].Time)
	for _, change := range changes[1:] {
		month := monthOf(change.Time)
		oldest = min(oldest, month)
		newest = max(newest, month)
	}
	return newest - oldest + 1
}

// monthOf returns the month of the given time as a number that increases with
// every month, so that two months can be compared and counted.
func monthOf(t time.Time) int {
	return t.Year()*12 + int(t.Month()) - 1
}
