package changes

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// command is git in dir, with optional locks off, so that reading the
// repository never writes its index, which a watch would see as a change,
// with its messages in English, to parse, and with paths taken as they
// are written, rather than as patterns.
func command(dir string, args ...string) *exec.Cmd {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(cmd.Environ(), "GIT_OPTIONAL_LOCKS=0", "LC_ALL=C", "GIT_LITERAL_PATHSPECS=1")
	return cmd
}

// run runs git in dir and returns its standard output, or its error
// message. Its standard input is empty.
func run(dir string, args ...string) ([]byte, error) {
	cmd := command(dir, args...)
	cmd.Stdin = strings.NewReader("")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("git %s: %s", args[0], msg)
	}
	return out, nil
}

// line runs git and returns the first line of its output.
func line(dir string, args ...string) (string, error) {
	out, err := run(dir, args...)
	if err != nil {
		return "", err
	}
	first, _, _ := strings.Cut(string(out), "\n")
	return strings.TrimSpace(first), nil
}

// RepoRoot returns the top of the work tree holding dir, as the system
// writes paths: git writes C:/Users on Windows, where Go writes C:\Users.
func RepoRoot(dir string) (string, error) {
	root, err := line(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		if _, lookErr := exec.LookPath("git"); lookErr != nil {
			return "", errors.New("git is not installed")
		}
		return "", fmt.Errorf("%s is not inside a Git repository", dir)
	}
	return filepath.Clean(root), nil
}

// Revisions returns the commits revs name now, as one string to compare
// with what an earlier call returned: it changes as a commit moves any of
// them. It is "" when one names nothing, as HEAD before the first commit.
func Revisions(dir string, revs ...string) string {
	out, err := run(dir, append([]string{"rev-parse"}, revs...)...)
	if err != nil {
		return ""
	}
	return string(out)
}

// IgnoredDirs returns the directories of the work tree at dir that an
// ignore rule matches, as node_modules, rather than those holding nothing
// but what is ignored for now, by their paths with slashes.
func IgnoredDirs(dir string) map[string]bool {
	ignored := map[string]bool{}
	out, _ := run(dir, "status", "--porcelain=v1", "-z", "--ignored=matching", "--untracked-files=normal")
	for entry := range strings.SplitSeq(string(out), "\x00") {
		if p, ok := strings.CutPrefix(entry, "!! "); ok {
			if d, ok := strings.CutSuffix(p, "/"); ok {
				ignored[d] = true
			}
		}
	}
	return ignored
}

// TrackedDirs returns the directories Git tracks files in, by their paths
// with slashes.
func TrackedDirs(dir string) map[string]bool {
	tracked := map[string]bool{}
	out, _ := run(dir, "ls-files", "-z")
	for p := range strings.SplitSeq(string(out), "\x00") {
		for d := path.Dir(p); d != "." && !tracked[d]; d = path.Dir(d) {
			tracked[d] = true
		}
	}
	return tracked
}

// resolveCommit returns the commit rev names, or ok false when there is
// none, as HEAD before the first commit.
func resolveCommit(dir, rev string) (string, bool) {
	oid, err := line(dir, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
	return oid, err == nil && oid != ""
}

// emptyTree returns the ID of the empty tree in the repository's hash.
func emptyTree(dir string) (string, error) {
	return line(dir, "hash-object", "-t", "tree", "--stdin")
}

// mergeBase returns where a and b forked, as GitHub compares a pull
// request's branch with its base: what the base gained since then is not
// the branch's change. Histories with nothing in common compare with a.
func mergeBase(dir, a, b string) string {
	if oid, err := line(dir, "merge-base", a, b); err == nil && oid != "" {
		return oid
	}
	oid, _ := resolveCommit(dir, a)
	return oid
}

// currentBranch returns the branch checked out, or the short commit when
// HEAD is detached, "" before the first commit.
func currentBranch(dir string) string {
	if name, err := line(dir, "symbolic-ref", "--short", "-q", "HEAD"); err == nil && name != "" {
		return name
	}
	short, _ := line(dir, "rev-parse", "--short", "HEAD")
	return short
}

// nameStatus lists the files changed between old and new, a commit or ""
// for the work tree, renames found: those under paths, when given.
func nameStatus(dir, old, new string, paths []string) ([]Change, error) {
	args := []string{"-c", "core.quotepath=off", "diff", "--no-ext-diff", "--no-color", "--name-status", "-z", "-M", old}
	if new != "" {
		args = append(args, new)
	}
	args = append(append(args, "--"), paths...)
	out, err := run(dir, args...)
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out)
}

