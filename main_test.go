package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	orig := version
	defer func() { version = orig }()

	// -ldflags injected a real version: use it verbatim.
	version = "1.2.3"
	if got := resolveVersion(); got != "1.2.3" {
		t.Errorf("resolveVersion() with ldflags = %q, want %q", got, "1.2.3")
	}
	version = "0.2.1-rc.1"
	if got := resolveVersion(); got != "0.2.1-rc.1" {
		t.Errorf("resolveVersion() with ldflags = %q, want %q", got, "0.2.1-rc.1")
	}

	// No ldflags. Under `go test` the build info's Main.Version is
	// "(devel)", which resolveVersion treats as absent, so it falls
	// back to the "dev" sentinel rather than leaking "(devel)".
	version = "dev"
	if got := resolveVersion(); got != "dev" {
		t.Errorf("resolveVersion() without ldflags = %q, want %q", got, "dev")
	}
}

func TestPrintVersion(t *testing.T) {
	orig := version
	defer func() { version = orig }()
	version = "9.9.9"

	var buf bytes.Buffer
	printVersion(&buf)
	out := buf.String()

	if !strings.HasPrefix(out, "umerge 9.9.9\n") {
		t.Errorf("printVersion() = %q, want it to start with %q", out, "umerge 9.9.9\n")
	}
	if !strings.Contains(out, "BSD 3-Clause License") {
		t.Errorf("printVersion() = %q, want it to mention the license", out)
	}
}
