---
id: 5tsqmdd3vtj1s8d8gmf50efj
title: Guarantee the shipped tool runs on anyone's machine — no hidden personal dependencies
status: validation
source: hx decomposition (C of 3) — captain 2026-06-01; staff review
score: "0.28"
started: 2026-06-01T06:04:55Z
completed:
verdict:
worktree: .worktrees/spacedock-ensign-no-hidden-machine-dependencies
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

## Ideation design (2026-05-31)

### What zs #246 settled (the boundary this test depends on)

zs `claude-runtime-segregation` (#246, archived PASSED, on `next` — the base every implementation
worktree this sprint is cut from) established the host-coupling boundary this entity rests on:

- **Generic core is host-neutral.** `internal/dispatch` + `internal/status` carry no `~/.claude` read
  (code-side `go/parser` oracle `hostneutrality.TestNoClaudeHomeReadsInGenericPackages`). The generic FO
  operating contract `first-officer-shared-core.md` carries no unqualified Claude-only helper token
  (prose-side oracle `hostneutrality.TestSharedCoreHasNoUnqualifiedClaudeHelpers`).
- **Claude coupling is quarantined.** All `~/.claude` reads live in `internal/claudeteam`; all
  Claude-only commands/mechanics live in the `claude-*-runtime.md` host adapters.
- **The steady-state FO loop is Python-free.** The FO reuse/feedback/standing paths now call the native
  `spacedock dispatch context-budget|list-standing|show-standing|spawn-standing` — verified: on `next`
  there is **zero** `python3`/`commission/bin` reference anywhere in the shipped `skills/`+`mods/` `.md`
  surface (it was `claude-first-officer-runtime.md:160 → {project_root}/skills/commission/bin/claude-team
  context-budget` on `main`; #246 moved it to the native binary).

**What #246's oracles do NOT cover, and why part C exists.** #246 polices the *Go source* and the
*generic FO contract prose*. It does NOT police the full shipped *instruction surface* (all of `skills/`
+ `mods/`) for a reintroduced personal-config / interpreter-on-PATH / internal-helper-path dependency in
the shipped text a clean-room user actually reads and follows. Part C is that third, complementary
oracle — same spirit as #246's two, one altitude up (the shipped instruction surface, not the binary).

### The portable-vs-host-specific exclusion set (checklist item 2)

The non-portable markers and what each one is, verified against the real `next` surface:

| marker | non-portable form (RED) | legitimate/portable form (NOT flagged) |
|---|---|---|
| **personal config** | a HOME-rooted `~/.claude` / `os.UserHomeDir`+`.claude` / `$HOME`+`.claude` read, or a global `~/.claude/CLAUDE.md` dependency | a **project-relative** `.claude/agents/...` or `.claude/worktrees/...` path (exists in any checkout) — commission/refit/debrief use these legitimately |
| **interpreter on PATH** | a `python`/`python3` shell-out or a `commission/bin/{status,claude-team}` invocation on the FO/ensign critical path | (none — #246 removed the last one; the Python files still ship as the parity oracle but the shipped *instructions* no longer name them) |
| **internal helper path** | an absolute plugin-private path baked into shipped instructions — `skills/commission/bin/status`, `{spacedock_plugin_dir}`, `.agents/plugins/marketplace.json` | the user-facing `spacedock <verb>` binary surface (a published command, not an internal path) |

**The exclusion is structural, not a deny-list — and it is two-layered:**

1. **Scope = `shippedSkillText`** (the existing helper in `skill_surface_test.go`): it walks `skills/**/*.md`
   (skipping the test-only `skills/integration/`) plus the top-level `mods/`. `docs/dev/README.md` — this
   workflow's own guide that *intentionally* carries during-migration `python3`/`commission/bin` commands
   (verified: 3 such references) — lives under `docs/dev/`, which is **outside both walk roots**. So the
   staff-review correction is satisfied by the scope itself: the guide is never read, no explicit
   exclusion sentence needed. This is the correct mechanism — same as how `skills/integration/` is
   excluded by being SKILL.md-less rather than by an allow-list.
2. **Host-adapter exclusion for the `~/.claude` marker only.** `~/.claude/teams/` is a *legitimate*
   host-specific read inside the Claude runtime adapters (`claude-first-officer-runtime.md`,
   `claude-ensign-runtime.md`) — that is exactly where #246 says it belongs. The test excludes the
   `claude-*-runtime.md` adapter files from the `~/.claude` check (and only that check). The
   interpreter-on-PATH and internal-helper-path checks apply to ALL shipped files including the adapters
   (an adapter may name `~/.claude`, but it must still not shell to `python3` or bake a plugin-private
   path — #246 already proved both clean, this pins them).

**Why HOME-rooted vs project-relative is the discriminator that prevents false positives.** A naive
`grep ".claude"` would flag commission's `{project_root}/.claude/agents/{agent}.md` (portable — every
checkout has a `.claude/agents/` if it uses agents) and debrief's `.claude/worktrees/` prune note
(portable). The personal-config marker must match only the HOME-rooted form: `~/.claude`, `$HOME` +
`.claude`, or `os.UserHomeDir` + `.claude`. That is the form that says "depends on the running user's
home directory layout"; the project-relative form does not.

### Coordination with se (`ship-working-principles-in-contract`, part B, in implementation now)

`skills/integration/skill_surface_test.go` holds the shared file-list helper `shippedSkillText` (and
`skillsRoot`/`repoRoot`/`frontmatter`). se's part B edits shipped *instruction text* (adds the four
principles, the "no hidden dependencies" principle, the FO posture, the test-first rule). It does NOT
edit `skill_surface_test.go` (verified: se's worktree copy is byte-identical to `next`).

**Decision: SHARE the helper, do NOT duplicate.** Add part C's test as a **new file**
`skills/integration/portability_test.go` in the same `package integration`, calling the existing
`shippedSkillText(t, skillsRoot(t), repoRoot(t))`. Rationale: duplicating the walk helper would violate
the de-dup rule and risk drift between two copies of "what ships"; a new file in the same package shares
the helper with zero edit to `skill_surface_test.go`, so there is **no file-level collision** with se —
se edits `.md` files, part C adds a `.go` test file, neither touches the other's lines. The only ordering
note: part C's test asserts a *property of the shipped text*; if se's part B adds the literal phrase "no
hidden dependencies" as prose, that phrase is not a marker (it is documentation of the rule, not a
dependency), so the two are independent — part C's RED set is interpreter/personal-config/internal-path
*usages*, not the word "dependency". No sequencing constraint; they can land in either order.

### Staff review

**Warranted — recommend YES.** Per the stage definition's staff-review trigger (skill integration, and
"the test should flag a hidden dependency without false-positiving on the legitimately-host-specific" is
exactly the kind of subtle scope judgment a reviewer should sanity-check). The review should confirm:
(1) the HOME-rooted-vs-project-relative discriminator does not false-positive on commission/refit/debrief's
project-relative `.claude/` paths; (2) the host-adapter exclusion is scoped to the `~/.claude` check only
(adapters are still policed for python/internal-path); (3) the `shippedSkillText`-scope exclusion of
`docs/dev/README.md` is genuinely structural (the guide is outside both walk roots) and not a fragile
deny-list; (4) the test is falsifiable — reintroducing each marker makes it RED, and it is not a tautology
that passes on an empty match.

