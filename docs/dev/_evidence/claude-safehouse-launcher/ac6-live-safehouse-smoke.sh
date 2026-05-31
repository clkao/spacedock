#!/bin/bash
# ABOUTME: AC-6 captain-run live safehouse smoke — runs the canonical safehouse
# ABOUTME: argv outside the sandbox and records evidence that closes F3 (Risk A).
#
# Run this OUTSIDE the spacedock sandbox in a real unsandboxed shell. We run
# inside safehouse during development, so nested safehouse will not run here —
# AC-6 is captain-run by design.
#
# What it proves (see entity AC-6): `safehouse --trust-workdir-config` is a flag
# safehouse accepts, the `--` correctly hands the remainder to claude, and claude
# accepts both `--dangerously-skip-permissions` and `--agent`. Success = claude's
# OWN --help text on stdout (NOT a safehouse "unknown flag" / "unrecognized
# argument" error, and NOT a claude "unknown flag --trust-workdir-config /
# --dangerously-skip-permissions" error) and rc == 0.
#
# If the observed flag surface differs (safehouse spells the trust flag
# differently, or rejects `--`), the AC-1 canonical argv in
# internal/cli/safehouse_frontdoor_test.go must be corrected to match reality
# BEFORE the validation gate locks it. This script is the gate F3 guards.

set -u

EVIDENCE_DIR="${EVIDENCE_DIR:-/tmp/spacedock-ac6-evidence}"
mkdir -p "$EVIDENCE_DIR"
STDOUT_FILE="$EVIDENCE_DIR/ac6-stdout.txt"
STDERR_FILE="$EVIDENCE_DIR/ac6-stderr.txt"

# The canonical argv this command must match — identical to AC-1's expected
# Launch argv (sans the trailing bootstrap prompt; --help is the smoke probe).
CANONICAL=(safehouse --trust-workdir-config -- claude --dangerously-skip-permissions --agent spacedock:first-officer --help)

echo "AC-6 live safehouse smoke"
echo "command: ${CANONICAL[*]}"
echo "evidence dir: $EVIDENCE_DIR"
echo "---"

"${CANONICAL[@]}" >"$STDOUT_FILE" 2>"$STDERR_FILE"
RC=$?

echo "rc: $RC"
echo "--- stdout head ---"
head -20 "$STDOUT_FILE"
echo "--- stderr head ---"
head -20 "$STDERR_FILE"
echo "---"

# Verdict heuristic: PASS requires rc==0 and claude help markers on stdout, with
# no safehouse/claude flag-rejection strings.
if [ "$RC" -eq 0 ] && grep -qiE 'usage|--help|options' "$STDOUT_FILE" \
   && ! grep -qiE 'unknown flag|unrecognized argument|unknown option' "$STDOUT_FILE" "$STDERR_FILE"; then
  echo "VERDICT: PASS — safehouse consumed --trust-workdir-config and handed the rest to claude."
else
  echo "VERDICT: INSPECT — rc or flag surface differs from the canonical argv."
  echo "Correct AC-1's expected argv to match reality before the validation gate locks it."
fi
exit "$RC"
