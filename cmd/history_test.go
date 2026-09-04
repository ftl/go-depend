package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanWithHistory(t *testing.T) {
	t.Chdir(newModuleRepository(t))

	output := runScanCmd(t, "scan", "--history")

	// churn changed in all four months of the history, stable and the main
	// package only in the first one.
	assert.Regexp(t, `\nPACKAGE +CA +CE +STD +EXT +I +A +D +ZONE +SDP +PERCEIVED +DIFF\n`, "\n"+output)
	assert.Regexp(t, `\nchurn +1 +0 +0 +0 +0\.00 +0\.00 +1\.00 +PAIN +- +1\.00 +\+1\.00\n`, output)
	assert.Regexp(t, `\nstable +1 +0 +0 +0 +0\.00 +0\.00 +1\.00 +PAIN +- +0\.25 +\+0\.25\n`, output)
	assert.Regexp(t, `\n\. +0 +2 +0 +0 +1\.00 +0\.00 +0\.00 +- +- +0\.25 +-0\.75\n`, output)
}

func TestScanWithHistorySince(t *testing.T) {
	t.Chdir(newModuleRepository(t))

	output := runScanCmd(t, "scan", "--history", "--since", "2026-03-01")

	// Only march and april are left, and only churn changed in them.
	assert.Regexp(t, `\nchurn .* 1\.00 +\+1\.00\n`, output)
	assert.Regexp(t, `\nstable .* 0\.00 +\+0\.00\n`, output)
}

func TestScanWithoutHistory(t *testing.T) {
	t.Chdir(newModuleRepository(t))

	output := runScanCmd(t, "scan")

	assert.NotContains(t, output, "PERCEIVED")
}

func TestScanWithHistoryFailsOutsideOfARepository(t *testing.T) {
	t.Chdir(fixtureModule)

	root := newRootCmd()
	root.SetOut(&testWriter{})
	root.SetErr(&testWriter{})
	root.SetArgs([]string{"scan", "--history"})

	err := root.Execute()

	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot read the git history")
}

type testWriter struct{}

func (w *testWriter) Write(p []byte) (int, error) { return len(p), nil }

// newModuleRepository creates a directory that is a Go module and a git
// repository at the same time. churn changes in every month of the history,
// stable and the main package only in the first one.
func newModuleRepository(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	dir := t.TempDir()
	writeFile(t, dir, "go.mod", "module example.com/hist\n\ngo 1.26\n")
	writeFile(t, dir, "main.go", mainSource)
	writeFile(t, dir, "stable/stable.go", "package stable\n\n// Name is stable.\nconst Name = \"stable\"\n")
	writeFile(t, dir, "churn/churn.go", "package churn\n\n// Version is 1.\nconst Version = 1\n")
	writeFile(t, dir, "doc/notes.md", "notes\n")

	git(t, dir, "", "init", "-q", "-b", "main")
	git(t, dir, "", "add", "-A")
	commit(t, dir, "2026-01-05T10:00:00Z", "first")

	for i, date := range []string{"2026-02-05T10:00:00Z", "2026-03-05T10:00:00Z", "2026-04-05T10:00:00Z"} {
		writeFile(t, dir, "churn/churn.go", "package churn\n\n// Version is changed.\nconst Version = "+string(rune('2'+i))+"\n")
		commit(t, dir, date, "churn "+date)
	}

	return dir
}

const mainSource = `package main

import (
	"example.com/hist/churn"
	"example.com/hist/stable"
)

func main() {
	_ = churn.Version
	_ = stable.Name
}
`

// commit creates a commit with the given date as author and commit date.
func commit(t *testing.T, dir string, date string, message string) {
	t.Helper()
	git(t, dir, date, "commit", "-qam", message, "--date="+date)
}

// git runs one git command in the repository, without the configuration of
// the user: neither a hook nor a signing key must interfere.
func git(t *testing.T, dir string, commitDate string, args ...string) {
	t.Helper()

	command := exec.Command("git", args...)
	command.Dir = dir
	command.Env = append(os.Environ(),
		"GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_SYSTEM=/dev/null",
		"GIT_AUTHOR_NAME=go-depend",
		"GIT_AUTHOR_EMAIL=go-depend@example.com",
		"GIT_COMMITTER_NAME=go-depend",
		"GIT_COMMITTER_EMAIL=go-depend@example.com",
	)
	if commitDate != "" {
		command.Env = append(command.Env, "GIT_COMMITTER_DATE="+commitDate)
	}

	output, err := command.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, output)
}

func writeFile(t *testing.T, dir string, name string, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}
