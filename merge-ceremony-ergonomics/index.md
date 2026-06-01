---
id: 3qpv8qv6pbvejajejs2dn3v8
title: Merge-ceremony ergonomics — ship-local combo, honest no-remote fallback, workflow merge-policy
status: ideation
source: issue sweep (2026-05-31) — CL dev-workflow-ergonomics triage; consolidates #223, #217, #225
started: 2026-06-01T05:12:07Z
completed:
verdict:
score: "0.30"
worktree:
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

The state-installed `_mods/pr-merge.md` no-PR fallback (line 100) instructs the FO to clear
`mod-block` *after* the local merge lands, and then claims: "clearing `mod-block` only after the
local merge lands keeps that guard satisfied through the no-PR path." **That claim is false.** The
guard does not look at merge ordering — it looks at the *post-update* state. The FO ceremony per
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
- **FO honoring:** the FO reads `merge:` at boot (it already reads README via `--boot`) and, under
  `merge: local`, runs the local-merge ceremony without ever reaching for `--force`. The shared-core
  Merge-and-Cleanup prose gains a branch: when `merge: local` (or no PR host), terminalize without
  `--force` because the guard now permits it.

### #217 — make the no-PR fallback honest (and fix the false prose)

Two sub-parts:

1. **Fix the contradictory prose immediately** (a bug, fix-on-sight): the `_mods/pr-merge.md`
   fallback claim that clearing `mod-block` "keeps that guard satisfied" is false and must be
   corrected. Under `merge: local` the corrected prose says terminalize succeeds because the policy
   exempts the guard. For workflows that have NOT declared `merge: local`, the fallback sets a
   sentinel before clearing `mod-block`:
   `spacedock status --set {slug} pr=local-merge:{short-sha}` (#217 option 1). The guard then sees
   a non-empty `pr` and is satisfied honestly — `pr` truthfully records that a merge landed, just
   not a remote PR.
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
Verified by: a fixture workflow with `merge: local` + a registered merge hook where the terminal
guard does not demand a PR or `--force`; plus the existing guard fixtures (default policy) continuing
to pass unchanged in both runners.

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

## Test plan

- **Guard behavior (AC-1, AC-3, AC-4):** Go unit/behavioral tests over the terminal-transition guard
  in `internal/status` for: (a) sentinel-satisfied no-PR terminalize succeeds without `--force`;
  (b) no-sentinel/no-policy still refuses (the catch is preserved); (c) `merge: local` policy exempts
  the empty-`pr`/empty-`mod-block` case; (d) `mod-block`-pending still blocks under any policy. Each
  guard test must be mirrored as an oracle-parity assertion (`runNative` vs `runOracle`) — the oracle
  `bin/status` and the Go native runner change together. Cost: moderate — touches the serialized
  `internal/status` lane and the vendored oracle; sequence after `zs` + architecture-review-cleanups.
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
