package history_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ftl/go-depend/history"
)

// gitLog is the output of
//
//	git log --name-status -M --pretty=tformat:%x00%at -- .
//
// captured from a repository with a merge commit, a chain of two renames and
// a deleted file. The log begins with the newest commit, and %ct is the
// commit date.
const gitLog = "\x001778580000\n" + // the merge commit, without any file
	"\x001778493600\n" +
	"\n" +
	"A\td.go\n" +
	"\x001778407200\n" +
	"\n" +
	"M\tc/renamed.go\n" +
	"\x001775815200\n" +
	"\n" +
	"D\tb/b.go\n" +
	"\x001773136800\n" +
	"\n" +
	"R100\ta/renamed.go\tc/renamed.go\n" +
	"\x001770717600\n" +
	"\n" +
	"R100\ta/a.go\ta/renamed.go\n" +
	"\x001768039200\n" +
	"\n" +
	"A\ta/a.go\n" +
	"A\tb/b.go\n"

func TestParse(t *testing.T) {
	changes, err := history.Parse(strings.NewReader(gitLog))
	require.NoError(t, err)

	assert.Equal(t, []history.Change{
		{Time: at(1778493600), Path: "d.go"},
		{Time: at(1778407200), Path: "c/renamed.go"},
		{Time: at(1775815200), Path: "b/b.go"},
		// The two renames and the change before them all belong to the file
		// that is called c/renamed.go today.
		{Time: at(1773136800), Path: "c/renamed.go"},
		{Time: at(1770717600), Path: "c/renamed.go"},
		{Time: at(1768039200), Path: "c/renamed.go"},
		{Time: at(1768039200), Path: "b/b.go"},
	}, changes)
}

func TestParseIgnoresACommitWithoutFiles(t *testing.T) {
	changes, err := history.Parse(strings.NewReader(gitLog))
	require.NoError(t, err)

	for _, change := range changes {
		assert.NotEqual(t, at(1778580000), change.Time, "the merge commit changes no file")
	}
}

func TestParseCopy(t *testing.T) {
	// A copy changes only the new file, the source stays as it is.
	log := "\x001768039200\n\nC100\ta/a.go\tb/b.go\n\x001768000000\n\nM\ta/a.go\n"

	changes, err := history.Parse(strings.NewReader(log))
	require.NoError(t, err)

	assert.Equal(t, []history.Change{
		{Time: at(1768039200), Path: "b/b.go"},
		{Time: at(1768000000), Path: "a/a.go"},
	}, changes)
}

func TestParseEmptyLog(t *testing.T) {
	changes, err := history.Parse(strings.NewReader(""))

	require.NoError(t, err)
	assert.Empty(t, changes)
}

func TestParseFailsForAFileBeforeTheFirstCommit(t *testing.T) {
	_, err := history.Parse(strings.NewReader("M\ta/a.go\n"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "before the first commit")
}

func TestParseFailsForAnInvalidTimestamp(t *testing.T) {
	_, err := history.Parse(strings.NewReader("\x00not-a-timestamp\n"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid commit timestamp")
}

func TestParseFailsForAnInvalidStatusLine(t *testing.T) {
	_, err := history.Parse(strings.NewReader("\x001768039200\n\nM a/a.go\n"))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid status line")
}

func at(seconds int64) time.Time {
	return time.Unix(seconds, 0).UTC()
}
