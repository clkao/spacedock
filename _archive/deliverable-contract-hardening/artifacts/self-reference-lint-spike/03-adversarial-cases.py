import re
SELF_PHRASE = re.compile(
    r"this entity'?s|review of (this|the) (entity|decision|design)[^.]*section|"
    r"the entity'?s own (prose|decision|section)", re.IGNORECASE)
EXTERNAL_TOKENS = re.compile(
    r"\btest\b|\.go\b|exit\s|exit-code|exit code|command|`status|--\w|fixture|"
    r"golden|byte|on-disk|stdout|stderr|assert|parser|mutator|frontmatter|"
    r"code path|command/parser|run(s|ning)? the|invok|drive the|driving the", re.IGNORECASE)
def strip_quotes(s):
    s=re.sub(r'"[^"]*"',' ',s); s=re.sub(r'`[^`]*`',' ',s); return s
def flag(clause):
    c=strip_quotes(clause)
    return bool(SELF_PHRASE.search(c) and not EXTERNAL_TOKENS.search(c))

cases = [
  # (label, text, expected_flag)
  ("TP: real self-oracle (AC-6)", "Verified by: this entity's v1 DECISION section states the decision and cites the roadmap.", True),
  ("TN: AC-1 quotes the antipattern", 'Verified by: a behavioral test driving the real binary — a fixture entity whose only AC is a self-oracle ("verified by review of this entity\'s own decision section") makes status --set exit non-zero.', False),
  ("TN: AC-5 cites code paths", "Verified by: design review of this entity + the spec shows no branching; absence of tracker names in command/parser code paths.", False),
  ("FN-risk: dodged phrasing", "Verified by: inspection of the recorded rationale in the body confirms the decision is justified.", False),  # KNOWN miss — caught by FO cross-check, not lint
  ("TN: ordinary external oracle", "Verified by: a Go test asserts the written file's frontmatter; --validate is VALID immediately after.", False),
  ("TP variant: self prose only", "Verified by: review of this entity's design section shows the approach is sound.", True),
]
print("label | flagged | expected | OK?")
for label,text,exp in cases:
    got=flag(text)
    print(f"{'OK ' if got==exp else 'XX '} {label:42} -> flagged={got} expected={exp}")
