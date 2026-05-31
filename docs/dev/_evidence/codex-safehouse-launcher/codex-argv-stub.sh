#!/bin/bash
# ABOUTME: argv-recording stub — drop on PATH as `codex` to record the exact argv
# ABOUTME: `spacedock codex` hands the launcher, without launching the real codex.
#
# To observe what `spacedock codex` would exec without launching the interactive
# codex TUI (which would hang), put this file early on PATH named `codex`, then
# run `spacedock codex --foo` in a dir with a `.safehouse` profile (and safehouse
# on PATH). The recorded argv lands in $RECORD (default below). When safehouse
# wraps the launch, safehouse — not this stub — is exec'd first; for the
# inner-argv recording, run WITHOUT a `.safehouse` profile (no-safehouse path) or
# use a safehouse that passes its remainder through to this stub.
#
# argv[0] (the `codex` program name) is recorded first so the recorded file is
# the FULL launched argv, matching the AC-1 Launch-argv oracle.
{ printf 'codex\n'; printf '%s\n' "$@"; } > "${RECORD:-/tmp/spacedock-ac4-evidence/recorded-argv.txt}"
exit 0
