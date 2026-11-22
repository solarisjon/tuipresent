package transition

import (
	"strings"
	"testing"
)

func TestLookup(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Kind
		ok   bool
	}{
		{"wipe", Wipe, true},
		{" Dissolve ", Dissolve, true},
		{"NONE", None, true},
		{"wpie", Slide, false},
		{"", Slide, false},
	} {
		k, ok := Lookup(tc.in)
		if k != tc.want || ok != tc.ok {
			t.Errorf("Lookup(%q) = %q, %v; want %q, %v", tc.in, k, ok, tc.want, tc.ok)
		}
		if Parse(tc.in) != tc.want {
			t.Errorf("Parse(%q) = %q, want %q", tc.in, Parse(tc.in), tc.want)
		}
	}
	w := UnknownWarning("wpie")
	for _, s := range []string{`"wpie"`, "using slide", "slide, wipe, cover, dissolve, none"} {
		if !strings.Contains(w, s) {
			t.Errorf("UnknownWarning = %q, missing %q", w, s)
		}
	}
}
