---
id: 3qpv8qv6pbvejajejs2dn3v8
title: Merge-ceremony ergonomics — ship-local combo, honest no-remote fallback, workflow merge-policy
status: implementation
source: issue sweep (2026-05-31) — CL dev-workflow-ergonomics triage; consolidates #223, #217, #225
started: 2026-06-01T05:12:07Z
completed:
verdict:
score: "0.30"
worktree: .worktrees/spacedock-ensign-merge-ceremony-ergonomics
issue:
---

Collapse the repetitive local-merge ceremony an FO performs at every terminal boundary when a
workflow has no PR host, and stop the merge-hook guard from forcing `--force` on the documented
no-remote fallback. Today an FO running a local (non-PR) merge repeats a multi-step sequence per
entity, and the terminal-transition guard treats the legitimate no-remote fallback as a skipped
hook — so the operator must pass `--force`, which defeats the guard's purpose.

Consolidates three open issues:
- **#225** — a workflow-level merge-policy declaration (e.g. `merge: local | pr`) so the FO and the
  guard stop re-deriving "is there a PR host?" per entity and stop demanding `--force`.
- **#217** — the pr-merge mod's no-remote fallback plus the terminal-transition guard force
  `--force` on every local merge; the fallback should satisfy the guard honestly without `--force`.
  Hit LIVE this session terminalizing `0m` (the no-code pr-merge mod path): the guard forced
  `--force`.
- **#223** — a `ship-local` combo that collapses the ~7-step merge→terminalize→archive→cleanup
  sequence into one guided action for the no-PR path.

## Problem analysis

### The guard, precisely

`status --set` and `status --archive` both enforce the same merge-hook invariant (Go native:
`internal/status/handlers.go:128` and `internal/status/mutate.go:196`; Python oracle:
`internal/status/vendor/status:2431` and `:1914`). The terminal-`--set` form is:

```
if !force && isTerminalUpdate() && modBlock == "" && postUpdatePR == "" && postUpdateVerdict != "rejected" {
    if scanMods(entityDir)["merge"] is non-empty { refuse }
}
```

`isTerminalUpdate()` is true when the update sets `status` to a terminal stage, or sets `completed`,
or sets `verdict`, or clears `worktree`. So the guard fires on a terminal `--set` whenever a `merge`
hook is registered AND `pr` is empty AND `mod-block` is empty. The intent is honest: catch an FO
advancing to terminal when the merge hook never ran.

### #217 is a prose/mechanism contradiction, not just friction

The state-installed `_mods/pr-merge.md` no-PR fallback (the `### Fallback: no PR host available`
section, line 100) instructs the FO to clear `mod-block` *after* the local merge lands, and then
claims: "clearing `mod-block` only after the local merge lands keeps that guard satisfied through the
no-PR path." **That claim is false.** The guard does not look at merge ordering — it looks at the
*post-update* state.

**Which file carries the false claim (pin, to avoid the review file-mixup).** The false claim lives
in the **LIVE state-checkout mod**:
`docs/dev/.spacedock-state/_mods/pr-merge.md`, in its `### Fallback: no PR host available`
section at line 100 (verified this session: the parenthetical "(The mechanism-level guard refuses
terminalizing while `pr` and `mod-block` are both empty and a merge hook is registered; clearing
`mod-block` only after the local merge lands keeps that guard satisfied through the no-PR path.)").
It does NOT live in the code-repo copy `mods/pr-merge.md`, whose fallback prose (lines 49/51) merely
says "fall back to local merge" with no guard-satisfaction claim. A prior staff review reported the
line-100 claim as a false positive because it inspected the code-repo `mods/pr-merge.md` copy rather
than the live state-checkout mod; the claim is **real** in the state-checkout file. The prose fix
targets `docs/dev/.spacedock-state/_mods/pr-merge.md`. **Coordinate with f2.** The
`mods-definition-dir-location` entity (id `f2yr32fgw3pfxp7ekq4wy1np`) relocates this mod from the
state checkout to `docs/dev/_mods/pr-merge.md` (delete-from-state-branch + add-to-main-repo, atomic
with its `scanMods(definitionDir)` swap). If f2 lands first, the corrected fallback prose lands in
`docs/dev/_mods/pr-merge.md`; if this entity's prose fix lands first, it corrects the state-checkout
copy and f2 carries the corrected text across in its move. Either order is fine as long as the false
claim is corrected in whichever copy is live at the time — the two entities touch the same file and
must not silently overwrite each other's edit to that section. The FO ceremony per
the shared core (`first-officer-shared-core.md:227-233`) is: set `mod-block` → local merge → clear
`mod-block` in its OWN standalone `--set` → then terminalize (`completed verdict= worktree=`). At
the terminalize step `mod-block == ""` and `pr == ""`, so the guard fires. There is no merge order
that satisfies it: clearing `mod-block` is mandatory before terminalizing (the guard *also* refuses
combining a `mod-block` clear with terminal fields in one call — `handlers.go:112`), and once
cleared with `pr` empty, the merge-hook invariant trips. The FO's only escape today is `--force` on
both the terminal `--set` and the `--archive` — the exact 18-`--force` live friction #217 reports,
and what this session hit terminalizing `0m`.

