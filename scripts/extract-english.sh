#!/bin/sh
# Extract text from the English PDF sources the importer reads.
#
#   scripts/extract-english.sh [ENG_PRAYER_DIR] [OUT_DIR]
#
# Defaults: $ENG_PRAYER_DIR, var/english. Needs `pdftotext` (poppler-utils).
set -eu
src=${1:-${ENG_PRAYER_DIR:?set ENG_PRAYER_DIR or pass it}}
out=${2:-var/english}
command -v pdftotext >/dev/null || { echo "pdftotext not found" >&2; exit 1; }
mkdir -p "$out"
pdftotext "$src/Coverdale.pdf" "$out/coverdale.txt"
