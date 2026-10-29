package main

// installFallback installs osprey and osp where this user can write in
// none of the usual directories of the PATH, as on a fresh Linux: in
// ~/.local/bin.
func installFallback(exe string) (string, error) { return installInHome(exe) }
