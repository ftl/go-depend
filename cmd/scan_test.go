package cmd

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixtureModule is the module that the load package uses as testdata.
const fixtureModule = "../load/testdata/module"

func TestScanFixtureModule(t *testing.T) {
	t.Chdir(fixtureModule)

	output := runScanCmd(t, "scan")

	assert.Contains(t, output, "PACKAGE  CA  CE  STD  EXT  I     A     D     ZONE  SDP")
	assert.Contains(t, output, "\ndecl ")
	assert.Contains(t, output, "\nmodel ")
	assert.NotContains(t, output, "MODULE", "a single module needs no module column")
}

func TestScanCalculatesTheMetrics(t *testing.T) {
	t.Chdir(fixtureModule)

	output := runScanCmd(t, "scan")

	// decl: 4 types and 2 funcs, 3 of them abstract, imported by nobody,
	// imports only the standard library.
	assert.Regexp(t, `\ndecl +0 +0 +1 +0 +1\.00 +0\.50 +0\.50 +- +-\n`, output)
	// model: imported by scan, gen and decl is not among them.
	assert.Regexp(t, `\nmodel +2 +0 +0 +0 +0\.00 +0\.00 +1\.00 +PAIN +-\n`, output)
}

func TestScanExcludesFiles(t *testing.T) {
	t.Chdir(fixtureModule)

	withGenerated := runScanCmd(t, "scan")
	assert.Regexp(t, `\ngen +0 +1 +1 +`, withGenerated, "gen.pb.go imports encoding/json")

	withoutGenerated := runScanCmd(t, "scan", "--exclude", `\.pb\.go$`)
	assert.Regexp(t, `\ngen +0 +1 +0 +`, withoutGenerated, "the import of the excluded file is gone")
}

func TestScanWithMaxDistance(t *testing.T) {
	t.Chdir(fixtureModule)

	strict := runScanCmd(t, "scan", "--max-distance", "0.1")

	assert.Contains(t, strict, "zone of pain")
	assert.Regexp(t, `\ndecl +0 +0 +1 +0 +1\.00 +0\.50 +0\.50 +USELESS`, strict,
		"decl is abstract, and no package imports it")
}

func TestScanAcceptsAPattern(t *testing.T) {
	t.Chdir(fixtureModule)

	output := runScanCmd(t, "scan", "./model")

	assert.Contains(t, output, "\nmodel ")
	assert.NotContains(t, output, "\ndecl ")
}

func TestScanWithJSONFormat(t *testing.T) {
	t.Chdir(fixtureModule)

	output := runScanCmd(t, "scan", "--format", "json")

	var parsed struct {
		Packages []struct {
			ImportPath string `json:"importPath"`
			Types      int    `json:"types"`
			Abstract   int    `json:"abstract"`
		} `json:"packages"`
	}
	require.NoError(t, json.Unmarshal([]byte(output), &parsed))

	require.Len(t, parsed.Packages, 6)
	assert.Equal(t, "example.com/fixture", parsed.Packages[0].ImportPath)
	for _, pkg := range parsed.Packages {
		if pkg.ImportPath == "example.com/fixture/decl" {
			assert.Equal(t, 4, pkg.Types)
			assert.Equal(t, 3, pkg.Abstract)
			return
		}
	}
	assert.Fail(t, "the package decl is missing")
}

func TestScanWithCSVFormat(t *testing.T) {
	t.Chdir(fixtureModule)

	output := runScanCmd(t, "scan", "--format", "csv")

	records, err := csv.NewReader(strings.NewReader(output)).ReadAll()
	require.NoError(t, err)

	require.Len(t, records, 7, "the header and six packages")
	assert.Equal(t, "module", records[0][0])
	assert.Equal(t, "example.com/fixture", records[1][1])
}

func TestScanWithoutFailOnViolationSucceeds(t *testing.T) {
	t.Chdir(fixtureModule)

	// The fixture module contains a package in the zone of pain, but without
	// the flag this is data and not a failure.
	output := runScanCmd(t, "scan")

	assert.Contains(t, output, "PAIN")
}

func TestScanFailsOnViolation(t *testing.T) {
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"scan", "--fail-on-violation"})
	t.Chdir(fixtureModule)

	err := cmd.Execute()

	require.ErrorIs(t, err, ErrViolations)
	assert.Contains(t, out.String(), "PAIN", "the report is written before the error")
}

func TestScanWithoutViolationsSucceedsWithFailOnViolation(t *testing.T) {
	t.Chdir(fixtureModule)

	// With the maximum distance no package of the fixture is in a zone, and
	// the fixture violates the stable dependencies invariant nowhere.
	output := runScanCmd(t, "scan", "--fail-on-violation", "--max-distance", "1.0")

	assert.NotContains(t, output, "PAIN")
	assert.NotContains(t, output, "Violations:")
}

func TestScanFailsForAnUnknownFormat(t *testing.T) {
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"scan", "--format", "xml"})
	t.Chdir(fixtureModule)

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), `unknown format "xml"`)
	assert.NotContains(t, out.String(), "Usage:", "the usage does not help with this error")
}

func TestScanFailsForAnUnknownPattern(t *testing.T) {
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"scan", "./does-not-exist/..."})
	t.Chdir(fixtureModule)

	require.Error(t, cmd.Execute())
}

func TestScanFailsForAnInvalidExcludeExpression(t *testing.T) {
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"scan", "--exclude", "("})
	t.Chdir(fixtureModule)

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid exclude expression")
}

// runScanCmd executes go-depend in the current working directory. The caller
// changes into the directory of the module under test once per test, because
// t.Chdir resolves a relative path against the current working directory.
func runScanCmd(t *testing.T, args ...string) string {
	t.Helper()

	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs(args)
	require.NoError(t, cmd.Execute())

	return out.String()
}
