#!/bin/bash
# Stub that records argv to a file so the spike can assert the canonical shape.
printf '%s\n' "$@" > /tmp/spacedock-launcher-spike/recorded-argv.txt
exit 0
