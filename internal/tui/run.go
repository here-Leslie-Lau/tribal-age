package tui

import (
	"fmt"
	"io"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/term"
	"tribal-age/internal/game"
)

// CanRun prevents raw/full-screen control sequences from reaching redirected IO.
func CanRun(input io.Reader, output io.Writer) bool {
	return isTerminal(input) && isTerminal(output)
}

func isTerminal(stream any) bool {
	file, ok := stream.(interface{ Fd() uintptr })
	return ok && term.IsTerminal(file.Fd())
}

func Run(s *game.State, input io.Reader, output io.Writer) error {
	if s == nil || input == nil || output == nil {
		return fmt.Errorf("游戏状态或终端输入输出为空")
	}
	if err := s.Validate(); err != nil {
		return fmt.Errorf("无法启动界面：%w", err)
	}
	p := tea.NewProgram(newModel(s), tea.WithInput(input), tea.WithOutput(output))
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("终端界面运行失败：%w", err)
	}
	return nil
}
