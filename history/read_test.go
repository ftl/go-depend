package history_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ftl/go-depend/history"
)

func TestReadRepository(t *testing.T) {
	dir := newRepository(t)

	changes, err := history.Read(history.Options{Dir: dir})
	require.NoError(t, err)

	assert.Equal(t, []history.Change{
		// The last commit was written earlier than it was committed, and the
		// commit date counts.
		{Time: date(t, "2026-03-05T10:00:00Z"), Path: "b/moved.go"},
		{Time: date(t, "2026-02-05T10:00:00Z"), Path: "b/moved.go"},
		{Time: date(t, "2026-01-05T10:00:00Z"), Path: "b/moved.go"},
		{Time: date(t, "2026-01-05T10:00:00Z"), Path: "keep.go"},
	}, changes, "the renames of a/a.go are resolved to its current path")
}

func TestReadSince(t *testing.T) {
	dir := newRepository(t)

	changes, err := history.Read(history.Options{Dir: dir, Since: "2026-02-01"})
	require.NoError(t, err)

	assert.Equal(t, []history.Change{
		{Time: date(t, "2026-03-05T10:00:00Z"), Path: "b/moved.go"},
		{Time: date(t, "2026-02-05T10:00:00Z"), Path: "b/moved.go"},
	}, changes)
}

func TestReadModuleInASubdirectory(t *testing.T) {
	// The module is not the root of the repository. The paths of the changes
	// must be relative to the module, not to the repository.
	dir := t.TempDir()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}
	git(t, dir, "", "init", "-q", "-b", "main")
	writeFile(t, dir, "sub/pkg/p.go", "package pkg\n")
	writeFile(t, dir, "outside.go", "package outside\n")
	git(t, dir, "", "add", "-A")
	commit(t, dir, "2026-01-05T10:00:00Z", "2026-01-05T10:00:00Z", "first")

	changes, err := history.Read(history.Options{Dir: filepath.Join(dir, "sub")})
	require.NoError(t, err)

	assert.Equal(t, []history.Change{
		{Time: date(t, "2026-01-05T10:00:00Z"), Path: "pkg/p.go"},
	}, changes, "the file outside of the module is not part of the result")
}

func TestReadFailsOutsideOfARepository(t *testing.T) {
	_, err := history.Read(history.Options{Dir: t.TempDir()})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not a git repository")
}

// newRepository creates a repository with three commits: one that adds two
// files, one that renames a file, and one that moves the renamed file into
// another directory.
func newRepository(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not available")
	}

	dir := t.TempDir()
	git(t, dir, "", "init", "-q", "-b", "main")
	// Rename detection is the default of git, but a user can switch it off.
	// go-depend passes -M and does not depend on the configuration.
	git(t, dir, "", "config", "diff.renames", "false")

	writeFile(t, dir, "a/a.go", "package a\n")
	writeFile(t, dir, "keep.go", "package keep\n")
	git(t, dir, "", "add", "-A")
	commit(t, dir, "2026-01-05T10:00:00Z", "2026-01-05T10:00:00Z", "first")

	git(t, dir, "", "mv", "a/a.go", "a/renamed.go")
	commit(t, dir, "2026-02-05T10:00:00Z", "2026-02-05T10:00:00Z", "rename")

	require.NoError(t, os.MkdirAll(filepath.Join(dir, "b"), 0o755))
	git(t, dir, "", "mv", "a/renamed.go", "b/moved.go")
	// The author date is much earlier than the commit date, as it is after a
	// rebase or a cherry-pick.
	commit(t, dir, "2025-12-01T10:00:00Z", "2026-03-05T10:00:00Z", "move")

	return dir
}

// commit creates a commit with the given author and commit date.
func commit(t *testing.T, dir string, authorDate string, commitDate string, message string) {
	t.Helper()
	git(t, dir, commitDate, "-c", "user.name=go-depend", "-c", "user.email=go-depend@example.com",
		"commit", "-qam", message, "--date="+authorDate)
}

// git runs one git command in the repository. The configuration of the user
// is ignored, so that neither a hook nor a signing key interferes.
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

func date(t *testing.T, value string) time.Time {
	t.Helper()
	result, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return result.UTC()
}