// parseNameStatus reads git diff --name-status -z: a status, then one
// path, or two for renames and copies, each ended by NUL.
func parseNameStatus(out []byte) ([]Change, error) {
	fields := strings.Split(strings.TrimSuffix(string(out), "\x00"), "\x00")
	var changes []Change
	for i := 0; i < len(fields); i++ {
		code := fields[i]
		if code == "" {
			continue
		}
		next := func() (string, error) {
			i++
			if i >= len(fields) {
				return "", fmt.Errorf("git diff: truncated output after %q", code)
			}
			return fields[i], nil
		}
		path, err := next()
		if err != nil {
			return nil, err
		}
		switch code[0] {
		case 'A':
			changes = append(changes, Change{Path: path, Status: StatusAdded})
		case 'D':
			changes = append(changes, Change{Path: path, Status: StatusDeleted})
		case 'R', 'C':
			to, err := next()
			if err != nil {
				return nil, err
			}
			if code[0] == 'C' { // a copy is a new file
				changes = append(changes, Change{Path: to, Status: StatusAdded})
			} else {
				changes = append(changes, Change{Path: to, OldPath: path, Status: StatusRenamed})
			}
		default: // M, T (type changed), U (unmerged)
			changes = append(changes, Change{Path: path, Status: StatusModified})
		}
	}
	return changes, nil
}

// untracked lists the files in the work tree Git doesn't track and
// doesn't ignore, new files an agent wrote: those under paths, when given.
func untracked(dir string, paths []string) ([]string, error) {
	args := []string{"-c", "core.quotepath=off", "ls-files", "--others", "--exclude-standard", "-z", "--"}
	out, err := run(dir, append(args, paths...)...)
	if err != nil {
		return nil, err
	}
	var files []string
	for p := range strings.SplitSeq(string(out), "\x00") {
		if p != "" {
			files = append(files, p)
		}
	}
	return files, nil
}

// catFiles reads the objects specs name, as "<commit>:<path>", in one
// git process. A missing object reads as nil.
func catFiles(dir string, specs []string) ([][]byte, error) {
	texts := make([][]byte, len(specs))
	if len(specs) == 0 {
		return texts, nil
	}
	cmd := command(dir, "cat-file", "--batch")
	cmd.Stdin = strings.NewReader(strings.Join(specs, "\n") + "\n")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("git cat-file: %w", err)
	}
	r := bufio.NewReaderSize(stdout, 64<<10)
	for i := range specs {
		text, err := readObject(r)
		if err != nil {
			cmd.Process.Kill()
			cmd.Wait()
			return nil, fmt.Errorf("git cat-file: %w", err)
		}
		texts[i] = text
	}
	return texts, cmd.Wait()
}

// readObject reads an object git cat-file --batch prints: a header of its
// ID, type and size, then its content, nil for a missing one. A
// submodule's commit stands for its text.
func readObject(r *bufio.Reader) ([]byte, error) {
	header, err := r.ReadString('\n')
	if err != nil {
		return nil, err
	}
	header = strings.TrimSuffix(header, "\n")
	if strings.HasSuffix(header, " missing") || strings.HasSuffix(header, " ambiguous") {
		return nil, nil
	}
	parts := strings.Fields(header)
	if len(parts) != 3 {
		return nil, fmt.Errorf("unexpected header %q", header)
	}
	size, err := strconv.Atoi(parts[2])
	if err != nil {
		return nil, fmt.Errorf("unexpected header %q", header)
	}
	buf := make([]byte, size+1) // the object and its newline
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	switch parts[1] {
	case "blob":
		return buf[:size], nil
	case "commit":
		return []byte("Subproject commit " + parts[0] + "\n"), nil
	}
	return nil, nil
}
