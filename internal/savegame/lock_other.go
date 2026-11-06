//go:build !unix

package savegame

import (
	"fmt"
	"os"
)

func lockDirectory(*os.File) error { return fmt.Errorf("当前平台尚不支持存档会话锁") }
