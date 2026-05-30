# Status read goldens

These golden files pin the read-path output of the vendored status oracle for
the `seq-workflow` fixture. They are captured from the oracle (the live status
script) under a pinned locale (`PYTHONUTF8=1`, `LANG=C.UTF-8`) and normalized so
no machine-specific path or wall-clock timestamp is baked in:

- absolute workflow roots are replaced with `<ROOT>` (the `--resolve` `workflow=`
  field is realpath-resolved first to account for the macOS `/var`→`/private/var`
  rewrite; `path=` is not);
- ISO-8601 UTC timestamps (second- and microsecond-precision) become `<TS>`.

## Regenerating

When the vendored script changes, re-capture the goldens from the oracle so the
diff is visible in review rather than silently absorbed:

```
go test ./internal/status -run TestGoldenRead -update
```

## Oracle pin (AC-2 reproducibility)

Goldens were captured under:

- interpreter: `python3` 3.14.x
- locale: `PYTHONUTF8=1`, `LANG=C.UTF-8`, `LC_ALL=C.UTF-8`

Regenerate under the same pinned locale and a compatible interpreter so byte-for-
byte output stays reproducible across machines.
