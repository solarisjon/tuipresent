//go:build unix

package main

import (
	"os"

	"golang.org/x/sys/unix"
)

// cellPixels is the terminal's cell size in pixels, or 0, 0 if it does not
// report its pixel size.
func cellPixels(f *os.File) (w, h int) {
	ws, err := unix.IoctlGetWinsize(int(f.Fd()), unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 {
		return 0, 0
	}
	return int(ws.Xpixel) / int(ws.Col), int(ws.Ypixel) / int(ws.Row)
}
