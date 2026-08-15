package cmd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"superdaemon/system"
)

func TestRootCommandUse(t *testing.T) {
	if rootCommand.Use != "superdaemon" {
		t.Fatalf("expected root command Use to be \"superdaemon\", got %q", rootCommand.Use)
	}
}

func TestVersionCommandOutput(t *testing.T) {
	var buf bytes.Buffer
	dummy := &cobra.Command{}
	dummy.SetOut(&buf)
	dummy.SetErr(&buf)
	versionCommand.Run(dummy, nil)
	out := buf.String()
	if !strings.Contains(out, "superdaemon v"+system.Version) {
		t.Fatalf("expected version output to contain \"superdaemon v%s\", got %q", system.Version, out)
	}
}
