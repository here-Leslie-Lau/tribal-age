package savegame

import (
	"os"
	"path/filepath"
	"testing"
	"tribal-age/internal/game"
)

func TestWriterRemainsBoundToLockedDirectoryAfterPathReplacement(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "slot")
	first, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Close()
	a := game.New(42)
	if err := EndMonth(a, first.Save); err != nil {
		t.Fatal(err)
	}
	moved := dir + "-moved"
	if err := os.Rename(dir, moved); err != nil {
		t.Fatal(err)
	}
	second, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	b := game.New(7)
	if err := EndMonth(b, second.Save); err != nil {
		t.Fatal(err)
	}
	if err := EndMonth(a, first.Save); err != nil {
		t.Fatal(err)
	}
	untouched, err := second.Load()
	if err != nil || untouched.Seed != 7 {
		t.Fatal("old writer wrote through another instance's directory lock")
	}
	original, err := first.Load()
	if err != nil || original.Seed != 42 || original.Calendar.Elapsed != 2 {
		t.Fatal("writer lost its own directory identity")
	}
}
