#!/bin/bash
# ABOUTME: AC-4 captain-run live codex smoke — confirms the safehouse+codex+skill
# ABOUTME: launch chain outside the sandbox and records evidence that locks AC-1/2.
#
# Run this OUTSIDE the spacedock sandbox in a real unsandboxed shell, in a
# directory that HAS a `.safehouse` profile and where the spacedock plugin is
# installed in codex (`codex plugin list` shows `spacedock@spacedock (installed`).
# We run inside safehouse during development and the interactive codex TUI would
# hang an agent, so AC-4 is captain-run by design.
#
# What it proves (entity AC-4): the canonical launch argv
#   safehouse --trust-workdir-config -- codex --dangerously-bypass-approvals-and-sandbox "<prompt naming spacedock:first-officer>"
# is what `spacedock codex` execs, and the full chain works:
#   (1) codex's startup banner shows `sandbox: danger-full-access` + `approval: never`
#       (the bypass flag landed; safehouse, not codex, is the sandbox);
#   (2) codex invokes the spacedock:first-officer skill from the positional prompt
#       and the FO startup sequence begins;
#   (3) the FO can run `spacedock status` / dispatch from inside the safehouse
#       sandbox (outside codex's native sandbox).
#
# PART A is a non-interactive argv check that does not hang: it records the exact
# argv via a `codex` stub on PATH. PART B is the interactive launch the captain
# drives by eye for the banner + skill-pickup evidence.
#
# If the observed argv or flag surface differs from the oracle, correct AC-1's
# expected Launch argv in internal/cli/safehouse_frontdoor_test.go to match
# reality BEFORE the validation gate locks it.

set -u

SPACEDOCK="${SPACEDOCK:-spacedock}"
EVIDENCE_DIR="${EVIDENCE_DIR:-/tmp/spacedock-ac4-evidence}"
mkdir -p "$EVIDENCE_DIR"
STUB_DIR="$EVIDENCE_DIR/stub-bin"
mkdir -p "$STUB_DIR"
RECORD="$EVIDENCE_DIR/recorded-argv.txt"
HERE="$(cd "$(dirname "$0")" && pwd)"

echo "AC-4 live codex smoke"
echo "evidence dir: $EVIDENCE_DIR"
echo "============================================================"
echo "PART A — non-interactive argv recording (no hang)"
echo "  Drops a codex stub on PATH and runs \`$SPACEDOCK codex\` in NO-safehouse"
echo "  mode to record the inner argv. (Captain option (b): no bypass flag here.)"
echo "------------------------------------------------------------"

# Install the recording stub as `codex` early on PATH.
cp "$HERE/codex-argv-stub.sh" "$STUB_DIR/codex"
chmod +x "$STUB_DIR/codex"

# Run in a temp dir with NO .safehouse so the stub (not safehouse) is exec'd and
# records the inner argv. The no-safehouse oracle: `codex <fo-prompt>`, NO bypass.
WORKDIR_NOSH="$EVIDENCE_DIR/no-safehouse-workdir"
mkdir -p "$WORKDIR_NOSH"
RECORD="$RECORD" PATH="$STUB_DIR:$PATH" \
  bash -c "cd '$WORKDIR_NOSH' && '$SPACEDOCK' codex --skip-contract-check" \
  >"$EVIDENCE_DIR/partA-stdout.txt" 2>"$EVIDENCE_DIR/partA-stderr.txt"
echo "recorded inner argv (no-safehouse path):"
cat "$RECORD" 2>/dev/null
echo "---"
if grep -q 'spacedock:first-officer' "$RECORD" 2>/dev/null \
   && grep -q '^codex$' "$RECORD" 2>/dev/null \
   && ! grep -q 'dangerously-bypass-approvals-and-sandbox' "$RECORD" 2>/dev/null; then
  echo "PART A VERDICT: PASS — no-safehouse argv = plain codex + FO-skill prompt, NO bypass."
else
  echo "PART A VERDICT: INSPECT — recorded argv differs from the no-safehouse oracle."
fi

echo
echo "============================================================"
echo "PART B — interactive safehouse+codex launch (captain drives by eye)"
echo "------------------------------------------------------------"
echo "In a directory WITH a .safehouse profile and the spacedock plugin installed"
echo "in codex, run:"
echo
echo "    $SPACEDOCK codex"
echo
echo "Expected exec argv (the implementation-locking oracle):"
echo
echo "    safehouse --trust-workdir-config -- codex --dangerously-bypass-approvals-and-sandbox \\"
echo "      \"Invoke the spacedock:first-officer skill: run your startup sequence and work the event loop.\""
echo
echo "Confirm by eye:"
echo "  1. codex banner shows  sandbox: danger-full-access  AND  approval: never"
echo "  2. codex invokes the spacedock:first-officer skill; FO startup begins"
echo "  3. FO can run \`spacedock status\` / dispatch inside the safehouse sandbox"
echo
echo "A non-.safehouse run should launch plain codex (PART A) — no bypass, no hang."
