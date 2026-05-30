---
id: hxs93wd0bjwhc3vsjwx1seew
title: Harden the deliverable contract — behavioral-oracle gate guard, four principles, portable TDD, ergonomics
status: ideation
source: sprint — FO/captain (deliverable-principles + TDD/template study)
score: "0.34"
worktree:
started: 2026-05-30T22:25:57Z
---

Encode the hard-won deliverable principles into the SHIPPED spacedock contract so future dev work cannot drift into the doc-only / grep-proof antipatterns, and so the discipline travels with the published plugin rather than depending on any one user's environment. Design inputs (both in `docs/dev/_proposals/`): `encoding-deliverable-principles.md` (the four principles + the code guard, with exact before/after wording) and `tdd-and-template-adoption.md` (template comparison + ergonomics) — **with that study's "rely on global CLAUDE.md" TDD conclusion SUPERSEDED** (see AC-3 / Notes).

The four principles: (P1) no doc-only deliverable — every AC has an oracle external to itself; (P2) proof is behavioral, not grep; (P3) enforce in code, not prose; (P4) spike the riskiest unknown in ideation. Plus a generalizing meta-principle surfaced this sprint: **PORTABILITY / self-sufficiency** — the shipped contract assumes nothing from the user's environment (no per-user CLAUDE.md, no Python-on-PATH in the dispatch path, no plugin-private paths). The TDD correction is an instance: TDD authoring discipline must live in the portable `spacedock:ensign` contract, not global CLAUDE.md.

Plus FO-POSTURE principles (captain, "What Awesome Looks Like") — distinct from the deliverable principles (those govern entity proof; these govern how the FO operates), and equally portable (shipped FO contract, not CLAUDE.md): (1) **begin with the end** — name the observable end value before the work, the orchestration analog of P1; (2) **bring a yes-able ask** — lead with the decision + recommendation, supply rationale + evidence to vote yes, cut the rest (generalizes the existing Gate Presentation discipline to every approval/escalation); (3) **JFDI, no doc ceremony** — take the obvious in-authority action without asking; produce no document that isn't a deliverable-with-an-oracle or analysis feeding a decision. #2 and #3 compose, not conflict: JFDI the reversible/in-authority work, reserve the captain for genuine forks *with* a yes-able proposal (= the handoff's auto-approve-vs-escalate rule).

**Doc-only ban (P1, operationalized — concise form for the contract).** Every entity ships a behavior change — code, test, config, a captured measurement, or a decision the captain records; a design alone is the IDEATION output of an implement-bound entity, never its own terminal stage. Captain veto before code lands is the IDEATION GATE itself — for a veto-needing entity the FO presents that gate for explicit approval rather than auto-approving — so there is no doc-only ceremony (file → worktree-to-write-markdown → gate-on-prose → "now implement it too" → pay twice). This workflow needs NO new `auto-approve` frontmatter field, `plan` stage, or `research-spike` entity-type (those are another workflow's mechanisms): here the ideation gate is the veto, P4 is the in-ideation spike, and a genuinely code-free decision lives as an ADR/roadmap entry, not a dev-queue entity. (external-tracker-checkpoint is the live cautionary tale: it should have folded into its siblings' ACs + an ADR.)