So the fallback path as documented *cannot* satisfy the guard. Either the fallback must leave a
truthful non-empty signal that the merge ran (a `pr` sentinel — #217's option 1), or the guard must
learn that this workflow merges locally and stop demanding a PR (a merge-policy — #225). These are
complementary, not competing: the policy is the durable per-workflow declaration; the sentinel is
the per-entity record of which local commit shipped.

### What already exists to build on

- README frontmatter is already parsed in both runners: `resolveRoots` reads `state:`
  (`roots.go:55`), and the stages block is parsed for terminal-stage detection (`handlers.go:72`,
  oracle `:2426`-region). Adding a new top-level `merge:` key is a low-risk extension of a seam that
  is already exercised in both the native runner and the Python oracle.
- `runSet`/`runArchive` already hold `roots.definitionDir` (the README dir) and call `scanMods`, so
  the guard sites can read the README merge-policy without new plumbing.
- The native Go runner is held in lockstep parity with the vendored Python oracle
  (`zz_independent_parity_test.go`, the `runNative` vs `runOracle` harness). **Any guard behavior
  change MUST land in both `bin/status` (the oracle, vendored at `internal/status/vendor/status`)
  and the Go native runner, or the parity suite breaks.** This is the dominant cost/risk of the #225
  and #217-sentinel-display work.

## Proposed approach

Three coordinated changes, sequenced from lowest-coupling to highest:

### #225 — declare merge policy in README frontmatter (RECOMMENDED FORM)

Add an optional README frontmatter key:

```yaml
merge: local        # or: merge: pr  (default when absent)
```

- **Declaration form:** a single top-level `merge:` key, value `local` or `pr`. Chosen over
  #225's verbatim `merge-policy: local-no-ff` because (a) `merge:` matches the existing one-word
  top-level key style (`state:`, `id-style:`), (b) the `--no-ff` detail is a *mechanic* of how the
  FO merges, not a *policy* the guard needs — the guard only needs "is a PR expected here?", and
  (c) the enum stays small and forward-extensible.
- **Default:** absent ⇒ `pr`. This preserves every existing workflow's behavior (the guard keeps
  demanding a PR or mod-block), so no backward-compat shim is needed and no current fixture changes.
- **Guard behavior under `merge: local`:** the merge-hook invariant at the terminal `--set` and
  `--archive` no longer refuses on empty-`pr`/empty-`mod-block`. The `mod-block`-pending guard
  (`handlers.go:112`) is unchanged — an in-flight `mod-block` still blocks terminal transitions
  regardless of policy, because that catches a genuinely interrupted ceremony. Under `merge: local`
  the guard's question becomes "did the FO complete the merge ceremony (mod-block cleared)?" rather
  than "is there a PR?".
- **`merge: local` relaxes the guard CHECK, NOT the ceremony STRUCTURE (the safety invariant).**
  This is the load-bearing distinction and must not be conflated. What `merge: local` changes is the
  *terminal-guard predicate at `handlers.go:128`*: instead of demanding a non-empty `pr` (or a
  satisfied `mod-block`) before permitting a terminal transition when a merge hook is registered, it
  validates that the merge-hook ceremony actually completed via the `mod-block` lifecycle — i.e. the
  guard accepts a cleared `mod-block` as proof the hook ran, without also requiring a `pr`. What
  `merge: local` does NOT change is the *ceremony the FO must perform*: when a `merge` hook is
  registered, the **set `mod-block=merge:{hook}` → invoke the hook → clear `mod-block`** sequence
  stays MANDATORY. `merge: local` does not authorize an FO to skip the hook, nor to manually clear
  `mod-block` and terminalize *without having run the hook*. Concretely: an FO that sets
  `mod-block=merge:pr-merge`, does NOT invoke the hook (no local merge actually performed), then
  clears `mod-block` and terminalizes, has produced a wrongful terminalization — the entity reads
  `done` but no merge landed. `merge: local` must not open that hole. The mechanism cannot, on its
  own, verify a merge physically happened (it only sees the `mod-block` lifecycle); the ceremony
  structure is therefore the load-bearing control, and the sentinel (below) is the preferred
  truthful artifact that records *which* commit shipped, so the cleared-`mod-block` claim is
  backed by an on-entity record rather than by trust alone. The merge-hook invariant's purpose —
  catch a terminal transition when the hook never ran — survives `merge: local` intact; only the
  "must be a PR" framing is relaxed.
- **FO honoring:** the FO reads `merge:` at boot (it already reads README via `--boot`) and, under
  `merge: local`, runs the local-merge ceremony without ever reaching for `--force`. The shared-core
  Merge-and-Cleanup prose gains a branch: when `merge: local` (or no PR host), terminalize without
  `--force` because the guard now permits it.

### #217 — make the no-PR fallback honest (and fix the false prose)

Two sub-parts:

1. **Fix the contradictory prose immediately** (a bug, fix-on-sight): the live state-checkout mod
   `docs/dev/.spacedock-state/_mods/pr-merge.md` `### Fallback: no PR host available` claim (line 100)
   that clearing `mod-block` "keeps that guard satisfied" is false and must be corrected (NOT the
   code-repo `mods/pr-merge.md` copy, which lacks the claim — see the pin in "#217 is a
   prose/mechanism contradiction" above; coordinate the edit with f2's relocation of this file). Under `merge: local` the corrected prose says terminalize succeeds because the policy
   exempts the guard. For workflows that have NOT declared `merge: local`, the fallback sets a
   sentinel:
   `spacedock status --set {slug} pr=local-merge:{short-sha}` (#217 option 1). The guard then sees
   a non-empty `pr` and is satisfied honestly — `pr` truthfully records that a merge landed, just
   not a remote PR.
   **Sentinel ordering (the safe primary path):** the sentinel `pr=local-merge:{short-sha}` is set
   ONLY after the local `--no-ff` merge has truly landed — `{short-sha}` is the SHA of the merge
   commit that exists on `next`, computed after the merge, not before. This ordering is what makes
   the sentinel a *truthful* record: the entity carries a `pr` value if and only if a real commit
   shipped, so a reviewer or a future debugger can resolve `local-merge:{sha}` to an actual merge in
   the log. The corrected sequence is therefore: invoke the hook → local merge lands → set the
   sentinel (records the landed SHA) → clear `mod-block` → terminalize. Setting the sentinel before
   the merge lands would re-introduce the same dishonesty the false prose had (a guard satisfied by
   a signal that does not yet correspond to a merge), so the prose MUST set it post-merge. This
   sentinel-first-after-merge path is the **safe primary** mechanism for un-declared workflows: it
   needs no guard-code change, satisfies the existing guard in both runners, and leaves an
   on-entity artifact tying terminalization to a concrete commit — strictly safer than relying on a
   bare cleared `mod-block`.
2. **Display recognizes the sentinel** (#217 AC-3): the `status` table renders a `pr` value with the
   `local-merge:` prefix as `{short-sha} (local)` rather than as a PR reference. This is the only
   piece that touches display/format code and therefore the parity surface.

Recommendation: ship the **sentinel** as the universal mechanism (works without any README change,
satisfies the existing guard in both runners with no guard-code change), and treat the **policy**
(#225) as the ergonomic layer on top that lets a whole-sprint workflow skip even the sentinel step.
A workflow that declares `merge: local` does not need the sentinel to pass the guard; a workflow
that has not declared it gets the sentinel from the fallback prose. Both reach terminal without
`--force`.

### #223 — ship-local combo: FO-prose-first, defer the status subcommand

Two candidate shapes:

- **(A) FO-prose-only:** codify the merge-local sequence as a single named operation in the
  first-officer shared core ("ship-local ceremony"), so the FO runs the fixed
  merge→clear-mod-block→terminalize→archive→worktree-cleanup steps from one prose block per entity,
  without re-deriving them. No code change; composes directly with the pr-merge fallback prose.
- **(B) `status --ship-local` subcommand:** a new status verb that performs the sequence as one
  invocation (still committing each step for audit trail).

Recommendation: **start with (A), defer (B).** Rationale: once #225 + #217 remove the `--force`
requirement, the residual pain #223 reports is *repetition*, not *force-bypass*. A prose-codified
ceremony solves the repetition for the FO (the actor who runs it) at zero parity cost. A
`--ship-local` subcommand is attractive but (i) it touches `internal/status` handlers/stages — the
serialized lane that must stay in oracle parity — and (ii) it embeds git-merge orchestration
(`git merge --no-ff`, conflict handling, `git worktree remove`, `git branch -d`) into the status
binary, which today never shells out to perform merges. That is a meaningful new responsibility and
risk surface. Recommend (A) now; capture (B) as a follow-up entity if FO-prose proves insufficient
at sprint scale. **Sequence the combo work AFTER `zs` + architecture-review-cleanups** (the
serialized `internal/status` lane), per the dispatch note.

## Acceptance criteria

**AC-1 — A no-PR terminal merge completes without `--force` (the #217 live friction).**
The no-remote fallback path reaches `done` + archived with `mod-block` cleared and NO `--force` on
either the terminal `--set` or the `--archive`. End state: `status=done`, `completed` stamped,
`verdict` set, `worktree` cleared, entity in `_archive/`, no `--force` invoked.
Verified by: a Go behavioral test (mirrored as an oracle-parity test) that constructs an entity in
the no-remote fallback state — merge hook registered, `pr=local-merge:{sha}` sentinel set,
`mod-block` cleared — and asserts the terminal `--set` and `--archive` both exit 0 without `--force`,
and a companion test asserting the SAME entity WITHOUT the sentinel and WITHOUT `merge: local` still
exits 1 (the guard still catches the genuinely-skipped hook).

**AC-2 — The `local-merge:` sentinel renders distinctly in status output.**
A `pr` field of `local-merge:{short-sha}` displays as `{short-sha} (local)` in the `status` table,
distinguishable from a real PR reference.
Verified by: a golden/snapshot test of `status` table output (native + oracle parity) over a fixture
entity carrying the sentinel.

**AC-3 — A workflow can declare `merge: local`; the guard and FO honor it.**
A README declaring `merge: local` lets the terminal `--set` and `--archive` succeed with empty `pr`
and empty `mod-block` (no `--force`), while the `mod-block`-pending guard still blocks if `mod-block`
is non-empty. Absent the key (default `pr`), behavior is byte-identical to today.
Verified by: a NEW fixture workflow with `merge: local` **and a registered merge hook** (a `_mods/`
dir containing a file with `## Hook: merge`) where the terminal guard does not demand a PR or
`--force`; plus the existing guard fixtures (default policy) continuing to pass unchanged in both
runners. The registered-hook part is mandatory — without a `_mods/` dir `scanMods` returns empty and
the merge-hook branch is never reached, so the fixture would pass vacuously (see "Test fixtures"
below).

**AC-4 — Existing GitHub-PR-shaped runs and the parity suite are unaffected.**
The PR path (push → `gh pr create` → `pr=#N` → block → detect-merge → terminalize) is unchanged, and
`zz_independent_parity_test.go` plus the existing guard fixtures pass against both native and oracle.
Verified by: the existing parity and guard test suites passing with no fixture edits beyond the new
`merge: local` fixture.

**AC-5 — One guided ship-local ceremony is documented for the FO (the #223 collapse).**
The first-officer shared core carries a single named ship-local ceremony block that an FO runs per
entity for the no-PR path without re-deriving the step sequence and without `--force`.
Verified by: a static skill/prose check that the ceremony block exists, references the merge-policy
branch, and contains no `--force` in the documented happy path; the runtime collapse is exercised by
the AC-1 behavioral test (the steps the ceremony names).

## Test fixtures

**A NEW fixture with a REGISTERED merge hook is required — the existing guard fixture cannot
exercise the invariant this entity relaxes.** Verified this session: the only guard fixture is
`internal/status/testdata/guard-workflow/` (a `README.md` + `010-blocked.md` with
`mod-block: merge:pr-merge`), and there is **no `_mods/` directory anywhere under
`internal/status/testdata/`**. Because `scanMods(entityDir)` (`mutate.go:282`) returns an empty map
when no `_mods/` dir exists, the merge-hook branch at `handlers.go:128-135` — `len(mergeHooks) > 0`
— is **never reached** by the current fixtures. Today's `010-blocked.md` only exercises the
`mod-block`-pending guard (`handlers.go:112`), NOT the merge-hook-unsatisfied guard that line 128
implements and that `merge: local` relaxes. A test written against `guard-workflow` would pass
whether or not the relaxation is correct — a vacuous pass.

The new fixture must therefore include:

- a workflow `README.md` declaring `merge: local` (plus a sibling default-policy variant or reuse of
  `guard-workflow` for the absent-key case), and
- a `_mods/` directory containing at least one mod file with a `## Hook: merge` heading, so
  `scanMods` returns a non-empty `["merge"]` and the line-128 branch actually fires. A minimal stub
  (`name:`/`description:` frontmatter + a `## Hook: merge` section) is sufficient — the guard reads
  only the hook registration, not the hook body.

This single fixture (registered merge hook, `merge: local`) is what lets the guard tests below
distinguish "guard correctly relaxed" from "guard never consulted." The companion negative cases
(no-sentinel, no-policy, `mod-block`-pending) run against the SAME registered-hook fixture so the
hook is present in every case and only the `pr`/`mod-block`/policy state varies.

**Single-root, for oracle parity.** The fixture must be single-root (the `_mods/` dir lives
alongside the README and entities, `definitionDir == entityDir`) so the guard tests can run as
`runNative`-vs-`runOracle` parity assertions. The Python oracle has no split-root concept and
resolves `scan_mods(pipeline_dir)` from its single dir (established by the `mods-definition-dir-location`
entity); a single-root fixture resolves `_mods/` identically in both runners, keeping the parity
suite green. Do not make this fixture split-root — that would diverge native from oracle on mod
resolution and force the merge-hook guard tests to native-only, losing the parity coverage AC-4
requires.

## Test plan

- **Guard behavior (AC-1, AC-3, AC-4):** Go unit/behavioral tests over the terminal-transition guard
  in `internal/status`, **all run against the new registered-merge-hook fixture** (see "Test
  fixtures" above — without a registered `## Hook: merge` mod the merge-hook branch is never reached
  and the tests pass vacuously), for: (a) sentinel-satisfied no-PR terminalize succeeds without
  `--force`; (b) no-sentinel/no-policy still refuses (the catch is preserved); (c) `merge: local`
  policy exempts the empty-`pr`/empty-`mod-block` case; (d) `mod-block`-pending still blocks under
  any policy. Each guard test must be mirrored as an oracle-parity assertion (`runNative` vs
  `runOracle`) — the oracle `bin/status` and the Go native runner change together. Cost: moderate —
  touches the serialized `internal/status` lane and the vendored oracle; sequence after `zs` +
  architecture-review-cleanups.
- **Display (AC-2):** golden/snapshot test of the `status` table with a `local-merge:` sentinel,
  native + oracle parity. Cost: low.
- **Policy parsing:** unit test that `merge: local` / `merge: pr` / absent parse correctly from README
  frontmatter and that an unknown value is rejected loudly (not silently treated as `pr`). Cost: low.
- **FO prose (AC-5):** static prose check that the shared-core ship-local ceremony block exists and
  the corrected pr-merge fallback prose no longer claims the false "keeps the guard satisfied"
  guarantee. Cost: low. (No live workflow test required for AC-5 — the runtime claim is covered by
  AC-1; the prose AC is a text-shape claim and uses a text-shape proof.)
- **Composition checks (ideation confirms):**
  - PR path unchanged: the `pr=#N` block-then-detect flow does not touch the new code paths.
  - Split-root: all state writes (entity body, sentinel `pr`, terminal fields) stay path-scoped to
    the `.spacedock-state` checkout; the combo/ceremony writes nothing to `main` beyond the existing
    `pr:` mirror.
  - The sentinel and the policy compose: `merge: local` makes the sentinel optional for guard
    satisfaction; the sentinel is the fallback for un-declared workflows. Neither breaks the other.

## Sequencing and staff review

- **Sequence:** the #217-sentinel prose fix and the #225 policy-parse are independently shippable.
  The guard-code change (#225) and the display change (#217 AC-2) touch `internal/status`
  (handlers.go / stages.go / format.go) and the vendored oracle — the serialized lane — so they
  sequence AFTER `zs` + architecture-review-cleanups. The #223 FO-prose ceremony (recommendation A)
  composes with the corrected pr-merge fallback prose and can land alongside the prose fixes.
- **Staff review: WARRANTED.** This crosses the oracle/native parity boundary, changes a security-
  relevant guard (the merge-hook invariant exists to prevent skipped merges — relaxing it must not
  open a hole), introduces a new README declaration with a default that must not regress existing
  workflows, and composes three issues plus the pr-merge mod and split-root. Per the ideation stage
  definition's staff-review trigger (split-root behavior + skill integration + guard design), an
  independent review of design soundness, the guard-relaxation invariant, and test-plan sufficiency
  should precede the ideation gate.

## Out of scope

- Eliminating the guard. The merge-hook invariant still catches the real mistake (terminalizing when
  the hook never ran); every relaxation here is gated on a truthful signal (sentinel, declared
  policy, or completed mod-block).
- Auto-detecting "no remote" and switching to local-merge silently. The captain still explicitly
  chooses local vs PR — via the `merge:` declaration or the pr-merge fallback's explicit path.
- The `status --ship-local` subcommand (#223 option B) — deferred to a follow-up entity if the
  FO-prose ceremony proves insufficient at sprint scale.

## Stage Report: ideation

- DONE: Decide how a workflow declares its merge-policy (recommend a README frontmatter key, e.g. merge: local|pr) so the FO and the terminal-transition guard stop re-deriving is-there-a-PR-host per entity and stop demanding --force on the documented no-remote local merge. This is issue #225 + the root of #217. Recommend the declaration form + default.
  Recommended a single top-level README key `merge: local | pr`, default `pr` when absent (preserves all current workflows, no compat shim). Chosen over `merge-policy: local-no-ff` because the guard only needs "is a PR expected?", not the `--no-ff` mechanic. Both runners already parse README frontmatter (`roots.go:55`, stages block) so it is a low-risk seam extension. See "#225" section + AC-3.
- DONE: Resolve #217 (the live friction the FO HIT this session): the terminal-transition guard refuses terminalization when a merge hook is registered AND pr is empty AND mod-block is empty, forcing --force on the no-PR fallback. Specify the guard fix so the documented no-PR-host fallback (mod-block set then cleared after the local merge lands) satisfies the guard WITHOUT --force. Behavioral AC: a no-PR-host terminal merge reaches done+archived with mod-block cleared and NO --force.
  Found the root cause is a prose/mechanism CONTRADICTION: `_mods/pr-merge.md:100` claims clearing mod-block after the merge "keeps that guard satisfied" but the guard (`handlers.go:128`) checks post-update state, not merge order — so clearing mod-block then terminalizing always trips it. Fix: fallback sets `pr=local-merge:{short-sha}` sentinel before clearing mod-block (#217 option 1) so `pr` truthfully records the merge; under declared `merge: local` the policy exempts the guard entirely. Behavioral AC-1 specified.
- DONE: Decide the ship-local combo (#223): a new status subcommand vs FO-prose-only, that collapses the local-merge to terminalize to archive to cleanup sequence into one guided action. Recommend; specify the behavioral AC. NOTE the impl touches internal/status (handlers.go/stages.go) — the serialized lane — so sequence AFTER zs + architecture-review-cleanups; and it composes with the pr-merge mod fallback prose. State whether staff review is warranted.
  Recommended FO-prose-only (option A) now, defer the `status --ship-local` subcommand (option B). Once #225/#217 remove the `--force` need, the residual #223 pain is repetition, not force-bypass; prose solves it at zero parity cost and avoids embedding git-merge orchestration into the status binary. Sequenced AFTER zs + architecture-review-cleanups. Staff review: WARRANTED (oracle/native parity boundary, guard-relaxation invariant, new README default, split-root + skill composition). AC-5 specified.

### Summary

Consolidated #225/#217/#223 into one coherent design: a `merge: local|pr` README key (default `pr`) is the durable policy layer; a `pr=local-merge:{sha}` sentinel is the per-entity honest merge record for un-declared workflows; an FO-prose ship-local ceremony collapses the repetition. Key finding: #217's live `--force` friction is rooted in a FALSE claim in `_mods/pr-merge.md:100` — clearing mod-block does not satisfy the guard, which checks post-update `pr`/`mod-block` state regardless of merge order. The guard relaxation must land in BOTH the Go native runner and the vendored Python oracle to keep `zz_independent_parity_test.go` green; that parity cost, plus the security-relevant guard relaxation and the new README default, is why staff review is recommended. Impl touches the serialized `internal/status` lane (handlers.go/stages.go/format.go) and sequences after zs + architecture-review-cleanups.

## Stage Report: ideation (cycle 2)

- DONE: [M-1, the safety-critical one] Close the merge:local guard-relaxation ambiguity. State EXPLICITLY: merge:local changes only the guard CHECK (it exempts the pr-field requirement, validating instead that the merge-hook ceremony completed via mod-block), NOT the ceremony STRUCTURE — the mod-block set then invoke-hook then clear ceremony stays MANDATORY when a merge hook is registered. An FO must NOT manually clear mod-block and terminalize without running the hook under merge:local. Reaffirm the sentinel path (pr=local-merge:{sha} set ONLY after the local merge truly lands) as the safe primary. Net: no wrongful-terminalization path.
  Added the "relaxes the guard CHECK, NOT the ceremony STRUCTURE" bullet under #225 (it pins handlers.go:128 as the predicate that changes, names the wrongful-terminalization scenario explicitly — set mod-block, skip hook, clear, terminalize → entity reads done with no merge — and states merge:local must not authorize it). Reaffirmed the sentinel as the safe primary in #217 sub-part 1: sentinel set ONLY post-merge with the landed merge-commit SHA, sequence invoke→merge-lands→set-sentinel→clear-mod-block→terminalize, noting the mechanism cannot verify a physical merge so ceremony structure is the load-bearing control.
- DONE: [M-2 clarification] Pin the false-claim reference precisely: it is the LIVE state-checkout mod docs/dev/.spacedock-state/_mods/pr-merge.md (the `### Fallback: no PR host available` section, ~line 100, which claims clearing mod-block keeps the guard satisfied — false, the guard checks post-update state). The reviewer looked at a different/stale code-repo copy (mods/pr-merge.md); note that the live state-checkout mod is the one with the claim, and that the f2 (mods-definition-dir) entity relocates it — coordinate the prose fix with f2's move.
  Verified the file-mixup directly: docs/dev/.spacedock-state/_mods/pr-merge.md:100 carries the verbatim false claim; the code-repo mods/pr-merge.md (lines 49/51) has no guard-satisfaction claim at all. Pinned both the problem-analysis reference and the #217 fix sub-part to the state-checkout file, noted the reviewer inspected the code-repo copy, and added the f2 (mods-definition-dir-location, id f2yr32fgw3pfxp7ekq4wy1np) coordination note — f2 relocates this mod to docs/dev/_mods/ as delete-state + add-main, either landing order is fine as long as the correction lands in whichever copy is live.
- DONE: [M-3] Specify that the guard tests need a NEW fixture with a REGISTERED merge hook (a _mods/ dir with a `## Hook: merge`) to exercise the merge-hook-unsatisfied guard — the existing guard-workflow fixture only has mod-block-pending and no _mods/, so it never tests the merge-hook invariant this entity relaxes. Keep the merge:pr default + parity (Go + Python oracle) requirements.
  Confirmed live: the only guard fixture is internal/status/testdata/guard-workflow (README + 010-blocked.md with mod-block: merge:pr-merge), and NO _mods/ dir exists anywhere under internal/status/testdata — so scanMods returns empty and the handlers.go:128 merge-hook branch is never reached today (a test there would pass vacuously). Added a "Test fixtures" section specifying a NEW single-root fixture with merge: local + a _mods/ dir carrying a `## Hook: merge` mod, tightened AC-3's "Verified by" and the guard-behavior test-plan bullet to require all guard tests run against it, and pinned single-root so the runNative-vs-runOracle parity holds. merge:pr default (AC-3) + Go/oracle parity (AC-4) requirements left intact.

### Summary

Rework of the PASS-WITH-CONCERNS ideation — design unchanged (merge: local|pr default pr, the sentinel, FO-prose ship-local all stand), three findings closed. M-1 (safety-critical): added an explicit invariant that merge:local relaxes only the guard CHECK at handlers.go:128 (exempts the pr requirement, validates the mod-block-cleared ceremony completed), never the MANDATORY set→invoke→clear ceremony structure, naming the wrongful-terminalization scenario it must not open; reaffirmed the sentinel set post-merge as the safe primary. M-2: verified the file-mixup (the false line-100 claim is in the LIVE state-checkout mod, not the code-repo copy the reviewer checked) and pinned both references plus the f2 relocation-coordination note. M-3: confirmed no _mods/ fixture exists, so the merge-hook branch is currently untested, and specified a NEW single-root fixture with a registered `## Hook: merge` mod that the guard tests must use, preserving the merge:pr default + Go/oracle parity requirements.

## Stage Report: implementation

- DONE: [SAFETY-CRITICAL] merge-policy guard relaxation (#225): add an optional README `merge: local|pr` key (default `pr`, absent = byte-identical to today). Under `merge: local` the terminal guard exempts the pr-requirement, validating instead that the mod-block-cleared ceremony completed — WITHOUT opening a wrongful-terminalization hole. Parity-paired Go + Python oracle. Add a NEW single-root fixture with a registered `## Hook: merge` mod + negative cases.
  `resolveMergePolicy` in roots.go + `resolve_merge_policy` in the oracle; guard at handlers.go:152 and mutate.go:220 (+ oracle :2606/:2071) now skips the pr-requirement only under `merge: local`. The mod-block-pending and combined-clear guards are untouched, so set→invoke→clear stays MANDATORY. NEW single-root `merge-local-workflow` (+ default-policy `merge-pr-workflow`, `merge-bogus-workflow`) each carry a `## Hook: merge` stub mod so the branch fires non-vacuously. Code commit b01b60df.
- DONE: #217 honest no-PR fallback: the `pr=local-merge:{short-sha}` sentinel satisfies the existing guard with NO --force; the status table renders it as `{short-sha} (local)` (parity-paired). AND fix the FALSE fallback prose in the RELOCATED docs/dev/_mods/pr-merge.md:100.
  `formatPRCell`/`formatColumnCell` (format.go) + `format_pr_cell`/`format_column_cell` (oracle) map `local-merge:abc1234` → `abc1234 (local)` in the status + next tables (JSON keeps the raw value). pr-merge.md fallback rewritten: removed the false "keeps that guard satisfied" claim, documented the merge:local exemption AND the post-merge sentinel (set AFTER the merge lands). TestSentinelDisplaysAsLocal + TestPRMergeFallbackProseIsHonest cover both.
- DONE: #223 FO-prose ship-local ceremony (option A): add ONE named ship-local ceremony block to first-officer-shared-core.md, no --force in the happy path, referencing the merge-policy branch (AC-5).
  Added `### Ship-Local Ceremony` (steps 1-7: set mod-block → invoke hook → record via policy-exemption or post-merge sentinel → standalone clear → terminalize → archive → cleanup) + a one-line policy note on the Mod-Block-Enforcement mechanism bullet. TestShipLocalCeremonyBlockExists asserts the block exists, references `merge: local`, names the sentinel, and instructs no affirmative --force.

### Summary

Shipped #225/#217/#223 as one coherent change, parity-paired across the Go native runner and both Python oracle copies (vendor/status and skills/commission/bin/status kept byte-identical). The safety-critical M-1 distinction is enforced mechanically: `merge: local` exempts ONLY the pr-requirement of the terminal merge-hook guard; the mod-block-pending and combined-clear-with-terminal refusals are policy-independent, so the mandatory set→invoke→clear ceremony cannot be collapsed under any policy — verified end-to-end (full ceremony succeeds without --force; combined clear+terminalize and default-policy no-sentinel both still refuse). The `pr=local-merge:{sha}` sentinel is the truthful artifact for un-declared workflows. Folded in the next-suppressed-by doc-comment correction (it CAN surface under `--all-fields --where`, parity-symmetric). Tests: 311 status-pkg + 24 integration green; new fixtures validate VALID. NON-regression: the env-only internal/cli TestCodexResolveManifestAgainstInstalledHost fails on `codex plugin list` (PATH/host issue, untouched by this work). High-stakes guard relaxation → expect the detached adversarial audit at validation to probe the wrongful-terminalization hole (held by ceremony structure + sentinel, not a mechanism merge-detector — which is impossible).
