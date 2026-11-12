#!/bin/sh
# Extract the RAR archives of the orthlib.ru "worship" section into a tree of
# .hip files that the importer reads.
#
#   scripts/extract-orthlib.sh [ORTHLIB_DIR] [OUT_DIR]
#
# Defaults: $ORTHLIB_DIR, var/orthlib. Needs `unrar`.
set -eu
src=${1:-${ORTHLIB_DIR:?set ORTHLIB_DIR or pass it}}
out=${2:-var/orthlib}
command -v unrar >/dev/null || { echo "unrar not found" >&2; exit 1; }
mkdir -p "$out"
out=$(cd "$out" && pwd)
cd "$src/worship"
find . -name '*.rar' | sort | while read -r f; do
	d="$out/$(dirname "$f")/$(basename "$f" .rar)"
	mkdir -p "$d"
	if ! unrar x -o+ -inul "$f" "$d/"; then
		echo "warning: $f could not be fully extracted" >&2
	fi
done
