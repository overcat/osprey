#!/bin/sh
# Builds Osprey and installs it for this user: Osprey.app in /Applications
# (~/Applications when that isn't writable), and the osprey and osp
# commands on the PATH, with their completions. Run it again to update.
set -eu

cd "$(dirname "$0")/.."

# Build outside the project and remove the build once installed: Spotlight
# and the launcher list every Osprey.app they find, a build left in the
# project's dist/ as a second Osprey.
build="$(mktemp -d)"
trap 'rm -rf "$build"' EXIT
echo "Building Osprey.app…"
go tool mygo build -skip-dmg -o "$build" . >/dev/null
app="$(find "$build" -maxdepth 2 -name Osprey.app -type d | head -n 1)"

apps=/Applications
if [ ! -w "$apps" ]; then
	apps="$HOME/Applications"
	mkdir -p "$apps"
fi
rm -rf "$apps/Osprey.app"
ditto "$app" "$apps/Osprey.app"
echo "Installed $apps/Osprey.app"

exe="$apps/Osprey.app/Contents/MacOS/Osprey"
"$exe" install-cli ${1:-}

# The program was kept here before it was an app, and builds in dist/
# before builds went elsewhere.
rm -rf "$HOME/.local/share/osprey"
lsregister=/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister
if [ -d dist ]; then
	find dist -maxdepth 2 -name Osprey.app -type d -exec "$lsregister" -u {} \; 2>/dev/null || true
	rm -rf dist
fi

# Earlier, this script put completions in the shells' own folders, where
# they would shadow those install-cli keeps up to date.
for f in "$HOME/.config/fish/completions/osprey.fish" "$HOME/.config/fish/completions/osp.fish"; do
	if [ -f "$f" ] && grep -q '^complete -c osp' "$f"; then
		rm "$f"
	fi
done
if [ -f "$HOME/.zfunc/_osprey" ] && grep -q '^#compdef osprey osp' "$HOME/.zfunc/_osprey"; then
	rm "$HOME/.zfunc/_osprey"
fi
