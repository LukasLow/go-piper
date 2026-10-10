//go:build !darwin

package inproc

// ensureSigned ist außerhalb von macOS ein No-op: andere Plattformen erzwingen
// beim dlopen keine gültige Code-Signatur.
func ensureSigned(string) error { return nil }
