// Package review keeps what the user reviewed: the changes to each file
// marked reviewed, so that a file changed since is to review again, and
// the commits whose every change is reviewed. A change reviewed in one
// comparison of a repository is reviewed in the others it shows in.
package review

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
)

// Store is the reviews of a repository, or of two folders or files,
// kept on disk between launches.
type Store struct {
	path string // "" when kept in memory only
	// marks are the hashes of the changes reviewed, by path, the latest
	// last.
	marks map[string][]string
	done  map[string]bool // by commit hash
}

// kept is how many changes to a path a store remembers.
const kept = 8

// saved is a store as it is kept on disk.
type saved struct {
	Marks map[string][]string `json:"marks"`
	Done  []string            `json:"done,omitempty"`
}

// New returns an empty store, kept in memory only.
func New() *Store {
	return &Store{marks: map[string][]string{}, done: map[string]bool{}}
}

// Load returns the store key names, as a source's Key, kept in the
// user's configuration directory: empty when there is none yet.
func Load(key string) *Store {
	s := New()
	dir, err := os.UserConfigDir()
	if err != nil {
		return s
	}
	sum := sha256.Sum256([]byte(key))
	s.path = filepath.Join(dir, "Osprey", "reviews", hex.EncodeToString(sum[:12])+".json")
	var file saved
	if b, err := os.ReadFile(s.path); err == nil && json.Unmarshal(b, &file) == nil {
		if file.Marks != nil {
			s.marks = file.Marks
		}
		for _, c := range file.Done {
			s.done[c] = true
		}
	}
	return s
}

// Reviewed reports whether the change with the hash, to the file at path,
// is reviewed.
func (s *Store) Reviewed(path, hash string) bool { return slices.Contains(s.marks[path], hash) }

// Set marks the change with the hash, to the file at path, reviewed or
// not.
func (s *Store) Set(path, hash string, reviewed bool) {
	hashes := slices.DeleteFunc(slices.Clone(s.marks[path]), func(h string) bool { return h == hash })
	if reviewed {
		hashes = append(hashes, hash)
		hashes = hashes[max(0, len(hashes)-kept):]
	}
	if len(hashes) == 0 {
		delete(s.marks, path)
	} else {
		s.marks[path] = hashes
	}
	s.save()
}

// Done reports whether every change of the commit is reviewed.
func (s *Store) Done(commit string) bool { return s.done[commit] }

// SetDone records whether every change of the commit is reviewed.
func (s *Store) SetDone(commit string, done bool) {
	if s.done[commit] == done {
		return
	}
	if done {
		s.done[commit] = true
	} else {
		delete(s.done, commit)
	}
	s.save()
}

func (s *Store) save() {
	if s.path == "" {
		return
	}
	file := saved{Marks: s.marks}
	for c := range s.done {
		file.Done = append(file.Done, c)
	}
	slices.Sort(file.Done)
	b, _ := json.Marshal(file)
	// Written whole, then put in place, so that a crash never leaves half
	// a file, which would lose every review.
	dir := filepath.Dir(s.path)
	os.MkdirAll(dir, 0o755)
	tmp, err := os.CreateTemp(dir, ".reviews-*")
	if err != nil {
		return
	}
	_, err = tmp.Write(b)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmp.Name(), s.path)
	}
	if err != nil {
		os.Remove(tmp.Name())
	}
}
