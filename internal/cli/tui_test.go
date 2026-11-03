package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestExplicitCLIAndRedirectedInputKeepOldCommands(t *testing.T) {
	for _, args := range [][]string{{"play", "--cli", "--seed", "42"}, {"play", "--seed", "42"}} {
		var out bytes.Buffer
		if err := Run(args, strings.NewReader("status\nquit\n"), &out); err != nil {
			t.Fatalf("args %v: %v", args, err)
		}
		if !strings.Contains(out.String(), "粮食：60") || !strings.Contains(out.String(), "已退出") || strings.Contains(out.String(), "\x1b[") {
			t.Fatalf("old CLI became interactive fullscreen for %v", args)
		}
	}
}

func TestHelpExplainsLineModeAndSimulationRejectsItsFlag(t *testing.T) {
	var out bytes.Buffer
	if err := Run([]string{"play", "--help"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "--cli") {
		t.Fatal("help hides the old CLI mode")
	}
	if err := Run([]string{"simulate", "--cli"}, strings.NewReader(""), &out); err == nil {
		t.Fatal("simulation accepted a play-only flag")
	}
}
