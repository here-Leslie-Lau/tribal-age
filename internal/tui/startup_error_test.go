package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"tribal-age/internal/savegame"
)

func TestDamagedArchiveErrorCannotHideOverwriteConfirmation(t *testing.T) {
	dir := t.TempDir()
	bad := `{"` + strings.Repeat("x", 10000) + `":true}`
	if err := os.WriteFile(filepath.Join(dir, "save.json"), []byte(bad), 0600); err != nil {
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
	press(m, "down", "enter")
	if !strings.Contains(m.View().Content, "确认开始新游戏") || !strings.Contains(m.View().Content, "[Y]确认") {
		t.Fatal("untrusted error displaced destructive-action confirmation")
	}
}
