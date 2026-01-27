#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
if [ -z "${1:-}" ]; then
  echo "usage: ./tests/test.sh <DIR with files to parse>" >&2
  exit 1
fi
DIR="$1"

TERMS="$ROOT/tests/terms"

# concurrent test run
cat "$TERMS" | go run "$ROOT/indexer.go" "$DIR" > "$ROOT/tests/out.conc" 2> "$ROOT/tests/conc.err"
echo "---------------------------------------------" >> "$ROOT/tests/out.conc"
cat "$ROOT/tests/conc.err" >> "$ROOT/tests/out.conc"
rm -f "$ROOT/tests/conc.err"


# sequential test run
cat "$TERMS" | INDEX_MODE=seq go run "$ROOT/indexer.go" "$DIR" > "$ROOT/tests/out.seq" 2> "$ROOT/tests/seq.err"
echo "---------------------------------------------" >> "$ROOT/tests/out.seq"
cat "$ROOT/tests/seq.err" >> "$ROOT/tests/out.seq"
rm -f "$ROOT/tests/seq.err"

# print the build time for each
conc_s=$(awk '/^BUILD /{for(i=1;i<=NF;i++) if($i ~ /^seconds=/){split($i,a,"="); print a[2]}}' conc.err)
seq_s=$(awk  '/^BUILD /{for(i=1;i<=NF;i++) if($i ~ /^seconds=/){split($i,a,"="); print a[2]}}' seq.err)

# compute speedup (seq/conc) using awk (no bc needed)
speedup=$(awk -v s="$seq_s" -v c="$conc_s" 'BEGIN{ if(c>0) printf("%.2f", s/c); else print "inf"}')

echo "---------------BUILD STATS-------------------" 
echo "seq_seconds=$seq_s" >&2
echo "conc_seconds=$conc_s" >&2
echo "speedup=${speedup}x" >&2
echo "---------------------------------------------" 
echo " "

echo "-------------------DIFF----------------------" 
diff -u "$ROOT/tests/out.seq" "$ROOT/tests/out.conc" || true
