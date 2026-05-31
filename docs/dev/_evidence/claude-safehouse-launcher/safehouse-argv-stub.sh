#!/bin/bash
# ABOUTME: argv-recording stub (salvaged) — drop on PATH as `safehouse` to record
# ABOUTME: the exact argv `spacedock claude` hands the launcher without a real exec.
#
# Optional helper for AC-6 inspection. To record what `spacedock claude` would
# exec without launching the real safehouse, put this file early on PATH named
# `safehouse`, then run `spacedock claude --foo` in a dir with a `.safehouse`
# profile. The recorded argv lands in $RECORD (default below).
printf '%s\n' "$@" > "${RECORD:-/tmp/spacedock-ac6-evidence/recorded-argv.txt}"
exit 0
