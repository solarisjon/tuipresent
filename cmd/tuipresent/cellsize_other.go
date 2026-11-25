//go:build !unix

package main

import "os"

func cellPixels(*os.File) (w, h int) { return 0, 0 }
