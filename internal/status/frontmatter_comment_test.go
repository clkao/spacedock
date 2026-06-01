// ABOUTME: AC-4(ii) generic entity-value inline-comment strip + quoted-value
// ABOUTME: protection — reader strips unquoted comments, keeps quoted # literal.
package status

import (
	"reflect"
	"testing"
)

func TestParseFrontmatterStripsInlineComment(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want map[string]string
	}{
		{
			name: "unquoted value with trailing comment stripped",
			in:   "---\nstatus: plan  # note\n---\n",
			want: map[string]string{"status": "plan"},
		},
		{
			name: "unquoted single-token value with trailing comment",
			in:   "---\nstatus: plan # note\n---\n",
			want: map[string]string{"status": "plan"},
		},
		{
			name: "unspaced hash kept (no preceding whitespace)",
			in:   "---\nref: v1.0#163\n---\n",
			want: map[string]string{"ref": "v1.0#163"},
		},
		{
			name: "unquoted space-preceded hash truncates (accepted under option C)",
			in:   "---\nsource: consolidates #223, #217\n---\n",
			want: map[string]string{"source": "consolidates"},
		},
		{
			name: "quoted value protects an interior hash",
			in:   "---\nsource: \"consolidates #223, #217\"\n---\n",
			want: map[string]string{"source": "consolidates #223, #217"},
		},
		{
			name: "quoted value with trailing comment after close quote",
			in:   "---\ntitle: \"hello\"  # a note\n---\n",
			want: map[string]string{"title": "hello"},
		},
		{
			name: "single-quoted value protects an interior hash",
			in:   "---\nsource: 'see #163'\n---\n",
			want: map[string]string{"source": "see #163"},
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
