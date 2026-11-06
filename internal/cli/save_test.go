package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"tribal-age/internal/game"
)

func runSavedLine(t *testing.T, seed, input string) string {
	t.Helper()
	var out bytes.Buffer
	if err := Run([]string{"play", "--cli", "--seed", seed}, strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	return out.String()
}
func loadSlot(t *testing.T, home string) (*game.State, []byte) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(home, ".config", "tribal-age", "save.json"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := game.UnmarshalSave(data)
	if err != nil {
		t.Fatal(err)
	}
	return s, data
}
func TestRunSavesMonthAndRestoresWithoutUsingNewSeed(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	runSavedLine(t, "42", "farm 3 4\nend\nwork 1\nquit\n")
	saved, data := loadSlot(t, home)
	if saved.Calendar.Elapsed != 1 || saved.Tribes[0].ForestPriority != 0 || saved.Tribes[0].AP != game.MonthlyAP {
		t.Fatal("saved partial month or lost complete settlement")
	}
	text := runSavedLine(t, "7", "1\nstatus\nquit\n")
	restored, after := loadSlot(t, home)
	if !reflect.DeepEqual(saved, restored) || string(data) != string(after) || !strings.Contains(text, "继续游戏") || !strings.Contains(text, "第 1 年 4 月") {
		t.Fatal("continue reset seed or wrote save on exit")
	}
	runSavedLine(t, "7", "2\nn\n3\n")
	_, after = loadSlot(t, home)
	if string(data) != string(after) {
		t.Fatal("cancel new game overwrote save")
	}
	runSavedLine(t, "7", "2\ny\nquit\n")
	_, after = loadSlot(t, home)
	if string(data) != string(after) {
		t.Fatal("new game replaced old slot before first month")
	}
	runSavedLine(t, "7", "2\ny\nend\nquit\n")
	fresh, _ := loadSlot(t, home)
	if fresh.Seed != 7 || fresh.Calendar.Elapsed != 1 {
		t.Fatal("new first month failed to replace save")
	}
}
func TestRunKeepsCorruptSlotAndSimulationNeverTouchesIt(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "tribal-age")
	var out bytes.Buffer
	if err := Run([]string{"simulate", "--months", "1"}, strings.NewReader(""), &out); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("simulation accessed player config")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "save.json")
	if err := os.WriteFile(path, []byte("broken"), 0600); err != nil {
		t.Fatal(err)
	}
	text := runSavedLine(t, "42", "1\n2\nn\n3\n")
	data, _ := os.ReadFile(path)
	if !strings.Contains(text, "无法读取存档") || string(data) != "broken" {
		t.Fatal("invalid slot silently discarded")
	}
	runSavedLine(t, "42", "")
	data, _ = os.ReadFile(path)
	if string(data) != "broken" {
		t.Fatal("EOF overwrote old slot")
	}
}
