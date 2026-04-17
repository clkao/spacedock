SHELL := /bin/bash

.PHONY: test-static test-e2e test-live-claude test-live-claude-opus test-live-codex test-live-claude-bare test-live-codex-bare

TEST ?= tests/
RUNTIME ?= claude
LIVE_CLAUDE_WORKERS ?= 4
LIVE_CODEX_WORKERS ?= 4
# Pinned to claude-opus-4-6 due to opus-4-7 ensign hallucination regression at low/medium effort; see #177 / #181. Reversible — override with `make test-live-claude-opus OPUS_MODEL=opus` to re-test on opus-4-7.
OPUS_MODEL ?= claude-opus-4-6

test-static:
	unset CLAUDECODE && uv run pytest tests/ --ignore=tests/fixtures \
	  -m "not live_claude and not live_codex" -q

# Single-file live override — pass TEST=tests/<file>.py RUNTIME=claude|codex.
# Replaces the old test-e2e-commission target: `make test-e2e TEST=tests/test_commission.py`.
test-e2e:
	unset CLAUDECODE && uv run pytest $(TEST) --runtime $(RUNTIME) -v

test-live-claude:
	unset CLAUDECODE && { \
	  uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_claude and serial" --runtime claude -x -v ; SEQ=$$? ; \
	  uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_claude and not serial" --runtime claude \
	    -n $(LIVE_CLAUDE_WORKERS) -v ; PAR=$$? ; \
	  test $$SEQ -eq 0 -a $$PAR -eq 0 ; \
	}
	# SKIPPED: test_scaffolding_guardrail.py carries @pytest.mark.skip (see the file). Track: file new task.

# test-live-claude-opus runs the same suite with --model opus --effort low overrides —
# use it to manually verify stronger-model compliance when haiku flakes.
test-live-claude-opus:
	unset CLAUDECODE && { \
	  uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_claude and serial" --runtime claude --model $(OPUS_MODEL) --effort low -x -v ; SEQ=$$? ; \
	  uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_claude and not serial" --runtime claude --model $(OPUS_MODEL) --effort low \
	    -n $(LIVE_CLAUDE_WORKERS) -v ; PAR=$$? ; \
	  test $$SEQ -eq 0 -a $$PAR -eq 0 ; \
	}

test-live-codex:
	{ \
	  uv run python scripts/run_pytest_tier.py --allow-no-tests -- \
	    uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_codex and serial" --runtime codex -x -v ; SEQ=$$? ; \
	  uv run python scripts/run_pytest_tier.py --allow-no-tests -- \
	    uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_codex and not serial" --runtime codex \
	    -n $(LIVE_CODEX_WORKERS) -v ; PAR=$$? ; \
	  test $$SEQ -eq 0 -a $$PAR -eq 0 ; \
	}

# Bare-mode variant of test-live-claude. Runs with CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS
# unset and --team-mode=bare, so tests pinned to teams_mode are auto-skipped and
# tests pinned to bare_mode run; mode-agnostic tests run under the bare dispatch path.
test-live-claude-bare:
	unset CLAUDECODE CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS && { \
	  uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_claude and serial" --runtime claude --team-mode=bare -x -v ; SEQ=$$? ; \
	  uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_claude and not serial" --runtime claude --team-mode=bare \
	    -n $(LIVE_CLAUDE_WORKERS) -v ; PAR=$$? ; \
	  test $$SEQ -eq 0 -a $$PAR -eq 0 ; \
	}

test-live-codex-bare:
	unset CLAUDE_CODE_EXPERIMENTAL_AGENT_TEAMS && { \
	  uv run python scripts/run_pytest_tier.py --allow-no-tests -- \
	    uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_codex and serial" --runtime codex --team-mode=bare -x -v ; SEQ=$$? ; \
	  uv run python scripts/run_pytest_tier.py --allow-no-tests -- \
	    uv run pytest tests/ --ignore=tests/fixtures \
	    -m "live_codex and not serial" --runtime codex --team-mode=bare \
	    -n $(LIVE_CODEX_WORKERS) -v ; PAR=$$? ; \
	  test $$SEQ -eq 0 -a $$PAR -eq 0 ; \
	}
