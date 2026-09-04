package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphWithoutArguments(t *testing.T) {
	t.Chdir(fixtureModule)

	output := runScanCmd(t, "graph")

	// The main package of the fixture imports scan, and nothing imports it.
	assert.Equal(t, `graph LR
  p0["."]
  p1["scan"]
  p0 --> p1
`, output)
}

func TestGraphOfAPackage(t *testing.T) {
	t.Chdir(fixtureModule)

	output := runScanCmd(t, "graph", "./model")

	assert.Contains(t, output, `p0["gen"]`)
	assert.Contains(t, output, `p1["model"]`)
	assert.Contains(t, output, `p2["scan"]`)
	assert.Contains(t, output, "p0 --> p1")
	assert.Contains(t, output, "p2 --> p1")
}

func TestGraphWithDepthAndDirection(t *testing.T) {
	t.Chdir(fixtureModule)

	wholeApp := runScanCmd(t, "graph", ".", "--outgoing", "--depth", "0")

	assert.Equal(t, `graph LR
  p0["."]
  p1["model"]
  p2["scan"]
  p0 --> p2
  p2 --> p1
`, wholeApp, "main imports scan, which imports model")
}

func TestGraphWritesToAFile(t *testing.T) {
	t.Chdir(fixtureModule)
	file := filepath.Join(t.TempDir(), "graph.mmd")

	output := runScanCmd(t, "graph", ".", "--output", file)

	assert.Empty(t, output, "nothing goes to stdout")
	content, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Contains(t, string(content), "graph LR\n")
}

func TestGraphFailsForAPatternWithSeveralPackages(t *testing.T) {
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"graph", "./..."})
	t.Chdir(fixtureModule)

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "must match exactly one")
}

func TestGraphFailsForMoreThanOnePattern(t *testing.T) {
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"graph", ".", "./model"})
	t.Chdir(fixtureModule)

	require.Error(t, cmd.Execute())
}

func TestGraphFailsIfTheGraphCannotBeWritten(t *testing.T) {
	if _, err := os.Stat("/dev/full"); err != nil {
		t.Skip("/dev/full is not available")
	}
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"graph", ".", "--output", "/dev/full"})
	t.Chdir(fixtureModule)

	err := cmd.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no space left on device")
}

func TestGraphFailsForAnUnwritableOutputFile(t *testing.T) {
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"graph", ".", "--output", filepath.Join(t.TempDir(), "no", "such", "dir", "graph.mmd")})
	t.Chdir(fixtureModule)

	require.Error(t, cmd.Execute())
}
