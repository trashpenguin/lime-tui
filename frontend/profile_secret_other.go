//go:build !windows

package main

import "os"

// Other platforms are source compatibility only; the distributed release is Windows x64.
func readProfileSecret(path string) (string, error) {
	data, err := os.ReadFile(path)
	return string(data), err
}
func writeProfileSecret(path, value string) error { return os.WriteFile(path, []byte(value), 0600) }
