// Package savegame owns the single on-disk slot; game rules never access files.
package savegame

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"tribal-age/internal/game"
)

// Store is owned by one play session. Its directory lock survives save-file
// replacement and is released by the OS on process exit, including a crash.
type Store struct {
	directory *os.File
	root      *os.Root
	closed    bool
}

func OpenDefault() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return Open(filepath.Join(home, ".config", "tribal-age"))
}
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("无法创建存档目录：%w", err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	f, err := root.Open(".")
	if err != nil {
		root.Close()
		return nil, err
	}
	if err := lockDirectory(f); err != nil {
		f.Close()
		root.Close()
		return nil, fmt.Errorf("无法锁定存档目录，可能已有游戏正在运行：%w", err)
	}
	return &Store{directory: f, root: root}, nil
}
func (s *Store) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return errors.Join(s.directory.Close(), s.root.Close())
}
func (s *Store) Load() (*game.State, error) {
	if s.closed {
		return nil, os.ErrClosed
	}
	f, err := s.root.Open("save.json")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data, err := io.ReadAll(io.LimitReader(f, game.MaxSaveBytes+1))
	if err != nil {
		return nil, err
	}
	return game.UnmarshalSave(data)
}
func (s *Store) Save(state *game.State) error {
	if s.closed {
		return os.ErrClosed
	}
	data, err := state.MarshalSave()
	if err != nil {
		return err
	}
	if len(data) > game.MaxSaveBytes {
		return fmt.Errorf("存档超过大小限制")
	}
	return s.replace(data)
}
func (s *Store) replace(data []byte) error {
	name := ".save-" + rand.Text() + ".tmp"
	f, err := s.root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer s.root.Remove(name)
	defer f.Close()
	if _, err = f.Write(data); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = s.root.Rename(name, "save.json"); err != nil {
		return err
	}
	if err = s.directory.Sync(); err != nil {
		return fmt.Errorf("存档已替换，但目录同步失败，持久性未确认：%w", err)
	}
	return nil
}

var ErrAutosave = errors.New("本月已结算，但自动存档失败；存档未确认，退出不会重试保存")

// EndMonth is the sole autosave trigger, shared by the two play interfaces.
// Persistence failure must not undo or repeat an already completed game month.
func EndMonth(s *game.State, save func(*game.State) error) error {
	if err := s.EndMonth(game.Interactive); err != nil {
		return err
	}
	if save != nil {
		if err := save(s); err != nil {
			return fmt.Errorf("%w：%w", ErrAutosave, err)
		}
	}
	return nil
}
