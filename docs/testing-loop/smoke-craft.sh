#!/usr/bin/env bash
# dojo-cli/docs/testing-loop/smoke-craft.sh
#
# Smoke tests for the /craft command group.
# Tests each subcommand in offline mode (no Gateway required for local commands)
# and online mode (Gateway required for memory/seed/adr/scout).
#
# Usage:
#   ./smoke-craft.sh              # run all tests
#   ./smoke-craft.sh --offline    # skip Gateway-dependent tests
#   DOJO_BIN=/path/to/dojo ./smoke-craft.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
GATEWAY="${DOJO_GATEWAY:-http://localhost:7340}"
OFFLINE=false

for arg in "$@"; do
  [[ "$arg" == "--offline" ]] && OFFLINE=true
done

# ─── Locate dojo binary ───────────────────────────────────────────────────────

locate_dojo() {
  if [[ -n "${DOJO_BIN:-}" ]]; then echo "$DOJO_BIN"; return; fi
  if command -v dojo &>/dev/null; then command -v dojo; return; fi
  echo "  [build] building dojo from $CLI_ROOT ..." >&2
  (cd "$CLI_ROOT" && go build -o /tmp/dojo-craft-smoke ./cmd/dojo/) >&2
  echo "/tmp/dojo-craft-smoke"
}

DOJO_BIN="$(locate_dojo)"

# ─── Temp workspace ──────────────────────────────────────────────────────────

TMPDIR="$(mktemp -d /tmp/dojo-craft-smoke-XXXXXX)"
trap 'rm -rf "$TMPDIR"' EXIT

# ─── Helpers ─────────────────────────────────────────────────────────────────

PASS=0
FAIL=0
SKIP=0
declare -a FAIL_DETAILS=()

pass() { PASS=$((PASS + 1)); printf "  %-40s  %s\n" "$1" "PASS"; }
fail() { FAIL=$((FAIL + 1)); FAIL_DETAILS+=("$1: $2"); printf "  %-40s  %s\n" "$1" "FAIL — $2"; }
skip() { SKIP=$((SKIP + 1)); printf "  %-40s  %s\n" "$1" "SKIP (offline)"; }

gateway_up() {
  $OFFLINE && return 1
  curl -s --connect-timeout 2 --max-time 3 -o /dev/null "$GATEWAY/health" 2>/dev/null
}

# ─── Header ──────────────────────────────────────────────────────────────────

echo ""
echo "  /craft smoke tests"
echo "  binary  : $DOJO_BIN"
echo "  gateway : $GATEWAY"
echo "  offline : $OFFLINE"
echo "  tmpdir  : $TMPDIR"
echo ""
printf "  %-40s  %s\n" "Test" "Result"
printf "  %s\n" "$(printf '─%.0s' $(seq 1 55))"

# ─── Test 1: /craft help ─────────────────────────────────────────────────────

out=$("$DOJO_BIN" --one-shot "/craft" --plain --gateway "$GATEWAY" 2>/dev/null || true)
if echo "$out" | grep -q "DojoCraft"; then
  pass "/craft help"
else
  fail "/craft help" "missing DojoCraft header"
fi

# ─── Test 2: /craft view ─────────────────────────────────────────────────────

cd "$CLI_ROOT"
out=$("$DOJO_BIN" --one-shot "/craft view ." --plain --gateway "$GATEWAY" 2>/dev/null || true)
if echo "$out" | grep -q "Codebase View"; then
  pass "/craft view ."
else
  fail "/craft view ." "missing Codebase View header"
fi

# Check it finds go.mod
if echo "$out" | grep -q "go.mod\|Go module"; then
  pass "/craft view detects go.mod"
else
  fail "/craft view detects go.mod" "go.mod not found in output"
fi

# Check it finds entry points
if echo "$out" | grep -q "main\|Entry"; then
  pass "/craft view finds entry points"
else
  fail "/craft view finds entry points" "no entry points in output"
fi

# ─── Test 3: /craft scaffold ─────────────────────────────────────────────────

cd "$TMPDIR"

# List templates
out=$("$DOJO_BIN" --one-shot "/craft scaffold" --plain --gateway "$GATEWAY" 2>/dev/null || true)
if echo "$out" | grep -q "go-service"; then
  pass "/craft scaffold lists templates"
else
  fail "/craft scaffold lists templates" "missing go-service template"
fi

# Actually scaffold
mkdir -p "$TMPDIR/test-scaffold" && cd "$TMPDIR/test-scaffold"
out=$("$DOJO_BIN" --one-shot "/craft scaffold orchestration" --plain --gateway "$GATEWAY" 2>/dev/null || true)
if [[ -d "$TMPDIR/test-scaffold/decisions" ]] && [[ -f "$TMPDIR/test-scaffold/CLAUDE.md" ]]; then
  pass "/craft scaffold orchestration"
else
  fail "/craft scaffold orchestration" "missing decisions/ or CLAUDE.md"
fi

# ─── Test 4: /craft converge ─────────────────────────────────────────────────

cd "$CLI_ROOT"
out=$("$DOJO_BIN" --one-shot "/craft converge" --plain --gateway "$GATEWAY" 2>/dev/null || true)
if echo "$out" | grep -qE "RED|YELLOW|GREEN"; then
  pass "/craft converge signal"
else
  fail "/craft converge signal" "no RED/YELLOW/GREEN signal"
fi

if echo "$out" | grep -q "dirty files"; then
  pass "/craft converge metrics"
else
  fail "/craft converge metrics" "missing dirty files metric"
fi

# ─── Test 5: /craft scaffold --skip existing ─────────────────────────────────

