package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"tribal-age/internal/game"
	"tribal-age/internal/savegame"
)

func saveStore(t *testing.T) *savegame.Store {
	t.Helper()
	s, err := savegame.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}
func TestAutosaveOnlyAfterConfirmedMonth(t *testing.T) {
	store := saveStore(t)
	m, err := newSavedModel(42, store)
	if err != nil {
		t.Fatal(err)
	}
	press(m, "b", "up", "e", "enter", "enter", "q", "n")
	if _, err := store.Load(); !os.IsNotExist(err) {
		t.Fatal("operation or canceled month created save")
	}
	press(m, "enter", "y")
	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(m.state, loaded) || loaded.Calendar.Elapsed != 1 {
		t.Fatal("did not save complete month exactly once")
	}
	press(m, "c", "m", "esc", "q", "y")
	latest, err := store.Load()
	if err != nil || !reflect.DeepEqual(latest, loaded) {
		t.Fatal("mid-month action or exit overwrote save")
	}
}
func TestStartupContinueNewConfirmationAndDelayedReplacement(t *testing.T) {
	store := saveStore(t)
	old := game.New(7)
	if err := savegame.EndMonth(old, store.Save); err != nil {
		t.Fatal(err)
	}
	m, err := newSavedModel(42, store)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(m.View().Content, "继续游戏") || !strings.Contains(m.View().Content, "新游戏") {
		t.Fatal("no startup choice")
	}
	press(m, "enter")
	if m.startup != nil || !reflect.DeepEqual(m.state, old) {
		t.Fatal("continue reset seed, AP, events or state")
	}
	m, err = newSavedModel(42, store)
	if err != nil {
		t.Fatal(err)
	}
	press(m, "down", "enter", "enter")
	if m.startup == nil {
		t.Fatal("Enter accidentally confirmed overwrite")
	}
	press(m, "enter", "n")
	if m.startup == nil {
		t.Fatal("cancel did not return to menu")
	}
	// Selection remains on new game after cancel.
	press(m, "enter", "y")
	if m.startup != nil || m.state.Seed != 42 || m.state.Calendar.Elapsed != 0 {
		t.Fatal("confirmed new game did not start")
	}
	latest, _ := store.Load()
	if !reflect.DeepEqual(latest, old) {
		t.Fatal("starting new game destroyed old save before first month")
	}
	press(m, "enter", "y")
	latest, err = store.Load()
	if err != nil || latest.Seed != 42 || latest.Calendar.Elapsed != 1 {
		t.Fatal("new first month did not replace old save")
	}
}
func TestSaveFailureReportsCompletedMonthAndKeepsWarning(t *testing.T) {
	store := saveStore(t)
	m, err := newSavedModel(42, store)
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	press(m, "enter", "y")
	if m.state.Calendar.Elapsed != 1 || !strings.Contains(m.View().Content, "自动存档失败") {
		t.Fatal("save error hid completed month or lost warning")
	}
	press(m, "up", "e")
	if !strings.Contains(m.View().Content, "自动存档失败") {
		t.Fatal("ordinary action concealed unsaved-month warning")
	}
}
func TestCorruptStartupCannotContinueOrOverwriteWithoutConfirmation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "save.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := savegame.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	m, err := newSavedModel(42, store)
	if err != nil {
		t.Fatal(err)
	}
	press(m, "enter")
	if m.startup == nil || !strings.Contains(m.View().Content, "无法读取存档") {
		t.Fatal("corrupt save silently started game")
	}
	press(m, "down", "enter", "esc", "q")
	data, _ := os.ReadFile(path)
	if string(data) != "broken" {
		t.Fatal("cancel/quit destroyed damaged archive")
	}
}

func TestSaveWarningClearsWhenNextMonthSucceeds(t *testing.T) {
	dir := t.TempDir()
	store, err := savegame.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	m, err := newSavedModel(42, store)
	if err != nil {
		t.Fatal(err)
	}
	obstruction := filepath.Join(dir, "save.json")
	if err := os.Mkdir(obstruction, 0700); err != nil {
		t.Fatal(err)
	}
	press(m, "enter", "y")
	if !m.saveFailed || m.state.Calendar.Elapsed != 1 {
		t.Fatal("failed file replacement did not retain warning")
	}
	if err := os.Remove(obstruction); err != nil {
		t.Fatal(err)
	}
	press(m, "enter", "y")
	loaded, err := store.Load()
	if err != nil || !reflect.DeepEqual(loaded, m.state) || loaded.Calendar.Elapsed != 2 || m.saveFailed {
		t.Fatal("retry on next month did not save current complete state or clear warning")
	}
}
