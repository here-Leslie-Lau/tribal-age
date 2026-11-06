package cli

import (
	"bufio"
	"bytes"
	"errors"
	"strings"
	"testing"
	"tribal-age/internal/game"
	"tribal-age/internal/savegame"
)

func TestStartupIOErrorsDoNotDiscardSave(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	store, err := savegame.OpenDefault()
	if err != nil {
		t.Fatal(err)
	}
	if err := savegame.EndMonth(game.New(42), store.Save); err != nil {
		t.Fatal(err)
	}
	store.Close()
	for _, input := range []string{"", "2\n", "garbage\n3\n"} {
		var out bytes.Buffer
		if err := Run([]string{"play", "--cli"}, strings.NewReader(input), &out); err != nil {
			t.Fatal(err)
		}
	}
	if err := Run([]string{"play"}, brokenReader{}, &bytes.Buffer{}); !errors.Is(err, errIO) {
		t.Fatal("startup lost scanner error")
	}
	if err := Run([]string{"play"}, strings.NewReader("1\n"), brokenWriter{}); !errors.Is(err, errIO) {
		t.Fatal("startup lost output error")
	}
	scanner := bufio.NewScanner(brokenReader{})
	if _, err := confirmLineNew(scanner, &bytes.Buffer{}); !errors.Is(err, errIO) {
		t.Fatal("confirmation lost read error")
	}
	if _, err := confirmLineNew(bufio.NewScanner(strings.NewReader("y\n")), brokenWriter{}); !errors.Is(err, errIO) {
		t.Fatal("confirmation lost write error")
	}
	store, err = savegame.OpenDefault()
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	loaded, err := store.Load()
	if err != nil || loaded.Seed != 42 || loaded.Calendar.Elapsed != 1 {
		t.Fatal("startup/EOF changed existing slot")
	}
}
func TestLineMonthSaveErrorDoesNotRepeatSettlement(t *testing.T) {
	store, err := savegame.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	s := game.New(42)
	var out bytes.Buffer
	if err := playScanner(s, bufio.NewScanner(strings.NewReader("end\nstatus\nquit\n")), &out, store.Save); err != nil {
		t.Fatal(err)
	}
	if s.Calendar.Elapsed != 1 || !strings.Contains(out.String(), "本月已结算") || !strings.Contains(out.String(), "第 1 年 4 月") {
		t.Fatal("failed persistence hid or repeated completed month")
	}
	t.Setenv("HOME", "")
	if err := Run([]string{"play"}, strings.NewReader(""), &out); err == nil {
		t.Fatal("storage-open error ignored")
	}
}
