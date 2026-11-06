//go:build unix

package savegame

import (
	"os"
	"syscall"
)

func lockDirectory(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}
