// ABOUTME: Identity engine — sequential / sd-b32 / slug id derivation, display
// ABOUTME: prefixes, and the sd-b32 SHA-256 digest material matching the oracle.
package status

import (
	"crypto/sha256"
	"fmt"
	"io"
	"math/big"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	sdB32Chars     = "0123456789abcdefghjkmnpqrstvwxyz"
	sdB32IDLength  = 24
	sdB32MinPrefix = 2
)

var (
	idStyles    = map[string]bool{"sequential": true, "slug": true, "sd-b32": true}
	sdB32IDRe   = regexp.MustCompile(`^[` + sdB32Chars + `]{24}$`)
	allDigitsRe = regexp.MustCompile(`^[0-9]+$`)
)

// env is the explicit process environment threaded through identity derivation
// and the boot probes, mirroring os.environ.get lookups in the oracle.
type env map[string]string

// envFromSlice builds an env map from KEY=VALUE entries (last wins).
func envFromSlice(pairs []string) env {
	e := env{}
	for _, kv := range pairs {
		if k, v, ok := strings.Cut(kv, "="); ok {
			e[k] = v
		}
	}
	return e
}

func (e env) get(key string) string { return e[key] }

// isValidSDB32ID matches is_valid_sd_b32_id.
func isValidSDB32ID(value string) bool {
	return sdB32IDRe.MatchString(value)
}

// isDigits reports whether s is a non-empty run of ASCII digits (Python's
// str.isdigit for the simple cases the oracle relies on).
func isDigits(s string) bool {
	return allDigitsRe.MatchString(s)
}

// workflowIDStyle reads the README id-style. Returns ("", err) on an
// unsupported style, matching workflow_id_style's ValueError. definitionDir is
// the README root. Missing README defaults to "sequential".
func workflowIDStyle(definitionDir string) (string, error) {
	readme := filepath.Join(definitionDir, "README.md")
	if !isRegularFile(readme) {
		return "sequential", nil
	}
	style := strings.TrimSpace(parseFrontmatter(readme)["id-style"])
	if style == "" {
		style = "sequential"
	}
	if !idStyles[style] {
		return "", fmt.Errorf("unsupported id-style: %s", style)
	}
	return style, nil
}

// computeSDB32DisplayIDs returns stored_id -> shortest unique address prefix for
// valid sd-b32 ids. Matches compute_sd_b32_display_ids.
func computeSDB32DisplayIDs(entities []*entity) map[string]string {
	var validIDs []string
	for _, e := range entities {
		id := e.storedID
		if isValidSDB32ID(id) {
			validIDs = append(validIDs, id)
		}
	}
	display := map[string]string{}
	for _, value := range validIDs {
		length := sdB32MinPrefix
		for length < len(value) {
			prefix := value[:length]
			count := 0
			for _, other := range validIDs {
				if strings.HasPrefix(other, prefix) {
					count++
				}
			}
			if count == 1 {
				break
			}
			length++
		}
		display[value] = value[:length]
	}
	return display
}

// applyEffectiveIDs sets each entity's displayID and overwrites its display-
// facing fields["id"], preserving storedID. all is the population used to
// compute sd-b32 prefixes. Matches apply_effective_ids.
func applyEffectiveIDs(entities []*entity, idStyle string, all []*entity) {
	if all == nil {
		all = entities
	}
	var sdDisplay map[string]string
	if idStyle == "sd-b32" {
		sdDisplay = computeSDB32DisplayIDs(all)
	}
	for _, e := range entities {
		var displayID string
		switch idStyle {
		case "slug":
			displayID = e.slug
		case "sd-b32":
			if d, ok := sdDisplay[e.storedID]; ok {
				displayID = d
			} else {
				displayID = e.storedID
			}
		default:
			displayID = e.storedID
		}
		e.displayID = displayID
		e.fields["id"] = displayID
	}
}

// sdB32Timestamp returns the sd-b32 timestamp, honoring the test hook env var,
// else now() in ISO-microsecond Z form. Matches sd_b32_timestamp.
func sdB32Timestamp(e env) string {
	if ts := strings.TrimSpace(e.get("SPACEDOCK_TEST_SD_B32_TIMESTAMP")); ts != "" {
		return ts
	}
	return time.Now().UTC().Format("2006-01-02T15:04:05.000000") + "Z"
}

