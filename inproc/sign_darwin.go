//go:build darwin

package inproc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ensureSigned gibt den entpackten dylibs eine gültige Ad-hoc-Signatur.
//
// macOS auf Apple Silicon verweigert das dlopen einer Bibliothek mit fehlender
// oder ungültiger Signatur und beendet dabei den GESAMTEN Prozess (SIGKILL,
// "Code Signature Invalid") — es gibt keine Fehlermeldung und keinen Go-Fehler.
// Solche ungültigen Signaturen entstehen z. B., wenn eine Homebrew-dylib per
// Byte-Patch relokiert wird (scripts/relocate_macos.py): das verändert die
// Datei NACH dem Signieren. Auch ein Cache, der von einer älteren Version
// stammt, kann so etwas enthalten.
//
// Ad-hoc-Signieren (codesign --sign -) ist billig und idempotent, daher wird
// jede Lib geprüft und nur bei Bedarf neu signiert. Best-effort: fehlt
// /usr/bin/codesign, geben wir nil zurück und lassen dlopen den echten Fehler
// melden.
func ensureSigned(dir string) error {
	libdir := filepath.Join(dir, "lib")
	entries, err := os.ReadDir(libdir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".dylib") {
			continue
		}
		p := filepath.Join(libdir, e.Name())
		if verifySigned(p) {
			continue
		}
		_ = exec.Command("codesign", "--force", "--sign", "-", p).Run()
	}
	return nil
}

// verifySigned meldet true, wenn p eine gültige Code-Signatur trägt.
func verifySigned(p string) bool {
	return exec.Command("codesign", "--verify", "--strict", p).Run() == nil
}
