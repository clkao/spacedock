import re, glob, sys

# Candidate self-reference regex from the proposal (principle 1, section c):
#   names *this entity's own* sections as its proof
# We translate that into a set of patterns over the "Verified by" / "Proof" text.
SELF_PATTERNS = [
    r"this entity'?s",
    r"\bthis entity\b",
    r"decision section",
    r"static (prose )?review of (this|the)[^.]*(decision|section)",
    r"review of (this|the) (entity|decision|design) section",
]
self_re = re.compile("|".join(SELF_PATTERNS), re.IGNORECASE)

# Extract per-AC verification text. An AC starts at a line matching **AC-...** and the
# verification text is everything from that line until the next AC line or a "## " heading
# or a blank line that is followed by a non-continuation. We keep it simple: accumulate
# lines after the AC header until the next blank line OR next AC/heading.
def extract_acs(path):
    acs = []
    with open(path) as fh:
        lines = fh.read().splitlines()
    i = 0
    while i < len(lines):
        if re.match(r"^\*\*AC-", lines[i]):
            header = lines[i].strip()
            buf = []
            j = i + 1
            while j < len(lines):
                l = lines[j]
                if re.match(r"^\*\*AC-", l) or re.match(r"^## ", l):
                    break
                buf.append(l)
                j += 1
            acs.append((header, " ".join(buf)))
            i = j
        else:
            i += 1
    return acs

total = 0
hits = []
for path in sorted(glob.glob("**/index.md", recursive=True)):
    for header, body in extract_acs(path):
        total += 1
        m = self_re.search(body)
        if m:
            hits.append((path, header, m.group(0), body[:200]))

print(f"Total AC items scanned: {total}")
print(f"Self-reference HITS: {len(hits)}\n")
for path, header, matched, snippet in hits:
    short = path.replace("./_archive/","arc:").replace("/index.md","")
    print(f"--- {short}")
    print(f"    {header[:90]}")
    print(f"    MATCHED: {matched!r}")
    print(f"    CTX: {snippet}")
    print()
