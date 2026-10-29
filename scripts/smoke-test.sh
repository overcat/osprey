#!/bin/sh
# Opens Osprey on a new repository with a change not committed yet, saves a
# picture of its window once the change has loaded, and checks that it
# quit: that the app starts, and shows changes, where it runs.
#
#   scripts/smoke-test.sh <the osprey program> <picture.png>
set -eu

app="$1"
shot="$2"
repo="$(mktemp -d)"
trap 'rm -rf "$repo"' EXIT
mkdir -p "$(dirname "$shot")"
rm -f "$shot"

git() { command git -C "$repo" -c user.name=Osprey -c user.email=osprey@example.com "$@"; }
git init -q -b main
printf 'package main\n\nfunc main() {}\n' >"$repo/main.go"
git add main.go
git commit -q -m "Add main"
printf 'package main\n\nfunc main() {\n\tprintln("hello")\n}\n' >"$repo/main.go"

OSPREY_SNAPSHOT="$shot" "$app" --wait -C "$repo" &
pid=$!
waited=0
while kill -0 "$pid" 2>/dev/null; do
	if [ "$waited" -ge 120 ]; then
		kill "$pid"
		echo "Osprey didn't quit within 2 minutes" >&2
		exit 1
	fi
	sleep 1
	waited=$((waited + 1))
done
wait "$pid"
if [ ! -s "$shot" ]; then
	echo "Osprey saved no picture of its window" >&2
	exit 1
fi
echo "Saved $shot"
