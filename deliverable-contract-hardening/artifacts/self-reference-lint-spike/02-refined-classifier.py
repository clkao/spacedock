import re, glob

# REFINED approach: classify the AC by what its PRIMARY oracle is, not by whether the
# self-reference phrase appears ANYWHERE. Strategy:
#  1. Isolate the "Verified by:" / "Oracle:" clause (the proof statement).
#  2. Strip quoted spans ("...") — quoted text is an EXAMPLE, not the live oracle.
#  3. A self-oracle = the proof clause, after quote-stripping, names this-entity's own
#     prose AND cites no external surface (no test/command/file/code/exit-code token).
EXTERNAL_TOKENS = re.compile(
    r"\btest\b|\.go\b|exit\s|exit-code|exit code|command|`status|--\w|fixture|"
    r"golden|byte|on-disk|stdout|stderr|assert|parser|mutator|frontmatter|"
    r"code path|command/parser|run(s|ning)? the|invok|drive the|driving the",
    re.IGNORECASE)
SELF_PHRASE = re.compile(
    r"this entity'?s|review of (this|the) (entity|decision|design)[^.]*section|"
    r"the entity'?s own (prose|decision|section)",
    re.IGNORECASE)

def strip_quotes(s):
    # remove "..." and `...` quoted spans (examples, not live oracle)
    s = re.sub(r'"[^"]*"', ' ', s)
    s = re.sub(r'`[^`]*`', ' ', s)
    return s

def proof_clause(body):
    # take from the first Verified by / Oracle / Proof marker onward
    m = re.search(r'(verified by|oracle:|proof:|end state[:.])', body, re.IGNORECASE)
    return body[m.start():] if m else body

def extract_acs(path):
    acs=[]
    lines=open(path).read().splitlines()
    i=0
    while i<len(lines):
        if re.match(r"^\*\*AC-",lines[i]):
            header=lines[i].strip(); buf=[]; j=i+1
            while j<len(lines) and not re.match(r"^\*\*AC-",lines[j]) and not re.match(r"^## ",lines[j]):
                buf.append(lines[j]); j+=1
            acs.append((header," ".join(buf))); i=j
        else: i+=1
    return acs

total=0; flagged=[]
for path in sorted(glob.glob("**/index.md",recursive=True)):
    for header,body in extract_acs(path):
        total+=1
        clause = proof_clause(body)
        cleaned = strip_quotes(clause)
        has_self = SELF_PHRASE.search(cleaned)
        has_ext  = EXTERNAL_TOKENS.search(cleaned)
        if has_self and not has_ext:
            flagged.append((path,header,has_self.group(0),cleaned[:220]))

print(f"Total AC scanned: {total}")
print(f"REFINED self-oracle flags (self-ref present, NO external token): {len(flagged)}\n")
for path,header,matched,snip in flagged:
    short=path.replace('./_archive/','arc:').replace('/index.md','')
    print(f"--- {short}\n    {header[:88]}\n    SELF: {matched!r}\n    CLEANED: {snip}\n")
