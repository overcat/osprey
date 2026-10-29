package changes

import (
	"cmp"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoRepository is what Open returns when it is given neither a
// directory nor arguments outside a repository, as a launch from the
// Dock: there is nothing to compare until the user picks a folder.
var ErrNoRepository = errors.New("not in a repository")

// A UsageError is a command line Open can't make sense of. Its message says
// why, and how to write it instead.
type UsageError string

func (e UsageError) Error() string { return string(e) }

func usage(format string, args ...any) error { return UsageError(fmt.Sprintf(format, args...)) }

// Open finds what a command line compares, as osp's help describes it.
// args are branches or commits, then files or folders to keep the changes
// to, which "--" marks as paths where a branch has their name; or two
// files or two folders to compare with each other. The repository is the
// one at dir, or the one holding the paths given alone, wherever it is,
// and relative paths are found from dir, the current directory when "",
// as git -C has them. With committed, a branch's changes leave out those
// not committed yet; alone, it compares the branch with the one it most
// likely left.
func Open(dir string, args []string, committed bool) (Source, error) {
	given := dir != ""
	if !given {
		dir, _ = os.Getwd()
	}
	dir, _ = filepath.Abs(dir)
	abs := func(p string) string {
		if filepath.IsAbs(p) {
			return filepath.Clean(p)
		}
		return filepath.Join(dir, p)
	}
	root, rootErr := RepoRoot(dir)
	isRef := func(r string) bool {
		if rootErr != nil {
			return false
		}
		_, ok := resolveCommit(root, r)
		return ok
	}

	// The branches and commits come first, then the paths, as git has them.
	var revs, paths []string
	marked := false // the paths followed "--"
	for i, a := range args {
		if a == "--" {
			paths, marked = append(paths, args[i+1:]...), true
			break
		}
		if len(paths) == 0 {
			if base, head, ok := strings.Cut(a, "..."); ok {
				base, head = cmp.Or(base, "HEAD"), cmp.Or(head, "HEAD")
				for _, r := range []string{base, head} {
					if !isRef(r) {
						return nil, usage("%s is not a branch or a commit", r)
					}
				}
				revs = append(revs, base, head)
				continue
			}
			if isRef(a) {
				revs = append(revs, a)
				continue
			}
		}
		if !exists(abs(a)) {
			switch {
			case strings.Contains(a, ".."):
				return nil, usage("%s: write <base>...<head>, with three dots, for what <head> changed since it left <base>", a)
			case len(paths) > 0 && isRef(a):
				return nil, usage("%s: give the branches and commits before the files and folders", a)
			}
			return nil, usage("%s is not a branch, a commit, a file or a folder", a)
		}
		paths = append(paths, a)
	}
	if len(revs) > 2 {
		return nil, usage("too many branches and commits: osp <base> <head> compares two")
	}

	// Two files or two folders on their own are compared with each other.
	if len(revs) == 0 && len(paths) == 2 && !marked {
		a, b := abs(paths[0]), abs(paths[1])
		switch {
		case isDir(a) && isDir(b):
			return &Dirs{Old: a, New: b}, nil
		case !isDir(a) && !isDir(b):
			return &Files{Old: a, New: b}, nil
		}
		return nil, usage("compare two files or two folders, not one of each; for the changes to both, write osp -- %s %s", paths[0], paths[1])
	}
	if len(revs) == 0 && len(paths) > 0 {
		root, rootErr = RepoRoot(existingDir(abs(paths[0])))
	}
	if rootErr != nil {
		if len(args) == 0 && !given {
			return nil, ErrNoRepository
		}
		return nil, rootErr
	}

	src := &Git{Root: root}
	for _, p := range paths {
		rel, ok := within(root, abs(p))
		if !ok {
			return nil, usage("%s is not in the repository at %s", p, root)
		}
		src.Paths = append(src.Paths, rel)
	}
	for _, p := range src.Paths {
		if p == "." {
			src.Paths = nil // the whole repository
			break
		}
	}
	switch len(revs) {
	case 0:
		if committed {
			base, _ := defaultBase(root)
			if base == "" {
				return nil, usage("no branch to compare with: give one, as in osp main --committed")
			}
			src.Base, src.Head = base, "HEAD"
		}
	case 1:
		src.Base = pin(root, revs[0])
		if committed {
			src.Head = "HEAD"
		}
	case 2:
		src.Base, src.Head = pin(root, revs[0]), revs[1]
		if src.Head != "HEAD" { // what is committed on the branch, as --committed
			src.Head = pin(root, src.Head)
		}
	}
	return src, nil
}

// pin names a commit by its hash when rev names it by where it is, as
// HEAD~3 does, rather than by a branch or a tag: the commits made while
// it shows, as an agent's, would move it, and with it the changes shown.
func pin(root, rev string) string {
	for _, refs := range []string{"refs/heads/", "refs/tags/", "refs/remotes/"} {
		if _, err := run(root, "show-ref", "--verify", "--quiet", refs+rev); err == nil {
			return rev
		}
	}
	if short, err := line(root, "rev-parse", "--short", rev+"^{commit}"); err == nil && short != "" {
		return short
	}
	return rev
}

// within returns p, a path on disk, as a path in the repository at root,
// separated by slashes, or ok false when it is outside root. Links are
// followed on both, so that the same folder by two names, as /var and
// /private/var on macOS, is the same.
func within(root, p string) (string, bool) {
	rel, err := filepath.Rel(resolved(root), resolved(p))
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(rel), true
}

// resolved returns p with its links followed, as far as it exists: a file
// deleted is a path still.
func resolved(p string) string {
	if r, err := filepath.EvalSymlinks(p); err == nil {
		return r
	}
	parent := filepath.Dir(p)
	if parent == p {
		return p
	}
	return filepath.Join(resolved(parent), filepath.Base(p))
}

// existingDir returns p when it is a folder, else the nearest folder
// holding it that exists, as a file's, or a deleted file's.
func existingDir(p string) string {
	for !isDir(p) {
		parent := filepath.Dir(p)
		if parent == p {
			break
		}
		p = parent
	}
	return p
}

func isDir(p string) bool {
	fi, err := os.Stat(p)
	return err == nil && fi.IsDir()
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
