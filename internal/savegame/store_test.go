package savegame

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"tribal-age/internal/game"
)

func TestSingleSlotLifecycleAndWriterLock(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "tribal-age")
	a, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if _, err := a.Load(); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("opening a session created a save")
	}
	if b, err := Open(dir); err == nil {
		b.Close()
		t.Fatal("second writer acquired the same slot")
	}
	s := game.New(42)
	if err := EndMonth(s, a.Save); err != nil {
		t.Fatal(err)
	}
	loaded, err := a.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(s, loaded) {
		t.Fatal("saved state differs from month end")
	}
	info, err := os.Stat(filepath.Join(dir, "save.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatal("save is not private")
	}
	data, _ := os.ReadFile(filepath.Join(dir, "save.json"))
	// A new game is only an in-memory state until its first successful month.
	fresh := game.New(7)
	if _, err := a.Load(); err != nil {
		t.Fatal(err)
	}
	s.Tribes[0].Food = -1
	if err := a.Save(s); err == nil {
		t.Fatal("invalid state overwrote slot")
	}
	after, _ := os.ReadFile(filepath.Join(dir, "save.json"))
	if string(data) != string(after) {
		t.Fatal("rejected save changed old file")
	}
	if err := EndMonth(fresh, a.Save); err != nil {
		t.Fatal(err)
	}
	loaded, err = a.Load()
	if err != nil || loaded.Seed != 7 {
		t.Fatal("new game did not replace the slot at month end")
	}
	if b, err := Open(dir); err == nil {
		b.Close()
		t.Fatal("atomic replace lost writer lock")
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	b, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	if err := a.Save(fresh); err == nil {
		t.Fatal("closed session can write")
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 || entries[0].Name() != "save.json" {
		t.Fatal("left temporary or lock files in save directory")
	}
}
func TestWriteFailureKeepsOldFileAndNeverRepeatsMonth(t *testing.T) {
	a, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	s := game.New(42)
	if err := EndMonth(s, a.Save); err != nil {
		t.Fatal(err)
	}
	old, _ := a.Load()
	if err := a.Close(); err != nil {
		t.Fatal(err)
	}
	err = EndMonth(s, a.Save)
	if !errors.Is(err, ErrAutosave) || s.Calendar.Elapsed != 2 {
		t.Fatal("save failure rolled back or repeated month, or hid its status")
	}
	b, err := Open(a.root.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer b.Close()
	loaded, err := b.Load()
	if err != nil || !reflect.DeepEqual(old, loaded) {
		t.Fatal("failed save destroyed previous month")
	}
	s.Tribes[0].Population = 0
	for id := range s.Tribes {
		s.Tribes[id].Population = 0
	}
	calls := 0
	err = EndMonth(s, func(*game.State) error { calls++; return nil })
	if err == nil || calls != 0 {
		t.Fatal("rejected month wrote a save")
	}
}
func TestDefaultDirectoryAndCorruptFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	a, err := OpenDefault()
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	if a.root.Name() != filepath.Join(home, ".config", "tribal-age") {
		t.Fatal("ignored requested config location")
	}
	if err := os.WriteFile(filepath.Join(a.root.Name(), "save.json"), []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Load(); err == nil {
		t.Fatal("corrupt save silently became a new game")
	}
	if err := EndMonth(game.New(42), nil); err != nil {
		t.Fatal(err)
	}
}
