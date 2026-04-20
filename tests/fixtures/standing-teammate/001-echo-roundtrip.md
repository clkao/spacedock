---
id: "001"
title: Echo-agent roundtrip live check
status: backlog
score: 0.50
source: test
started:
completed:
verdict:
worktree:
---

Append the line "work done" to this entity file body and commit. That is
the entire deliverable. After you commit, SendMessage to `echo-agent` with
exactly the text `ping` and capture the reply (which must start with
`ECHO: `). Append a line containing the captured "ECHO: ..." reply to THIS
ENTITY FILE'S body on disk (via Edit tool), then commit. The captured
reply must survive archival — the test reads the archived body to verify
the roundtrip completed.
