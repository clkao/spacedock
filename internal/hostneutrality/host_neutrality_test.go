// ABOUTME: AC-3 code-side host-neutrality oracle — a go/parser scan asserting the
// ABOUTME: generic internal/dispatch + internal/status source carries no ~/.claude read.
package hostneutrality

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// guardedPackages are the generic packages the host-neutrality invariant polices.
// They must carry no home-rooted Claude team/transcript read literal — those live
// only in internal/claudeteam behind the injected probe. Paths are relative to
// this test's package dir (go test runs with cwd at the package source dir).
var guardedPackages = []string{"../dispatch", "../status"}

// leak describes one host-coupling read found in guarded source.
type leak struct {
	file string
	line int
	kind string
	text string
}

// TestNoClaudeHomeReadsInGenericPackages parses every non-test .go file in the
// guarded packages and fails if any carries a ~/.claude path-join string literal
// or an os.UserHomeDir-rooted read. These are the home-rooted team/transcript
// reads AC-3 forbids in the generic core; after relocation behind the
// claudeteam.TeamStateProbe seam, none remain. Re-introducing any flips this RED.
func TestNoClaudeHomeReadsInGenericPackages(t *testing.T) {
	var leaks []leak
	for _, pkgDir := range guardedPackages {
		entries, err := os.ReadDir(pkgDir)
		if err != nil {
			t.Fatalf("read package dir %s: %v", pkgDir, err)
		}
		for _, ent := range entries {
			name := ent.Name()
			if ent.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
				continue
			}
			path := filepath.Join(pkgDir, name)
			leaks = append(leaks, scanFile(t, path)...)
		}
	}
	if len(leaks) > 0 {
		for _, l := range leaks {
			t.Errorf("%s:%d: host-coupling %s in generic package: %s", l.file, l.line, l.kind, l.text)
		}
		t.Fatalf("found %d ~/.claude/UserHomeDir read(s) in generic source; relocate them behind claudeteam.TeamStateProbe", len(leaks))
	}
}

// scanFile parses one source file and returns its host-coupling leaks: STRING
// literals containing ".claude" and calls to os.UserHomeDir. Comments are not AST
// nodes here, so prose that mentions ~/.claude (e.g. an adapter-naming comment) is
// correctly ignored — the invariant is over executable source, not text.
func scanFile(t *testing.T, path string) []leak {
	t.Helper()
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	var found []leak
	ast.Inspect(f, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.BasicLit:
			if node.Kind == token.STRING && strings.Contains(node.Value, ".claude") {
				pos := fset.Position(node.Pos())
				found = append(found, leak{file: path, line: pos.Line, kind: "string literal", text: node.Value})
			}
		case *ast.SelectorExpr:
			if pkg, ok := node.X.(*ast.Ident); ok && pkg.Name == "os" && node.Sel.Name == "UserHomeDir" {
				pos := fset.Position(node.Pos())
				found = append(found, leak{file: path, line: pos.Line, kind: "os.UserHomeDir call", text: "os.UserHomeDir"})
			}
		}
		return true
	})
	return found
}
