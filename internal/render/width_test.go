package render

import (
	"strings"
	"testing"

	"tuipresent/internal/deck"
)

func TestContentWidthMonotonic(t *testing.T) {
	prev := 0
	for w := 1; w <= 300; w++ {
		cw := contentWidth(w)
		if cw < prev {
			t.Errorf("contentWidth(%d) = %d < contentWidth(%d) = %d", w, cw, w-1, prev)
		}
		if cw < 1 || cw > w {
			t.Errorf("contentWidth(%d) = %d, want 1..%d", w, cw, w)
		}
		if w >= 40 {
			if want := min(w-8, maxContentWidth); cw != want {
				t.Errorf("contentWidth(%d) = %d, want %d (unchanged at normal sizes)", w, cw, want)
			}
		}
		prev = cw
	}
}

func TestFencedCodeUsesParserFenceRules(t *testing.T) {
	long := strings.Repeat("x", 60)
	for _, tc := range []struct {
		name, md, want string
	}{
		{"nested", "````markdown\n```go\n" + long + "\n```\n````\nprose", "```go\n" + long + "\n```"},
		{"info string does not close", "```\na\n```go\nb\n```\nprose", "a\n```go\nb"},
		{"longer closer", "~~~\na\n~~~~~\nprose", "a"},
		{"indented", "  ```\na\n  ```\nprose", "a"},
	} {
		if got := fencedCode(tc.md); got != tc.want {
			t.Errorf("%s: fencedCode = %q, want %q", tc.name, got, tc.want)
		}
	}
	// The long line inside the nested fence is code and must be measured.
	d, err := deck.Parse("````markdown\n```go\n" + long + "\n```\n````\n")
	if err != nil {
		t.Fatal(err)
	}
	if _, cols := New(d).Overflow(0, 60, 24); cols == 0 {
		t.Error("code inside a nested fence was not measured")
	}
}
