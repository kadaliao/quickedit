#!/bin/sh
# Makes .mygo, the MyGo that QuickEdit builds with (go.mod replaces the
# module with it): MyGo's release, with the patches of patches/mygo.
# Run again after changing a patch or MyGo's version.
set -e
cd "$(dirname "$0")/.."
version=v0.8.3
dst=.mygo
stamp="$dst/.quickedit-$version-$(cat patches/mygo/*.patch | shasum -a 256 | cut -c1-12)"
[ -f "$stamp" ] && exit 0
src=$(GOFLAGS=-mod=mod go mod download -json github.com/egoist/mygo@$version | sed -n 's/^[[:space:]]*"Dir": "\(.*\)",$/\1/p')
[ -d "$src" ] || { echo "patch-mygo: no MyGo $version" >&2; exit 1; }
rm -rf "$dst"
cp -R "$src" "$dst"
chmod -R u+w "$dst"
rm -rf "$dst/examples" "$dst/website" "$dst/benchmarks" "$dst/docs"
for p in patches/mygo/*.patch; do
  patch -s -p1 -d "$dst" < "$p"
done
touch "$stamp"
echo "patch-mygo: MyGo $version with $(ls patches/mygo/*.patch | wc -l | tr -d ' ') patch(es) in $dst"