## Acceptance criteria

Each AC is an end-state property of the finished entity, falsifiable by an oracle. No doc-only ACs.

**AC-1 — The shipped instruction surface carries no hidden machine dependency, enforced by a falsifiable
test.**
End state: a Go test over the real shipped surface (`shippedSkillText` = `skills/**/*.md` excl.
`integration/`, plus `mods/`) asserts that surface contains none of the three non-portable markers — a
HOME-rooted personal-config read (`~/.claude`, `$HOME`/`os.UserHomeDir`+`.claude`, global
`~/.claude/CLAUDE.md`), a `python`/`python3`/`commission/bin` interpreter-on-PATH shell-out, or a
plugin-private internal-helper path (`skills/commission/bin/status`, `{spacedock_plugin_dir}`,
`.agents/plugins/marketplace.json`).
Verified by: the test is GREEN on the `next` surface (already host-neutral); and it is FALSIFIABLE — a
spike (in the validation stage) reintroduces each marker into a shipped file and confirms the test goes
RED naming the file, then reverts to GREEN. The test is not a tautology: it walks a non-empty file set
(asserts `len(shippedSkillText(...)) > 0`) so a future scope bug that empties the walk fails loudly
rather than passing vacuously.

**AC-2 — The test discriminates non-portable from legitimately-host-specific without false positives.**
End state: the `~/.claude` personal-config check matches only the HOME-rooted form and is NOT triggered
by the project-relative `.claude/agents/...` / `.claude/worktrees/...` paths the shipped skills
legitimately use; the `~/.claude` check excludes the `claude-*-runtime.md` host-adapter files (where a
`~/.claude/teams` read is the legitimate Claude coupling per #246); the interpreter-on-PATH and
internal-helper-path checks apply to ALL shipped files including the adapters; and `docs/dev/README.md`
(this workflow's guide, carrying deliberate migration commands) is outside the walk and never read.
Verified by: a positive-control assertion in the same test — the real commission/refit/debrief project-
relative `.claude/` references and the adapter's legitimate `~/.claude/teams` read are present in the
walked surface yet the test is GREEN (proving the discriminator, not just an absence); and a spike that
adds a project-relative `.claude/foo` path to a shipped file leaves the test GREEN while a HOME-rooted
`~/.claude/foo` read in a non-adapter file makes it RED.

### Test plan

- **Proof level.** Static Go test in `skills/integration/` (the right altitude: the claim is about the
  *text* of shipped instructions, so a structural text oracle over the real files is the direct proof —
  not a runtime/live test). One new file `skills/integration/portability_test.go`, `package integration`,
  reusing `shippedSkillText`/`skillsRoot`/`repoRoot`. No fixture tree, no CLI, no live workflow.
- **Cost/complexity.** Low. ~80-120 lines: a marker table (HOME-rooted-`.claude` regex; `python`/`python3`/
  `commission/bin` substrings; the three plugin-private path substrings), the adapter-exclusion predicate
  for the `~/.claude` check, the non-empty-walk guard, and the positive-control assertion. Runs in the
  existing `go test ./skills/integration/` pass in well under a second.
- **What verifies it (the implementation must satisfy):** (a) GREEN on `next`; (b) RED-on-reintroduction
  for each of the three markers (validation-stage falsification spike — add marker, see RED naming the
  file, revert); (c) GREEN with the project-relative `.claude/` positive controls present; (d) the
  non-empty-walk guard fires if the file set is ever empty.
- **Falsifiability is a first-class deliverable**, not an afterthought: the validation stage MUST run the
  reintroduce-each-marker spike and record the RED→revert→GREEN evidence, mirroring how #246's two
  host-neutrality oracles were each shown RED-without-fix.

## Out of scope
- The code guard (part A) and the prose edits (part B, se's `ship-working-principles-in-contract`).
- Any cleanup of this workflow's own guide `docs/dev/README.md` (deliberately outside the portable scope;
  it carries during-migration compatibility commands by design — it is already outside the test's walk).
- Re-policing what #246 already covers (generic Go source host-neutrality, generic FO contract prose) —
  part C is the complementary shipped-instruction-surface oracle, not a duplicate of those.

## Stage Report: ideation

- DONE: Now that zs #246 settled the host-coupling boundary, design the portability test scoped to skills/: assert the shipped tool/contract assumes NOTHING from the user's environment — no global CLAUDE.md, no python3-on-PATH for the FO loop, no plugin-private absolute paths, no personal config/keychain assumptions baked into shipped instructions.
  `## Ideation design` → designed a new `skills/integration/portability_test.go` (package `integration`) reusing the existing `shippedSkillText` walk; three marker checks (HOME-rooted `~/.claude`/`$HOME`+`.claude` personal-config; `python`/`python3`/`commission/bin` interpreter-on-PATH; plugin-private `{spacedock_plugin_dir}`/`skills/commission/bin/status`/`.agents/plugins/marketplace.json` internal paths). Verified the `next` shipped surface is already GREEN (zero python3/commission/bin in shipped `.md`).
- DONE: Define the host-specific-vs-portable exclusion set given zs's boundary — flag a hidden personal/internal dependency without false-positiving on the legitimately-host-specific.
  Exclusion is two-layered and structural (not a deny-list): (1) scope = `shippedSkillText` walks `skills/**/*.md`+`mods/`, so `docs/dev/README.md` (the guide with deliberate migration commands, 3 verified) is outside both walk roots — staff-review correction satisfied by scope itself; (2) the `~/.claude` check alone excludes the `claude-*-runtime.md` adapters (legitimate Claude `~/.claude/teams` coupling per #246) and matches only HOME-rooted form, NOT project-relative `.claude/agents`/`.claude/worktrees` that commission/refit/debrief use. Captured as the marker table in `## Ideation design` and AC-2.
- DONE: Behavioral AC + test plan; note the skill_surface_test.go sharing question with se; state whether staff review is warranted.
  AC-1 (no hidden machine dependency, falsifiable + non-empty-walk guard) and AC-2 (discriminates without false positives, with positive controls) written as end-state properties, each with its oracle. Test plan: low-cost static Go test, ~80-120 lines, GREEN-on-next + RED-on-reintroduction-per-marker (validation-stage falsification spike) + project-relative positive controls. Coordination DECIDED: SHARE the helper via a NEW file, do not duplicate — se's part B edits `.md`, part C adds a `.go` file, no collision (se's worktree `skill_surface_test.go` confirmed byte-identical to `next`). Staff review: recommend YES, with four review focuses.

### Summary

Designed the part-C portability oracle as a new `skills/integration/portability_test.go` that reuses the existing `shippedSkillText` walk and asserts the shipped instruction surface (`skills/**/*.md` + `mods/`) carries none of three non-portable markers: a HOME-rooted personal-config read, a `python`/`commission/bin` interpreter-on-PATH shell-out, or a plugin-private internal path. The hard part — discriminating non-portable from legitimately-host-specific — is solved structurally: the `shippedSkillText` scope leaves `docs/dev/README.md` (the guide with intentional migration commands) out of the walk entirely (staff-correction satisfied by scope, no deny-list), and the `~/.claude` check excludes only the `claude-*-runtime.md` adapters and matches only the HOME-rooted form so project-relative `.claude/agents`/`.claude/worktrees` paths are not false-positived. Verified against the real `next` surface (the base every impl worktree this sprint is cut from): it is already GREEN and contains zero python3/commission/bin in shipped `.md`, so the test pins the host-neutral state #246 achieved at the shipped-instruction altitude. Decided to share the helper (new `.go` file, no edit to `skill_surface_test.go`) rather than duplicate, avoiding any file collision with se's concurrent part-B prose edits. Recommending an independent staff review. No worktree (state-checkout ideation); no code shipped this stage — falsifiability evidence (RED-per-marker) is a named validation-stage deliverable.

## Stage Report: implementation

- DONE: AC-1: implement skills/integration/portability_test.go (package integration) that walks the shipped instruction surface (skills/**/*.md, integration/, mods/) and asserts it carries NO hidden machine dependency — no global ~/CLAUDE.md / $HOME-rooted personal config, no python3-on-PATH assumption for the FO loop, no plugin-private absolute paths (the commission/bin substrings), no personal keychain assumption baked into shipped instructions.
  Code commit `41b6eb63` on branch `spacedock-ensign/no-hidden-machine-dependencies`. `TestShippedSurfaceHasNoHiddenMachineDependency` reuses `shippedSkillText(t, skillsRoot(t), repoRoot(t))` and asserts the three markers (HOME-rooted `~/.claude`/`$HOME`/`os.UserHomeDir`+`.claude`; `\bpython3?\b`/`commission/bin`; the three plugin-private path substrings) are absent; non-empty-walk guard prevents vacuous pass. GREEN on `next`.
- DONE: AC-2: the test DISCRIMINATES non-portable (RED) from legitimately host-specific (NOT flagged) per the designed exclusion set — the ~/.claude marker exclusion is scoped to the Claude adapter only (adapters still policed for python/internal-path); project-relative `.claude/` paths (e.g. commission's {project_root}/.claude/agents) are NOT flagged. Verify the discriminator with a RED fixture (a planted hidden-dep) and a portable control.
  `isClaudeAdapter` excludes `claude-*-runtime.md` for the personal-config check only; interpreter/internal-path apply to all files. `TestPortabilityCheckDiscriminatesHostSpecific` asserts the adapter's `~/.claude/teams` read AND project-relative `.claude/agents|worktrees` (commission/refit/debrief) are present yet GREEN. Falsification spike: planted `~/.claude` (RED), `python3` (RED), `{spacedock_plugin_dir}/commission/bin/status` (RED), `python3` in adapter (RED); planted project-relative `.claude/foo` (GREEN) and adapter `~/.claude` (GREEN); all reverted via git checkout.
- DONE: REUSE skill_surface_test.go's file-list helper READ-ONLY (zero edits to that file — se's working_principles_test.go and your portability_test.go both share it as separate new files, per the peer-coordination). go test ./... green except the pre-existing env-gated TestCodexResolveManifestAgainstInstalledHost; gofmt/vet clean.
  `git status` shows only `portability_test.go` added — `skill_surface_test.go` untouched. `go test ./...` green except `TestCodexResolveManifestAgainstInstalledHost` (predates this branch, commit `e3868282`; sandbox denies `~/.codex/config.toml` read so it Fatals instead of Skips — the host-dependency failure the checklist excepts). `gofmt -l .` empty; `go vet ./...` clean.

### Summary

Added `skills/integration/portability_test.go` (commit `41b6eb63`): the shipped-instruction-surface portability oracle complementing zs #246's two host-neutrality oracles one altitude up. Two tests — one asserts the surface names none of the three non-portable markers (with a non-empty-walk guard against vacuous pass), one proves the discriminator positively by confirming the legitimately-host-specific forms (adapter `~/.claude/teams`, project-relative `.claude/agents|worktrees`) are present in the walked surface yet GREEN. Shared the `shippedSkillText`/`skillsRoot`/`repoRoot` helpers read-only (zero edits to `skill_surface_test.go`, so no collision with se's concurrent part-B prose edits). Ran the full falsification spike required by AC-2: every planted marker went RED naming the file (including `python3` in an adapter, since the interpreter check ignores the adapter exclusion), and both portable controls stayed GREEN. `go test ./...`, `gofmt -l`, and `go vet` are clean save the pre-existing env-gated codex test the checklist excepts.
