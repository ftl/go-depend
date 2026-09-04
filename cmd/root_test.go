package cmd

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmdWithoutArgsPrintsHelp(t *testing.T) {
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{}) // nil would make cobra fall back to os.Args

	require.NoError(t, cmd.Execute())

	// The Usage: block appears as soon as the root command has subcommands.
	assert.Contains(t, out.String(), "go-depend calculates software package metrics")
}

func TestRootCmdWithUnknownFlagFails(t *testing.T) {
	cmd := newRootCmd()
	out := &bytes.Buffer{}
	cmd.SetOut(out)
	cmd.SetErr(out)
	cmd.SetArgs([]string{"--no-such-flag"})

	require.Error(t, cmd.Execute())
}