**Wording (shipped contract): plain language, no jargon.** Do NOT use the term "oracle" in the README, FO contract, or ensign contract — say plainly: proof must be EXTERNAL to the entity and able to FAIL — a test, a command's exit code/output, or resulting on-disk state — not a re-reading of the entity's own prose. The FO contract carries only the GATE CHECK (at each gate, confirm every AC's proof meets that bar; reject ACs proven only by re-reading the entity), in the register of the existing AC-coverage cross-check — not a test-theory lecture; the fuller "what makes a good test" guidance lives README-side (author-facing) and ensign-side (worker-facing). Name the guard plainly (e.g. self-reference check), not "self-oracle lint." (This seed uses shorthand for brevity; the SHIPPED contract text must be plain.)

## Acceptance criteria

Seed criteria for ideation to sharpen. The load-bearing deliverable is the CODE GUARD (AC-1) — its red/green tests are this entity's external behavioral oracle, so the entity is NOT itself doc-only. The prose-encoding ACs are honest about their ceiling (wording-present) and lean on the guard + the behavioral-proof gate for teeth, per P3.

**AC-1 — A `status --validate` self-oracle lint plus a terminal-PASSED `--set` guard refuse advancing an entity whose ACs cite no external oracle.**
Verified by: a behavioral test driving the real binary (modeled on `internal/status/archive_guard_test.go`) — a fixture entity whose only AC is a self-oracle ("verified by review of this entity's own decision section") makes `status --set <slug> verdict=passed status=done` exit non-zero and leaves the entity unmutated absent `--force`; a clean entity with an external-oracle AC reaches `done` cleanly; `--validate` flags the self-oracle AC with the standard `Error: … slug= … path=` line. Modeled on the existing mod-block / merge-hook terminal-transition guard.

**AC-2 — A portability invariant check proves the generic contract core relies on nothing environment-specific.**
Verified by: an invariant test that PARSES the real artifacts (the generic operating-contract core + the host-neutral binary packages) and asserts zero reliance on non-portable surfaces — no per-user CLAUDE.md dependency, no Python-on-PATH in the dispatch path, no plugin-private `skills/commission/bin/*` paths in shipped contract text. (Invariant over real values, not a substring spelling check — the legitimate static-test shape.)

**AC-3 — The four principles + the PORTABILITY principle are encoded in the shipped README/operating contract, and the TDD authoring rule lives in the portable `spacedock:ensign` contract.**
Verified by: the contract edits land in the SHIPPED/vendored surface (README stages + `first-officer-shared-core.md` + `spacedock:ensign`), and a clean-room worker (no CLAUDE.md) is governed by the TDD rule because it ships in the ensign skill. Proof ceiling acknowledged: contract prose is wording-present, not behavior (P3) — the behavioral teeth are AC-1's guard (makes a real external/behavioral test mandatory for every behavioral AC) and the gate's behavioral-proof cross-check. The TDD authoring-ORDER rule is unenforceable at the gate by design; its home is the portable contract, the gate enforces the RESULT.

**AC-4 — Entity-template ergonomics adopted from the newer builtin template (adapted, not copied).**
Verified by: the Task Template gains a `## Out of scope` slot and promotes `## Problem` / `## Proposed approach` / `## Test plan` to headings (keeping this workflow's richer wording + sharper AC lead-in), and Workflow State documents the native `spacedock status --workflow-dir docs/dev --next` query. Lower-stakes docs change; gated on the FO's structural scan, not a behavioral test.

## Test gates

- `go test ./...`
- The AC-1 guard behavioral tests (self-oracle refused at terminal-PASSED; clean entity passes; `--force` bypasses).
- The AC-2 portability invariant test.

## Notes

**Sequencing: this lands LAST among the contract/Go entities.** It touches `internal/status` (the guard, `validate.go`/`handlers.go`/`mutate.go`) and the operating-contract references — the same surfaces `agent-output-modes`, `native-dispatch-helper`, and `claude-runtime-segregation` edit. Sequence after them to avoid same-package + same-file merge collisions.

**Canonical vs vendored:** author the contract edits in this project's VENDORED copy first (the leading edge), then flow upstream to `~/git/spacedock` — the direction the launcher-command/split-root amendments already flowed; authoring in canonical first would force a re-vendor and risk re-introducing plugin-private paths.

**Captain sub-decisions (recommended defaults, confirm or adjust at ideation):** ADR home = the roadmap, no new `docs/decisions/` dir; no `decision` entity-type; spike rule = MUST-with-recorded-escape (`no spike needed: {mechanisms}`); skip the optional `--validate` grep-suspect/ideation-presence warning lints (low ROI). **Do-not-regress** when editing the README: the split-root state model, the Testing Resources table, sd-b32 specifics, native-status commands, and the project-specific Good/Bad + Staff-review triggers — the generic builtin template has none of these.

Per P4: if a design choice here rests on an unverified mechanism (e.g. whether the self-oracle lint can be made low-false-positive, or whether the portability invariant is mechanically checkable without a grep), ideation must SPIKE it and record the behavioral evidence.
