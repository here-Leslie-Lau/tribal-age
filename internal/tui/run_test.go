package tui

import (
	"bytes"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"tribal-age/internal/game"
)

func TestNonTerminalStreamsDoNotStartFullScreen(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "not-a-terminal")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	for _, pair := range []struct {
		input  *os.File
		output *os.File
	}{{file, file}, {file, os.Stdout}, {os.Stdin, file}} {
		if CanRun(pair.input, pair.output) {
			t.Fatal("a redirected file was treated as a terminal")
		}
	}
	if CanRun(strings.NewReader("q"), &bytes.Buffer{}) || CanRun(file, &bytes.Buffer{}) {
		t.Fatal("non-file streams were treated as a terminal")
	}
}

func TestRunnerRejectsInvalidGameBeforeTerminalSetup(t *testing.T) {
	var out bytes.Buffer
	if err := Run(nil, strings.NewReader(""), &out); err == nil {
		t.Fatal("nil game accepted")
	}
	s := game.New(42)
	s.Tribes[0].Food = -1
	if err := Run(s, strings.NewReader(""), &out); err == nil || out.Len() != 0 {
		t.Fatal("invalid game entered terminal mode")
	}
}

func TestRunnerProcessesQuitAndRestoresScreen(t *testing.T) {
	var out bytes.Buffer
	s := game.New(42)
	before := snapshot(s)
	if err := Run(s, strings.NewReader("qy"), &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "\x1b[?1049l") {
		t.Fatal("runtime did not leave the alternate screen")
	}
	if !reflect.DeepEqual(before, *s) {
		t.Fatal("runtime changed game state while quitting")
	}
}

type errorReader struct{ err error }

func (r errorReader) Read([]byte) (int, error) { return 0, r.err }

func TestRunnerReportsInputFailure(t *testing.T) {
	problem := errors.New("test terminal disconnected")
	var out bytes.Buffer
	if err := Run(game.New(42), errorReader{problem}, &out); !errors.Is(err, problem) {
		t.Fatalf("terminal error was lost: %v", err)
	}
}
