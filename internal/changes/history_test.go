package changes

import (
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/overcat/osprey/internal/gittest"
)

func TestParseShortstat(t *testing.T) {
	for _, tc := range []struct {
		in                    string
		files, added, deleted int
	}{
		{" 3 files changed, 18 insertions(+), 4 deletions(-)\n", 3, 18, 4},
		{" 1 file changed, 1 insertion(+)\n", 1, 1, 0},
		{" 1 file changed, 2 deletions(-)\n", 1, 0, 2},
		{"\n", 0, 0, 0},
	} {
		files, added, deleted := parseShortstat(tc.in)
		if files != tc.files || added != tc.added || deleted != tc.deleted {
			t.Errorf("%q: %d %d %d", tc.in, files, added, deleted)
		}
	}
}

func subjects(list []Commit) []string {
	var out []string
	for _, c := range list {
		out = append(out, c.Subject)
	}
	return out
}

func TestHistoryOnBranch(t *testing.T) {
	dir := gittest.Branch(t)
	h, err := (&HistoryCache{}).Load(dir, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if h.Base != "main" || h.Upstream {
		t.Fatalf("base %q upstream %v", h.Base, h.Upstream)
	}
	if got := subjects(h.Branch); !slices.Equal(got, []string{"Return errors from login", "Add login"}) {
		t.Fatalf("branch commits %q", got)
	}
	if got := subjects(h.Earlier); !slices.Equal(got, []string{"first"}) {
		t.Fatalf("earlier commits %q", got)
	}
	if c := h.Branch[1]; c.Files != 1 || c.Added != 3 || c.Deleted != 0 || time.Since(c.Time) > time.Hour || len(c.Short) < 7 {
		t.Fatalf("commit details %+v", c)
	}
	if c, ok := h.Find(h.Earlier[0].Hash); !ok || c.Subject != "first" {
		t.Fatal("couldn't find a commit of the history")
	}

	// Kept to README.md, which the branch's commits don't touch.
	h, err = (&HistoryCache{}).Load(dir, "", "", []string{"README.md"})
	if err != nil || len(h.Branch) != 0 || !slices.Equal(subjects(h.Earlier), []string{"first"}) {
		t.Fatalf("the history of README.md: %+v, %v", h, err)
	}
}

func TestHistoryOnMain(t *testing.T) {
	dir := gittest.Repo(t)
	h, err := (&HistoryCache{}).Load(dir, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if h.Base != "" || len(h.Branch) != 0 || !slices.Equal(subjects(h.Earlier), []string{"first"}) {
		t.Fatalf("main without a remote: %+v", h)
	}

	remote := filepath.Join(t.TempDir(), "remote.git")
	gittest.Run(t, dir, "init", "-q", "--bare", remote)
	gittest.Run(t, dir, "remote", "add", "origin", remote)
	gittest.Run(t, dir, "push", "-q", "-u", "origin", "main")
	gittest.Write(t, dir, "README.md", "hello, again\n")
	gittest.Run(t, dir, "commit", "-q", "-am", "Not pushed")
	h, err = (&HistoryCache{}).Load(dir, "", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if h.Base != "origin/main" || !h.Upstream || !slices.Equal(subjects(h.Branch), []string{"Not pushed"}) {
		t.Fatalf("main with commits not pushed: %+v", h)
	}
}

func TestHistoryBeforeFirstCommit(t *testing.T) {
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	gittest.Run(t, dir, "init", "-q")
	if h, err := (&HistoryCache{}).Load(dir, "", "", nil); err != nil || len(h.Branch)+len(h.Earlier) != 0 {
		t.Fatalf("history %+v, %v", h, err)
	}
}

func TestHistoryCacheFollowsCommits(t *testing.T) {
	dir := gittest.Branch(t)
	cache := &HistoryCache{}
	if h, _ := cache.Load(dir, "", "", nil); len(h.Branch) != 2 {
		t.Fatal(h)
	}
	gittest.Run(t, dir, "commit", "-q", "-am", "Third")
	if h, _ := cache.Load(dir, "", "", nil); len(h.Branch) != 3 || h.Branch[0].Subject != "Third" {
		t.Fatalf("a new commit didn't show: %q", subjects(h.Branch))
	}
}

func TestWatchedDirs(t *testing.T) {
	dir := gittest.Repo(t)
	gittest.Write(t, dir, ".gitignore", "node_modules/\n")
	gittest.Write(t, dir, "node_modules/x/index.js", "x\n")
	gittest.Write(t, dir, "web/node_modules/y/index.js", "y\n")
	ignored, tracked := IgnoredDirs(dir), TrackedDirs(dir)
	if !ignored["node_modules"] || !ignored["web/node_modules"] || ignored["web"] || len(ignored) != 2 {
		t.Errorf("ignored %v", ignored)
	}
	if !tracked["src"] || tracked["web"] || len(tracked) != 1 {
		t.Errorf("tracked %v", tracked)
	}
}