cd "$TMPDIR/test-scaffold"
out=$("$DOJO_BIN" --one-shot "/craft scaffold orchestration" --plain --gateway "$GATEWAY" 2>/dev/null || true)
if echo "$out" | grep -q "skip\|exists"; then
  pass "/craft scaffold skip existing"
else
  fail "/craft scaffold skip existing" "should skip existing files"
fi

# ─── Test 6: /craft memory (Gateway-dependent) ──────────────────────────────

if gateway_up; then
  cd "$CLI_ROOT"
  out=$("$DOJO_BIN" --one-shot "/craft memory ls" --plain --gateway "$GATEWAY" 2>/dev/null || true)
  if echo "$out" | grep -qE "Memory|No memories"; then
    pass "/craft memory ls"
  else
    fail "/craft memory ls" "unexpected output"
  fi

  # Add + search + rm cycle
  out=$("$DOJO_BIN" --one-shot "/craft memory add smoke-test-entry-$(date +%s)" --plain --gateway "$GATEWAY" 2>/dev/null || true)
  if echo "$out" | grep -q "stored\|id"; then
    pass "/craft memory add"
  else
    fail "/craft memory add" "no stored confirmation"
  fi

  out=$("$DOJO_BIN" --one-shot "/craft memory search smoke-test" --plain --gateway "$GATEWAY" 2>/dev/null || true)
  if echo "$out" | grep -q "Search\|result"; then
    pass "/craft memory search"
  else
    fail "/craft memory search" "no search results"
  fi
else
  skip "/craft memory ls"
  skip "/craft memory add"
  skip "/craft memory search"
fi

# ─── Test 7: /craft seed (Gateway-dependent) ────────────────────────────────

if gateway_up; then
  out=$("$DOJO_BIN" --one-shot "/craft seed ls" --plain --gateway "$GATEWAY" 2>/dev/null || true)
  if echo "$out" | grep -qE "Seeds|Garden|empty"; then
    pass "/craft seed ls"
  else
    fail "/craft seed ls" "unexpected output"
  fi

  out=$("$DOJO_BIN" --one-shot "/craft seed plant smoke-test-seed-content" --plain --gateway "$GATEWAY" 2>/dev/null || true)
  if echo "$out" | grep -q "planted\|Seed\|id"; then
    pass "/craft seed plant"
  else
    fail "/craft seed plant" "no planted confirmation"
  fi

  out=$("$DOJO_BIN" --one-shot "/craft seed search smoke" --plain --gateway "$GATEWAY" 2>/dev/null || true)
  if echo "$out" | grep -qE "Search|Seed|result|of"; then
    pass "/craft seed search"
  else
    fail "/craft seed search" "no search results"
  fi
else
  skip "/craft seed ls"
  skip "/craft seed plant"
  skip "/craft seed search"
fi

# ─── Test 8: /craft adr (Gateway-dependent) ─────────────────────────────────

if gateway_up; then
  out=$("$DOJO_BIN" --one-shot "/craft adr Test Decision for Smoke" --plain --gateway "$GATEWAY" 2>/dev/null || true)
  if echo "$out" | grep -qE "ADR|Decision|Context"; then
    pass "/craft adr"
  else
    fail "/craft adr" "no ADR output"
  fi
else
  skip "/craft adr"
fi

# ─── Test 9: /craft scout (Gateway-dependent) ───────────────────────────────

if gateway_up; then
  out=$("$DOJO_BIN" --one-shot "/craft scout Should we use SQLite or PostgreSQL" --plain --gateway "$GATEWAY" 2>/dev/null || true)
  if echo "$out" | grep -qE "Scout|Tension|Route|route|Decision|decision"; then
    pass "/craft scout"
  else
    fail "/craft scout" "no scout output"
  fi
else
  skip "/craft scout"
fi

# ─── Test 10: /craft claude-md ───────────────────────────────────────────────

if gateway_up; then
  cd "$CLI_ROOT"
  out=$("$DOJO_BIN" --one-shot "/craft claude-md" --plain --gateway "$GATEWAY" 2>/dev/null || true)
  if echo "$out" | grep -qE "CLAUDE.md|No CLAUDE.md|Analysis"; then
    pass "/craft claude-md"
  else
    fail "/craft claude-md" "no CLAUDE.md analysis"
  fi
else
  skip "/craft claude-md"
fi

# ─── Test 11: error handling ─────────────────────────────────────────────────

out=$("$DOJO_BIN" --one-shot "/craft nonexistent" --plain --gateway "$GATEWAY" 2>&1 || true)
if echo "$out" | grep -qE "unknown|try:"; then
  pass "/craft unknown subcommand error"
else
  fail "/craft unknown subcommand error" "no error message for unknown subcommand"
fi

out=$("$DOJO_BIN" --one-shot "/craft memory" --plain --gateway "$GATEWAY" 2>/dev/null || true)
if echo "$out" | grep -qE "Memory|memory"; then
  pass "/craft memory (no args = ls)"
else
  fail "/craft memory (no args = ls)" "did not default to ls"
fi

# ─── Summary ─────────────────────────────────────────────────────────────────

TOTAL=$((PASS + FAIL + SKIP))
printf "  %s\n" "$(printf '─%.0s' $(seq 1 55))"
echo ""
echo "  $PASS passed / $FAIL failed / $SKIP skipped  ($TOTAL total)"
echo ""

if [[ $FAIL -gt 0 ]]; then
  echo "  Failures:"
  for d in "${FAIL_DETAILS[@]}"; do
    echo "    - $d"
  done
  echo ""
fi

exit $FAIL
