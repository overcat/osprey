package main

import "strings"

// The sidebar's rows: the files as a tree of their directories, or as a
// list of their paths.

// row is a row of the sidebar: a directory or a file.
type row struct {
	depth int
	// name is what the row shows: a file's name, or a directory's, with
	// those it alone holds, as "src/ui".
	name string
	dir  string // the directory's path, with its slash
	f    *file
	// files and reviewed count the files under a directory.
	files, reviewed int
}

// treeRows lays out files, sorted by comparePaths, as a tree, leaving out
// what is under the directories collapsed.
func treeRows(files []*file, collapsed map[string]bool, isReviewed func(*file) bool) []row {
	var rows []row
	var walk func(files []*file, prefix string, depth int)
	walk = func(files []*file, prefix string, depth int) {
		for i := 0; i < len(files); {
			rest := files[i].Path[len(prefix):]
			slash := strings.IndexByte(rest, '/')
			if slash < 0 {
				rows = append(rows, row{depth: depth, name: rest, f: files[i]})
				i++
				continue
			}
			// The files in this directory, which sort together.
			dir := prefix + rest[:slash+1]
			j := i
			for j < len(files) && strings.HasPrefix(files[j].Path, dir) {
				j++
			}
			group := files[i:j]
			// Join the directories holding nothing but one directory.
			for {
				sub, _, ok := strings.Cut(group[0].Path[len(dir):], "/")
				if !ok {
					break
				}
				next := dir + sub + "/"
				if !allUnder(group, next) {
					break
				}
				dir = next
			}
			r := row{depth: depth, name: strings.TrimSuffix(dir[len(prefix):], "/"), dir: dir, files: len(group)}
			for _, f := range group {
				if isReviewed(f) {
					r.reviewed++
				}
			}
			rows = append(rows, r)
			if !collapsed[dir] {
				walk(group, dir, depth+1)
			}
			i = j
		}
	}
	walk(files, "", 0)
	return rows
}

func allUnder(files []*file, dir string) bool {
	for _, f := range files {
		if !strings.HasPrefix(f.Path, dir) {
			return false
		}
	}
	return true
}

// listRows lays out files as a list, by their paths.
func listRows(files []*file) []row {
	rows := make([]row, len(files))
	for i, f := range files {
		rows[i] = row{name: f.Path, f: f}
	}
	return rows
}

// filterFiles keeps the files whose paths hold each word of query,
// ignoring case.
func filterFiles(files []*file, query string) []*file {
	words := strings.Fields(strings.ToLower(query))
	if len(words) == 0 {
		return files
	}
	var out []*file
next:
	for _, f := range files {
		p := strings.ToLower(f.Path + " " + f.OldPath)
		for _, w := range words {
			if !strings.Contains(p, w) {
				continue next
			}
		}
		out = append(out, f)
	}
	return out
}

// comparePaths orders paths as the tree shows them: a directory's
// subdirectories before its files.
func comparePaths(a, b string) int {
	for {
		ai, bi := strings.IndexByte(a, '/'), strings.IndexByte(b, '/')
		switch {
		case ai < 0 && bi < 0:
			return compareNames(a, b)
		case ai < 0:
			return 1
		case bi < 0:
			return -1
		}
		if c := compareNames(a[:ai], b[:bi]); c != 0 {
			return c
		}
		a, b = a[ai+1:], b[bi+1:]
	}
}

// compareNames orders names ignoring case, then by case.
func compareNames(a, b string) int {
	if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
		return c
	}
	return strings.Compare(a, b)
}

// splitPath returns a path's directory, with its slash, and its name.
func splitPath(p string) (dir, name string) {
	i := strings.LastIndexByte(p, '/')
	return p[:i+1], p[i+1:]
}
