package history_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/ftl/go-depend/history"
	"github.com/ftl/go-depend/model"
)

const module = "example.com/module"

func TestPerceivedInstability(t *testing.T) {
	packages := []model.Package{
		{ImportPath: module, ModulePath: module, Dir: "."},
		{ImportPath: module + "/a", ModulePath: module, Dir: "a"},
		{ImportPath: module + "/b", ModulePath: module, Dir: "b"},
	}
	changes := []history.Change{
		{Time: month(2026, 4), Path: "a/a.go"},
		{Time: month(2026, 4), Path: "doc/readme.md"},
		{Time: month(2026, 3), Path: "a/a.go"},
		{Time: month(2026, 2), Path: "a/a.go"},
		{Time: month(2026, 2), Path: "b/b_test.go"},
		{Time: month(2026, 1), Path: "a/a.go"},
		{Time: month(2026, 1), Path: "b/b.go"},
		{Time: month(2026, 1), Path: "b/b.go"},
	}

	perceived := history.PerceivedInstability(changes, packages)

	assert.InDelta(t, 1.0, perceived[module+"/a"], 1e-9, "changed in all four months")
	assert.InDelta(t, 0.25, perceived[module+"/b"], 1e-9, "changed in one of four months, the test file does not count")
	assert.InDelta(t, 0.0, perceived[module], 1e-9, "never changed")
}

func TestPerceivedInstabilityOfABurst(t *testing.T) {
	packages := []model.Package{{ImportPath: module + "/a", ModulePath: module, Dir: "a"}}
	changes := []history.Change{
		{Time: month(2026, 12), Path: "doc/readme.md"},
		{Time: month(2026, 1), Path: "a/a.go"},
		{Time: month(2026, 1), Path: "a/other.go"},
	}

	perceived := history.PerceivedInstability(changes, packages)

	assert.InDelta(t, 1.0/12.0, perceived[module+"/a"], 1e-9, "one active month of twelve")
}

func TestPerceivedInstabilityAcrossAYear(t *testing.T) {
	packages := []model.Package{{ImportPath: module + "/a", ModulePath: module, Dir: "a"}}
	changes := []history.Change{
		{Time: month(2026, 1), Path: "a/a.go"},
		{Time: month(2025, 12), Path: "a/a.go"},
	}

	perceived := history.PerceivedInstability(changes, packages)

	assert.InDelta(t, 1.0, perceived[module+"/a"], 1e-9, "december and january are two of two months")
}

func TestPerceivedInstabilityOfTheRootPackage(t *testing.T) {
	packages := []model.Package{{ImportPath: module, ModulePath: module, Dir: "."}}
	changes := []history.Change{{Time: month(2026, 1), Path: "main.go"}}

	perceived := history.PerceivedInstability(changes, packages)

	assert.InDelta(t, 1.0, perceived[module], 1e-9)
}

func TestPerceivedInstabilityIgnoresFilesOfOtherDirectories(t *testing.T) {
	packages := []model.Package{{ImportPath: module + "/a", ModulePath: module, Dir: "a"}}
	changes := []history.Change{
		{Time: month(2026, 1), Path: "a/a.go"},
		{Time: month(2026, 1), Path: "a/nested/deep.go"},
		{Time: month(2026, 1), Path: "a/notes.txt"},
	}

	perceived := history.PerceivedInstability(changes, packages)

	assert.InDelta(t, 1.0, perceived[module+"/a"], 1e-9, "a file of a subdirectory belongs to another package")
}

func TestPerceivedInstabilityWithoutChanges(t *testing.T) {
	packages := []model.Package{{ImportPath: module + "/a", ModulePath: module, Dir: "a"}}

	perceived := history.PerceivedInstability(nil, packages)

	assert.Equal(t, map[string]float64{module + "/a": 0}, perceived)
}

func TestPerceivedInstabilityWithoutPackages(t *testing.T) {
	changes := []history.Change{{Time: month(2026, 1), Path: "a/a.go"}}

	assert.Empty(t, history.PerceivedInstability(changes, nil))
}

func month(year int, m time.Month) time.Time {
	return time.Date(year, m, 15, 12, 0, 0, 0, time.UTC)
}
