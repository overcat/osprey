package changes

import (
	"bytes"
	"cmp"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Git compares a Git repository's files: the old side is a commit, the
// new side a commit or the work tree.
type Git struct {
	Root string
	// Base is the ref to compare against, "" for HEAD; Head the ref
	// compared, "" for the work tree.
	Base, Head string
	// Commit, when set, is the one commit compared, with its parent, in
	// place of Base and Head.
	Commit string
	// Paths, when set, keep the changes to the files under them: paths
	// in the repository, separated by slashes.
	Paths []string
}

func (s *Git) Title() string { return filepath.Base(s.Root) }

func (s *Git) Subtitle() string {
	var what string
	switch {
	case s.Commit != "":
		what = "Commit " + s.Commit[:min(7, len(s.Commit))]
	case s.Base == "":
		what = "Uncommitted changes"
	case s.Head == "":
		what = "Since " + s.Base + " · with uncommitted"
	case s.Head == "HEAD":
		what = "Since " + s.Base
	default:
		what = s.Base + " → " + s.Head
	}
	return what + PathsLabel(s.Paths)
}

// PathsLabel says which files the changes are kept to, as a label of
// them ends: "" for all of them.
func PathsLabel(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	return " · in " + strings.Join(paths, ", ")
}

func (s *Git) WatchDir() string {
	if s.Head == "" && s.Commit == "" {
		return s.Root
	}
	return ""
}

// Key is the repository's: a change reviewed in one comparison is in the
// others it shows in, as the same change.
func (s *Git) Key() string { return "git\x00" + s.Root }

// Uncommitted reports whether the changes are those not committed yet.
func (s *Git) Uncommitted() bool { return s.Base == "" && s.Commit == "" }

// Branch returns the branch the work tree has checked out.
func (s *Git) Branch() string { return currentBranch(s.Root) }

// revs returns the commit of the old side and of the new, "" for the work
// tree.
func (s *Git) revs() (old, new string, err error) {
	if s.Commit != "" {
		new, ok := resolveCommit(s.Root, s.Commit)
		if !ok {
			return "", "", fmt.Errorf("unknown commit: %s", s.Commit)
		}
		if old, ok = resolveCommit(s.Root, new+"^"); !ok { // the first commit
			old, err = emptyTree(s.Root)
		}
		return old, new, err
	}
	head := cmp.Or(s.Head, "HEAD")
	switch {
	case s.Base == "":
		if oid, ok := resolveCommit(s.Root, "HEAD"); ok {
			old = oid
		} else if old, err = emptyTree(s.Root); err != nil { // before the first commit
			return "", "", err
		}
	default:
		if _, ok := resolveCommit(s.Root, s.Base); !ok {
			return "", "", fmt.Errorf("unknown branch or commit: %s", s.Base)
		}
		if _, ok := resolveCommit(s.Root, head); !ok {
			return "", "", fmt.Errorf("unknown branch or commit: %s", head)
		}
		old = mergeBase(s.Root, s.Base, head)
	}
	if s.Head != "" {
		new, _ = resolveCommit(s.Root, s.Head)
	}
	return old, new, nil
}

func (s *Git) Load() ([]Change, error) {
	old, new, err := s.revs()
	if err != nil {
		return nil, err
	}
	changes, err := nameStatus(s.Root, old, new, s.Paths)
	if err != nil {
		return nil, err
	}
	if new == "" {
		paths, err := untracked(s.Root, s.Paths)
		if err != nil {
			return nil, err
		}
		for _, p := range paths {
			changes = append(changes, Change{Path: p, Status: StatusAdded})
		}
	}
	// The old texts, and the new ones of a commit, in one git process:
	// at[i] are where change i's are among them, -1 for none.
	var specs []string
	spec := func(rev, path string) int {
		specs = append(specs, rev+":"+path)
		return len(specs) - 1
	}
	at := make([]struct{ old, new int }, len(changes))
	for i, ch := range changes {
		at[i].old, at[i].new = -1, -1
		if ch.Status != StatusAdded {
			at[i].old = spec(old, cmp.Or(ch.OldPath, ch.Path))
		}
		if new != "" && ch.Status != StatusDeleted {
			at[i].new = spec(new, ch.Path)
		}
	}
	blobs, err := catFiles(s.Root, specs)
	if err != nil {
		return nil, err
	}
	for i := range changes {
		ch := &changes[i]
		var oldText, newText []byte
		if at[i].old >= 0 {
			oldText = blobs[at[i].old]
		}
		switch {
		case at[i].new >= 0:
			newText = blobs[at[i].new]
		case ch.Status != StatusDeleted: // in the work tree
			var note string
			if newText, note = readWorkFile(filepath.Join(s.Root, filepath.FromSlash(ch.Path)), false); note != "" {
				noted(ch, note, oldText, newText)
				continue
			}
		}
		finish(ch, oldText, newText)
	}
	return changes, nil
}

// Dirs compares two directories, as git difftool --dir-diff gives them.
type Dirs struct{ Old, New string }

func (s *Dirs) Title() string    { return filepath.Base(s.New) }
func (s *Dirs) Subtitle() string { return s.Old + " → " + s.New }
func (s *Dirs) WatchDir() string { return s.New }
func (s *Dirs) Key() string      { return "dir\x00" + s.Old + "\x00" + s.New }

func (s *Dirs) Load() ([]Change, error) {
	oldFiles, err := listFiles(s.Old)
	if err != nil {
		return nil, err
	}
	newFiles, err := listFiles(s.New)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(oldFiles)+len(newFiles))
	for p := range oldFiles {
		paths = append(paths, p)
	}
	for p := range newFiles {
		if !oldFiles[p] {
			paths = append(paths, p)
		}
	}
	slices.Sort(paths)
	var changes []Change
	for _, p := range paths {
		ch := Change{Path: p, Status: StatusModified}
		switch {
		case !oldFiles[p]:
			ch.Status = StatusAdded
		case !newFiles[p]:
			ch.Status = StatusDeleted
		}
		oldText, oldNote := readWorkFile(filepath.Join(s.Old, filepath.FromSlash(p)), true)
		newText, newNote := readWorkFile(filepath.Join(s.New, filepath.FromSlash(p)), true)
		if ch.Status == StatusModified && oldNote == "" && newNote == "" && bytes.Equal(oldText, newText) {
			continue
		}
		if note := cmp.Or(oldNote, newNote); note != "" {
			noted(&ch, note, oldText, newText)
		} else {
			finish(&ch, oldText, newText)
		}
		changes = append(changes, ch)
	}
	return changes, nil
}

// listFiles lists the files under dir, by their paths in it.
func listFiles(dir string) (map[string]bool, error) {
	files := map[string]bool{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if p == dir {
				return err
			}
			return nil // leave out what can't be read
		}
		if d.IsDir() {
			if d.Name() == ".git" && p != dir {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		files[filepath.ToSlash(rel)] = true
		return nil
	})
	return files, err
}

// Files compares two files.
type Files struct{ Old, New string }

func (s *Files) Title() string    { return filepath.Base(s.New) }
func (s *Files) Subtitle() string { return s.Old + " → " + s.New }
func (s *Files) WatchDir() string { return s.New }
func (s *Files) Key() string      { return "file\x00" + s.Old + "\x00" + s.New }

func (s *Files) Load() ([]Change, error) {
	oldText, err := os.ReadFile(s.Old)
	if err != nil {
		return nil, err
	}
	newText, err := os.ReadFile(s.New)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(oldText, newText) {
		return nil, nil
	}
	ch := Change{Path: filepath.Base(s.New), Status: StatusModified}
	if name := filepath.Base(s.Old); name != ch.Path {
		ch.OldPath = name
	}
	finish(&ch, oldText, newText)
	return []Change{ch}, nil
}
