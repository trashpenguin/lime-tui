//go:build !windows

package main

import "os"

func launchDefaultWindow() bool {
	if len(os.Args) == 2 && (os.Args[1] == "--window" || os.Args[1] == "--terminal-window") {
		os.Args = os.Args[:1]
	}
	return false
}
