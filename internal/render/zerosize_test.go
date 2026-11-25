package render

import "testing"

func TestZeroSizeScreensDoNotPanic(t *testing.T) {
	r := renderDeck(t, "T\n===\n<!-- title_size: 2 -->\n<!-- pause -->\nx\n")
	for _, sz := range [][2]int{{0, 0}, {0, 5}, {5, 0}} {
		for name, fn := range map[string]func(){
			"frame":    func() { r.Frame(0, 0, sz[0], sz[1]) },
			"overview": func() { r.Overview(0, sz[0], sz[1]) },
			"bigtitle": func() { r.BigTitleAt(0, 0, sz[0], sz[1]) },
			"overflow": func() { r.Overflow(0, sz[0], sz[1]) },
		} {
			func() {
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("%s %v: panic %v", name, sz, p)
					}
				}()
				fn()
			}()
		}
	}
}
