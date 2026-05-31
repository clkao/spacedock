// ABOUTME: The contract-version axis — CONTRACT_VERSION, the half-open
// ABOUTME: requires-contract range grammar, and the five-verdict compatibility compare.
package contract

import (
	"fmt"
	"strconv"
	"strings"
)

// CONTRACT_VERSION is the monotonic integer naming the binary's observable
// contract the vendored skill surface depends on (the set of `spacedock`
// subcommands, flags, and output sections). It is distinct from the plugin's
// display semver and from the build version. Bump it only when a change to the
// binary alters the observable surface the FO/ensign contracts call — never as a
// side effect of a routine release bump (see the entity's OPEN-2 bump discipline).
const CONTRACT_VERSION = 1

// Verdict is the compatibility class produced by comparing a binary's contract
// version against a plugin's declared requires-contract range.
type Verdict int

const (
	// Compatible means lo <= C < hi: the binary's contract sits inside the
	// plugin's declared half-open range.
	Compatible Verdict = iota
	// TooOldBinary means C < lo: the installed binary predates the contract the
	// plugin needs. Remedy: rebuild/upgrade the binary.
	TooOldBinary
	// TooOldPlugin means C >= hi: the installed plugin predates the binary's
	// contract. Remedy: update/reinstall the plugin.
	TooOldPlugin
	// MalformedRange means the manifest's requires-contract does not parse as
	// ">=N,<M". A packaging bug, not a too-old install; no upgrade remedy.
	MalformedRange
	// NoPluginFound means no installed plugin manifest could be resolved for the
	// host. Distinct, non-fatal-by-default; reported rather than asserting
	// compatibility.
	NoPluginFound
)

// String renders the verdict's stable kebab-case token (the oracle string AC-1
// asserts on for compatible/no-plugin-found and embeds in the remedy lines).
func (v Verdict) String() string {
	switch v {
	case Compatible:
		return "compatible"
	case TooOldBinary:
		return "too-old-binary"
	case TooOldPlugin:
		return "too-old-plugin"
	case MalformedRange:
		return "malformed-range"
	case NoPluginFound:
		return "no-plugin-found"
	default:
		return "unknown"
	}
}

// Result carries a comparison's verdict and the operator-facing message. For
// Compatible the message is a one-line "OK" report; for every mismatch it is the
// shared-shape actionable message with the per-class remedy.
type Result struct {
	Verdict Verdict
	Message string
}

// ParseRange parses a requires-contract value of the form ">=N,<M" into its
// half-open integer bounds [lo, hi). Surrounding whitespace is tolerated. Any
// other shape — missing a bound, the wrong operator, non-integer bounds, an
// empty or inverted interval, or extra clauses — is a parse error.
func ParseRange(raw string) (lo int, hi int, err error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected exactly two clauses %q: %q", ">=N,<M", raw)
	}
	loStr := strings.TrimSpace(parts[0])
	hiStr := strings.TrimSpace(parts[1])
	if !strings.HasPrefix(loStr, ">=") {
		return 0, 0, fmt.Errorf("lower bound must use >=: %q", loStr)
	}
	if !strings.HasPrefix(hiStr, "<") || strings.HasPrefix(hiStr, "<=") {
		return 0, 0, fmt.Errorf("upper bound must use <: %q", hiStr)
	}
	lo, err = strconv.Atoi(strings.TrimSpace(loStr[2:]))
	if err != nil {
		return 0, 0, fmt.Errorf("lower bound is not an integer: %q", loStr)
	}
	hi, err = strconv.Atoi(strings.TrimSpace(hiStr[1:]))
	if err != nil {
		return 0, 0, fmt.Errorf("upper bound is not an integer: %q", hiStr)
	}
	if lo >= hi {
		return 0, 0, fmt.Errorf("empty interval: lower bound %d not below upper bound %d", lo, hi)
	}
	return lo, hi, nil
}

// Compare classifies a binary at contract version c against a plugin's raw
// requires-contract range, for the named host and (pre-release) dev branch. It
// returns the verdict and the operator-facing message. NoPluginFound is produced
// by the caller (when the manifest is absent), not here — Compare always has a
// raw range string to evaluate.
func Compare(c int, raw, host, branch string) Result {
	manifestNote := ""
	return compareWithManifest(c, raw, host, branch, manifestNote)
}

// compareWithManifest is Compare with an optional manifest path woven into the
// malformed-range message so a packaging bug names the offending file.
func compareWithManifest(c int, raw, host, branch, manifestPath string) Result {
	lo, hi, err := ParseRange(raw)
	if err != nil {
		loc := manifestPath
		if loc == "" {
			loc = "the plugin manifest"
		}
		return Result{
			Verdict: MalformedRange,
			Message: fmt.Sprintf(
				"malformed contract range %q in %s: expected \">=N,<M\". "+
					"This is a packaging bug — the plugin manifest is wrong, not your install.",
				strings.TrimSpace(raw), loc),
		}
	}
	rangeStr := fmt.Sprintf(">=%d,<%d", lo, hi)
	switch {
	case c < lo:
		return Result{Verdict: TooOldBinary, Message: mismatchMessage(c, rangeStr, tooOldBinaryRemedy(c, rangeStr, branch))}
	case c >= hi:
		return Result{Verdict: TooOldPlugin, Message: mismatchMessage(c, rangeStr, tooOldPluginRemedy(c, rangeStr, host))}
	default:
		return Result{Verdict: Compatible, Message: fmt.Sprintf("OK: binary contract %d satisfies plugin range %s.", c, rangeStr)}
	}
}

// mismatchMessage assembles the shared-shape mismatch message: a header naming
// the binary's contract and the required range, the per-class remedy line, and
// the doctor pointer.
func mismatchMessage(c int, rangeStr, remedy string) string {
	return fmt.Sprintf(
		"Spacedock contract mismatch: binary is contract %d, plugin requires %s.\n"+
			"  %s\n"+
			"Run `spacedock doctor` for details.",
		c, rangeStr, remedy)
}

// tooOldBinaryRemedy is the pinned too-old-binary remedy line. The optional
// pre-release branch suffixes the go-install path so a dev install pins the
// branch; the default release path omits it.
func tooOldBinaryRemedy(c int, rangeStr, branch string) string {
	suffix := "@latest"
	if branch != "" {
		suffix = "@" + branch
	}
	return fmt.Sprintf(
		"too-old-binary: your spacedock binary (contract %d) predates this plugin (needs %s). "+
			"Rebuild/upgrade spacedock: go install github.com/clkao/spacedock-v1/cmd/spacedock%s "+
			"(or pull and 'go build').",
		c, rangeStr, suffix)
}

// tooOldPluginRemedy is the pinned too-old-plugin remedy line, parameterized by
// the detected host for the init/update hint.
func tooOldPluginRemedy(c int, rangeStr, host string) string {
	return fmt.Sprintf(
		"too-old-plugin: your installed plugin (needs %s) predates this binary (contract %d). "+
			"Update it: spacedock init --host %s (or '%s plugin update spacedock').",
		rangeStr, c, host, host)
}

// noPluginMessage is the pinned no-plugin-found report for a host. Not a
// mismatch — there is no range to compare — so it stands alone, exit non-fatal
// by the caller's policy.
func noPluginMessage(host string) string {
	return fmt.Sprintf(
		"no installed Spacedock plugin found for host %s. Install it: spacedock init --host %s.",
		host, host)
}
