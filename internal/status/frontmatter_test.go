// ABOUTME: AC-2 frontmatter parser table tests — the line-oriented parser
// ABOUTME: matches the oracle's _has_opening_fence + parse_frontmatter edge cases.
package status

import (
	"reflect"
	"testing"
)

func TestContentHasOpeningFence(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want bool
	}{
		{"plain fence", "---\nid: 1\n---\n", true},
		{"leading blank lines skipped", "\n\n---\nid: 1\n---\n", true},
		{"whitespace-first-line disqualifies", "   \n---\n", false},
		{"no fence", "# heading\n", false},
		{"bom then fence", utf8BOM + "---\nid: 1\n---\n", true},
		{"empty file", "", false},
		{"text before fence", "hello\n---\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := contentHasOpeningFence([]byte(tc.in)); got != tc.want {
				t.Fatalf("contentHasOpeningFence(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseFrontmatterContent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]string
	}{
		{
			name: "basic fields",
			in:   "---\nid: \"001\"\ntitle: Hello\nscore: 0.8\n---\nbody\n",
			want: map[string]string{"id": "001", "title": "Hello", "score": "0.8"},
		},
		{
			name: "empty value yields empty string",
			in:   "---\nscore:\n---\n",
			want: map[string]string{"score": ""},
		},
		{
			name: "matched single quotes stripped",
			in:   "---\ntitle: 'Quoted'\n---\n",
			want: map[string]string{"title": "Quoted"},
		},
		{
			name: "mismatched quotes preserved",
			in:   "---\ntitle: \"half'\n---\n",
			want: map[string]string{"title": "\"half'"},
		},
		{
			name: "nested indented lines ignored",
			in:   "---\nstages:\n  defaults:\n    worktree: false\nid: 1\n---\n",
			want: map[string]string{"stages": "", "id": "1"},
		},
		{
			name: "last top-level key wins",
			in:   "---\nstatus: a\nstatus: b\n---\n",
			want: map[string]string{"status": "b"},
		},
		{
			name: "no opening fence yields empty",
			in:   "# heading\nid: 1\n",
			want: map[string]string{},
		},
		{
			name: "value with colon splits on first colon only",
			in:   "---\nurl: http://x:8080\n---\n",
			want: map[string]string{"url": "http://x:8080"},
		},
		{
			name: "leading bom on first key line",
			in:   utf8BOM + "---\nid: 1\n---\n",
			want: map[string]string{"id": "1"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseFrontmatterContent([]byte(tc.in))
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseFrontmatterContent(%q) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}
