package savegame

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
	"tribal-age/internal/game"
)

func TestDirectoryLockReleasedAfterCrash(t *testing.T) {
	if dir := os.Getenv("TRIBAL_SAVE_LOCK_TEST"); dir != "" {
		store, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		fmt.Println("locked")
		var data [1]byte
		os.Stdin.Read(data[:])
		return
	}
	dir := t.TempDir()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestDirectoryLockReleasedAfterCrash$")
	cmd.Env = append(os.Environ(), "TRIBAL_SAVE_LOCK_TEST="+dir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	defer stdin.Close()
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	ready, err := bufio.NewReader(stdout).ReadString('\n')
	if err != nil || ready != "locked\n" {
		t.Fatalf("lock child did not become ready: %s %v", ready, err)
	}
	if store, err := Open(dir); err == nil {
		store.Close()
		t.Fatal("different process bypassed writer lock")
	}
	if err := cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	cmd.Wait()
	store, err := Open(dir)
	if err != nil {
		t.Fatal("crashed process left a stale lock:", err)
	}
	defer store.Close()
}
func TestFilesystemFailuresPreserveDataAndCleanTemporaryFiles(t *testing.T) {
	t.Run("directory moved", func(t *testing.T) {
		dir := filepath.Join(t.TempDir(), "slot")
		store, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		s := game.New(42)
		if err := EndMonth(s, store.Save); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(dir, "save.json"))
		if err != nil {
			t.Fatal(err)
		}
		moved := dir + "-moved"
		if err := os.Rename(dir, moved); err != nil {
			t.Fatal(err)
		}
		if err := EndMonth(s, store.Save); err != nil {
			t.Fatal("rooted writer failed to follow its locked directory:", err)
		}
		after, err := os.ReadFile(filepath.Join(moved, "save.json"))
		if err != nil || string(after) == string(data) {
			t.Fatal("rooted writer lost its original save directory")
		}
	})
	t.Run("rename refused", func(t *testing.T) {
		dir := t.TempDir()
		store, err := Open(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		if err := os.Mkdir(filepath.Join(dir, "save.json"), 0700); err != nil {
			t.Fatal(err)
		}
		if err := EndMonth(game.New(42), store.Save); err == nil {
			t.Fatal("could not replace destination but reported success")
		}
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) != 1 {
			t.Fatal("left partially written temporary files after rename error")
		}
	})
}
func TestStoreEntryFailures(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ordinary-file")
	if err := os.WriteFile(path, []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if s, err := Open(path); err == nil {
		s.Close()
		t.Fatal("file accepted as directory")
	}
	t.Setenv("HOME", "")
	if s, err := OpenDefault(); err == nil {
		s.Close()
		t.Fatal("missing home was accepted")
	}
	store, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	store.Close()
	if _, err := store.Load(); err == nil {
		t.Fatal("closed store can read")
	}
}