// encodeSDB32Digest extracts a 24-char id from the digest via big-endian 5-bit
// windows. Matches encode_sd_b32_digest.
func encodeSDB32Digest(digest []byte) string {
	value := new(big.Int).SetBytes(digest)
	totalBits := len(digest) * 8
	var sb strings.Builder
	mask := big.NewInt(31)
	for offset := totalBits - 5; offset >= 0; offset -= 5 {
		shifted := new(big.Int).Rsh(value, uint(offset))
		idx := new(big.Int).And(shifted, mask).Int64()
		sb.WriteByte(sdB32Chars[idx])
		if sb.Len() == sdB32IDLength {
			break
		}
	}
	return sb.String()
}

// sdB32DigestMaterial builds the exact digest input lines. Matches
// sd_b32_digest_material. definitionDir is realpath'd for the workflow= line.
func sdB32DigestMaterial(definitionDir, idSeed, idActor, timestamp string, nonce int, e env) []byte {
	seed := strings.TrimSpace(idSeed)
	if seed == "" {
		seed = "manual"
	}
	actor := strings.TrimSpace(idActor)
	if actor == "" {
		actor = strings.TrimSpace(e.get("SPACEDOCK_ID_ACTOR"))
	}
	if actor == "" {
		actor = strings.TrimSpace(e.get("USER"))
	}
	if actor == "" {
		actor = strings.TrimSpace(e.get("USERNAME"))
	}
	context := strings.TrimSpace(e.get("SPACEDOCK_ID_CONTEXT"))
	lines := []string{
		"spacedock-sd-b32-v1",
		"workflow=" + realpathOf(definitionDir),
		"context=" + context,
		"seed=" + seed,
		"actor=" + actor,
		"timestamp=" + timestamp,
		"nonce=" + strconv.Itoa(nonce),
	}
	return []byte(strings.Join(lines, "\n"))
}

// sdB32Candidate computes one candidate id. Matches sd_b32_candidate.
func sdB32Candidate(definitionDir, idSeed, idActor, timestamp string, nonce int, e env) string {
	sum := sha256.Sum256(sdB32DigestMaterial(definitionDir, idSeed, idActor, timestamp, nonce, e))
	return encodeSDB32Digest(sum[:])
}

// computeNextSequentialID returns the zero-padded next id across active +
// archived entities. Matches compute_next_sequential_id (uses the display id
// field, which equals stored for sequential).
func computeNextSequentialID(entityDir string, stderr io.Writer) string {
	maxID := 0
	all := append(scanEntities(entityDir, stderr), archiveEntities(entityDir, stderr)...)
	for _, e := range all {
		if e.fields["id"] != "" {
			if n, err := strconv.Atoi(e.fields["id"]); err == nil && n > maxID {
				maxID = n
			}
		}
	}
	return fmt.Sprintf("%03d", maxID+1)
}

// computeNextSDB32ID mints the next unique sd-b32 id, looping nonces 0..1023.
// Matches compute_next_sd_b32_id. Returns ("", err) on invalid candidate or
// collision-retry exhaustion (the oracle exits 1 with these messages).
func computeNextSDB32ID(definitionDir, entityDir, idSeed, idActor string, e env, stderr io.Writer) (string, error) {
	existing := map[string]bool{}
	for _, ent := range activeAndArchivedEntities(entityDir, stderr) {
		if ent.storedID != "" {
			existing[ent.storedID] = true
		}
	}
	timestamp := sdB32Timestamp(e)
	for nonce := 0; nonce < 1024; nonce++ {
		candidate := sdB32Candidate(definitionDir, idSeed, idActor, timestamp, nonce, e)
		if !isValidSDB32ID(candidate) {
			return "", fmt.Errorf("sd-b32 stored id candidate is invalid: %s", candidate)
		}
		if !existing[candidate] {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("sd-b32 stored id collision retry exhausted")
}

// computeNextID dispatches by id-style. Returns ("", err) with the oracle's
// error text for the slug / seed-on-sequential cases. Matches compute_next_id.
func computeNextID(definitionDir, entityDir, idStyle, idSeed, idActor string, e env, stderr io.Writer) (string, error) {
	switch idStyle {
	case "sequential":
		if idSeed != "" || idActor != "" {
			return "", fmt.Errorf("--id-seed and --id-actor are only applicable for id-style: sd-b32")
		}
		return computeNextSequentialID(entityDir, stderr), nil
	case "slug":
		return "", fmt.Errorf("status --next-id is not applicable for id-style: slug")
	case "sd-b32":
		return computeNextSDB32ID(definitionDir, entityDir, idSeed, idActor, e, stderr)
	}
	return "", fmt.Errorf("unsupported id-style: %s", idStyle)
}
