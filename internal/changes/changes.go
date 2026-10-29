// Package changes reads what changed: the files of a Git repository's
// history or work tree, or of two folders or two files, each with its
// texts before and after, and the commits of a repository to pick from.
package changes

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// Status is how a file changed.
type Status byte

const (
	StatusModified Status = 'M'
	StatusAdded    Status = 'A'
	StatusDeleted  Status = 'D'
	StatusRenamed  Status = 'R'
)

func (s Status) String() string {
	switch s {
	case StatusAdded:
		return "Added"
	case StatusDeleted:
		return "Deleted"
	case StatusRenamed:
		return "Renamed"
	}
	return "Modified"
}

// Change is a file compared: its texts before and after.
type Change struct {
	// Path is where the file is, or was when deleted, separated by
	// slashes; OldPath where it was before a rename.
	Path, OldPath string
	Status        Status
	Old, New      string
	// Note says why the texts don't show, as for a binary file: Old and
	// New are then summaries of the texts, which Hash follows.
	Note string
}

// Hash identifies the change by what it is, so that a review of it lasts
// until the file changes again.
func (ch Change) Hash() string {
	h := sha256.New()
	fmt.Fprintf(h, "%c\x00%s\x00%s\x00%d\x00", ch.Status, ch.OldPath, ch.Note, len(ch.Old))
	h.Write([]byte(ch.Old))
	h.Write([]byte{0})
	h.Write([]byte(ch.New))
	return hex.EncodeToString(h.Sum(nil)[:12])
}

// Source is what the changes are read from.
type Source interface {
	// Load reads the changes. It runs on a goroutine of its own.
	Load() ([]Change, error)
	// Title names what is compared, as the repository; Subtitle says
	// how, as "Since main".
	Title() string
	Subtitle() string
	// WatchDir is the directory whose files are the new side, "" when they
	// are not on disk, as a commit's.
	WatchDir() string
	// Key names the reviews of the changes, kept between launches: a
	// repository's are those of all its comparisons.
	Key() string
}

// maxText is the largest file whose text shows.
const maxText = 8 << 20

const tooLarge = "File too large to show"

// finish sets a change's texts, or its note for texts that don't show as
// text.
func finish(ch *Change, old, new []byte) {
	switch {
	case isBinary(old) || isBinary(new):
		noted(ch, "Binary file", old, new)
	case len(old) > maxText || len(new) > maxText:
		noted(ch, tooLarge, old, new)
	default:
		ch.Old, ch.New = string(old), string(new)
	}
}

// noted gives a change whose texts don't show its note, and in place of
// its texts summaries of them, which its hash follows.
func noted(ch *Change, note string, old, new []byte) {
	ch.Note = note
	ch.Old, ch.New = binarySummary(old), binarySummary(new)
}

// binarySummary stands for a text that doesn't show, so that the hash of
// its change still follows its content.
func binarySummary(b []byte) string {
	if b == nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%d bytes %x", len(b), sum[:8])
}

func isBinary(b []byte) bool {
	return bytes.IndexByte(b[:min(len(b), 8000)], 0) >= 0
}

// readWorkFile reads a file of the work tree as Git sees it: a symlink is
// its target, a directory (a submodule) has no text, and a file too large
// is its size and time, with the note that says why. With follow, a
// symlink is the file it links to, as git difftool --dir-diff links the
// work tree's files.
func readWorkFile(path string, follow bool) ([]byte, string) {
	stat := os.Lstat
	if follow {
		stat = os.Stat
	}
	fi, err := stat(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return nil, ""
	case err != nil:
		return nil, err.Error()
	case fi.Mode()&fs.ModeSymlink != 0:
		target, err := os.Readlink(path)
		if err != nil {
			return nil, err.Error()
		}
		return []byte(target), ""
	case fi.IsDir():
		return nil, "Submodule"
	case fi.Size() > maxText:
		return fmt.Appendf(nil, "%d bytes, modified %d", fi.Size(), fi.ModTime().UnixNano()), tooLarge
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err.Error()
	}
	return b, ""
}
