#!/bin/sh
# Releases Osprey as of today. Its version is the date, as 2026.10.29, and
# 2026.10.29.1 for another release the same day: the script sets it in
# mygo.json, commits that on main, tags the commit with it, and pushes
# both. The Release workflow then builds every platform into a draft
# release, to check and publish by hand. With -n, it only says what it
# would release.
#
#   scripts/release.sh [-n]
set -eu

cd "$(dirname "$0")/.."
fail() {
	echo "release: $*" >&2
	exit 1
}

# What is released is main as it is on GitHub, once CI passed it.
[ "$(git symbolic-ref --short HEAD)" = main ] || fail "check out main first"
[ -z "$(git status --porcelain)" ] || fail "commit or stash what isn't committed first"
git fetch -q --tags origin main
[ "$(git rev-parse HEAD)" = "$(git rev-parse origin/main)" ] || fail "main isn't GitHub's main: pull or push first"
ci="$(gh run list --branch main --workflow CI --commit "$(git rev-parse HEAD)" --json conclusion --jq '.[0].conclusion' 2>/dev/null || true)"
[ "$ci" = success ] || fail "CI hasn't passed main's last commit yet (${ci:-no run found})"

# Today, its numbers without zeros in front, which some tools refuse.
version="$(date +%Y).$(expr "$(date +%m)" + 0).$(expr "$(date +%d)" + 0)"
if git rev-parse -q --verify "refs/tags/$version" >/dev/null; then
	n=1
	while git rev-parse -q --verify "refs/tags/$version.$n" >/dev/null; do
		n=$((n + 1))
	done
	version="$version.$n"
fi
if [ "${1:-}" = -n ]; then
	echo "Would release $version"
	exit 0
fi

sed "s/^\(  \"version\": \"\)[^\"]*\"/\1$version\"/" mygo.json >mygo.json.new
mv mygo.json.new mygo.json
grep -q "\"version\": \"$version\"" mygo.json || fail "couldn't set the version in mygo.json"
git commit -q -m "Release $version" mygo.json
git tag -a "$version" -m "Osprey $version"
git push -q --atomic origin main "$version"
echo "Released $version: the Release workflow is building it into a draft release."
