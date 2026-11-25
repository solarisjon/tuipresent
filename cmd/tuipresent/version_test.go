package main

import (
	"bytes"
	"runtime/debug"
	"strings"
	"testing"
)

func info(ver string, kv ...string) func() (*debug.BuildInfo, bool) {
	return func() (*debug.BuildInfo, bool) {
		bi := &debug.BuildInfo{Main: debug.Module{Version: ver}}
		for i := 0; i+1 < len(kv); i += 2 {
			bi.Settings = append(bi.Settings, debug.BuildSetting{Key: kv[i], Value: kv[i+1]})
		}
		return bi, true
	}
}

func TestFormatVersionInjected(t *testing.T) {
	got := formatVersion("v1.0.0-rc1", "abcdef0123456789", "2026-01-02T03:04:05Z", "go1.27", info("(devel)"))
	want := "tuipresent v1.0.0-rc1 (commit abcdef0, built 2026-01-02T03:04:05Z, go1.27)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFormatVersionFromBuildInfo(t *testing.T) {
	got := formatVersion("", "", "", "go1.27", info("(devel)", "vcs.revision", "1234567890", "vcs.time", "T", "vcs.modified", "true"))
	want := "tuipresent dev (commit 1234567-dirty, built T, go1.27)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := formatVersion("", "", "", "go1.27", info("v1.2.3")); got != "tuipresent v1.2.3 (go1.27)" {
		t.Errorf("module version: %q", got)
	}
	none := func() (*debug.BuildInfo, bool) { return nil, false }
	if got := formatVersion("", "", "", "go1.27", none); got != "tuipresent dev (go1.27)" {
		t.Errorf("no build info: %q", got)
	}
}

func TestPrintVersionOneLine(t *testing.T) {
	var b bytes.Buffer
	printVersion(&b)
	s := b.String()
	if !strings.HasPrefix(s, "tuipresent ") || strings.Count(s, "\n") != 1 || !strings.Contains(s, "go1") {
		t.Errorf("unexpected -version output %q", s)
	}
}
