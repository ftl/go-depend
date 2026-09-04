// Package history evaluates the git history of a module to find out which
// files were changed when.
package history

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// commitPrefix marks the beginning of a commit in the output of git log. A
// NUL character cannot occur in a line that describes a changed file.
const commitPrefix = "\x00"

// logFormat writes one line per commit that contains only the commit date as
// a unix timestamp. The commit date, and not the author date, is the date
// that git log --since compares against: with any other date the limit of the
// history and the dates of the changes would not agree.
const logFormat = "tformat:%x00%ct"

// Change is one changed file in one commit.
type Change struct {
	// Time is the commit date, which is the date that git log --since
	// compares against. It differs from the author date after a rebase or a
	// cherry-pick, and it says when the change entered this repository.
	Time time.Time
	// Path is the path of the file, relative to the root of the module and
	// separated by slashes. Renames are already resolved: the path is the one
	// that the file has today.
	Path string
}

// Options control which part of the history is read.
type Options struct {
	// Dir is the directory to run git in. It should be the root directory of
	// the module.
	Dir string
	// Since limits the history to the commits after this point in time. The
	// value is passed to git log as it is, for example "12 months ago" or
	// "2026-01-01". An empty string means the full history.
	Since string
}

// Read runs git log and returns all changes of the files below the directory
// of the options, the newest change first.
func Read(options Options) ([]Change, error) {
	command := exec.Command("git", logArgs(options.Since)...)
	command.Dir = options.Dir

	output, err := command.Output()
	if err != nil {
		return nil, gitError(err)
	}

	return Parse(strings.NewReader(string(output)))
}

func logArgs(since string) []string {
	// --relative makes git print the paths relative to the current directory,
	// which is the root of the module. Without it the paths would be relative
	// to the root of the repository, and a module in a subdirectory of a
	// repository would not match any package.
	result := []string{"log", "--name-status", "-M", "--relative", "--pretty=" + logFormat}
	if since != "" {
		result = append(result, "--since="+since)
	}
	// Only the files below the current directory belong to the module.
	return append(result, "--", ".")
}

func gitError(err error) error {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && len(exitError.Stderr) > 0 {
		return fmt.Errorf("cannot read the git history: %s", strings.TrimSpace(string(exitError.Stderr)))
	}
	return fmt.Errorf("cannot read the git history: %w", err)
}

// Parse reads the output of git log --name-status and returns the changes in
// the order of the log, the newest change first.
//
// A path that contains a tab or a quote is not supported: git quotes such a
// path, and Parse uses it as it is.
func Parse(r io.Reader) ([]Change, error) {
	parser := &parser{renames: make(map[string]string)}

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		if err := parser.line(scanner.Text()); err != nil {
			return nil, err
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("cannot read the git history: %w", err)
	}

	return parser.changes, nil
}

// parser holds the state while the output of git log is read. The log begins
// with the newest commit, therefore a rename is found before the commits that
// still used the old path.
type parser struct {
	changes []Change
	// renames maps a path that an older commit used to the path that the file
	// has today.
	renames map[string]string
	// current is the commit date of the commit that is being read.
	current time.Time
	// started reports whether the first commit was read.
	started bool
}

func (p *parser) line(line string) error {
	if after, found := strings.CutPrefix(line, commitPrefix); found {
		return p.commit(after)
	}
	if strings.TrimSpace(line) == "" {
		return nil
	}
	return p.file(line)
}

func (p *parser) commit(timestamp string) error {
	seconds, err := strconv.ParseInt(strings.TrimSpace(timestamp), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid commit timestamp %q: %w", timestamp, err)
	}

	p.current = time.Unix(seconds, 0).UTC()
	p.started = true
	return nil
}

func (p *parser) file(line string) error {
	if !p.started {
		return fmt.Errorf("changed file %q before the first commit", line)
	}

	fields := strings.Split(line, "\t")
	if len(fields) < 2 {
		return fmt.Errorf("invalid status line %q", line)
	}
	status, paths := fields[0], fields[1:]

	// A rename or a copy names the old and the new path. Only a rename means
	// that the older commits changed the same file under its old path.
	if isRenameOrCopy(status) && len(paths) >= 2 {
		p.add(paths[1])
		if strings.HasPrefix(status, "R") {
			p.renames[paths[0]] = p.currentPath(paths[1])
		}
		return nil
	}

	p.add(paths[0])
	return nil
}

func isRenameOrCopy(status string) bool {
	return strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C")
}

func (p *parser) add(path string) {
	p.changes = append(p.changes, Change{Time: p.current, Path: p.currentPath(path)})
}

// currentPath returns the path that the given file has today.
func (p *parser) currentPath(path string) string {
	if current, ok := p.renames[path]; ok {
		return current
	}
	return path
}
