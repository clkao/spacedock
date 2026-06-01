---
id: 5tsqmdd3vtj1s8d8gmf50efj
title: Guarantee the shipped tool runs on anyone's machine — no hidden personal dependencies
status: ideation
source: hx decomposition (C of 3) — captain 2026-06-01; staff review
score: "0.28"
started: 2026-06-01T06:04:55Z
completed:
verdict:
worktree:
issue:
---

**What this is for (plain).** Add a test that fails the moment the shipped tool quietly starts
depending on something only your machine has — your personal config file, a particular program being
installed, or an internal-only file path that won't exist on a fresh install. So a clean install just
works for any user, instead of breaking on a hidden assumption nobody noticed.

**Value to the user / FO.** No "works on my machine" surprises. A new user, a teammate, or a second
checkout gets a tool that runs without first reproducing someone's personal environment — and if a
future change reintroduces such a dependency, the test catches it before it ships.

This is part C of three from the now-superseded parent `deliverable-contract-hardening`
(id `hxs93wd0bjwhc3vsjwx1seew`). This child is the TEST half.

## Scope (ideation hardens; carries the staff-review correction)

- A test (alongside the existing shipped-files tests) that parses the real shipped files and fails if
  the portable surface contains a non-portable dependency: a per-user personal-config dependency, a
  `python`/`python3`-on-PATH requirement in the dispatch path, or an internal-only helper-script path.
  It fails on real drift (a reintroduced internal path or interpreter dependency), not on a missing
  sentence.
- **Staff-review correction (must apply):** scope the test to the *shipped* `skills/` set (reuse the
  existing shipped-files list) and **explicitly exclude this workflow's own guide** (`docs/dev/README.md`)
  — that file is project scaffolding, not part of the shipped plugin, and it *intentionally* contains
  during-migration compatibility commands. Including it would make the test wrong.

## Acceptance criteria (provisional)

**AC-1 — The portable shipped surface has no hidden machine dependency, enforced by a test.**
Verified by: a test over the real shipped `skills/` files asserts the portable surface contains none
of the non-portable dependency markers (personal-config / interpreter-on-PATH / internal helper path);
reintroducing such a marker makes it red; the host-specific runtime-adapter files are correctly
excluded (host-specifics legitimately live there).

## Out of scope
- The code guard (part A) and the prose edits (part B).
- Any cleanup of this workflow's own guide (that file is deliberately out of the portable scope).
