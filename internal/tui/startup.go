package tui

import (
	"errors"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"tribal-age/internal/game"
	"tribal-age/internal/savegame"
)

type startupMenu struct {
	seed       int64
	saved      *game.State
	problem    error
	selected   int
	confirming bool
}

func newSavedModel(seed int64, store *savegame.Store) (*model, error) {
	if store == nil {
		return nil, fmt.Errorf("存档会话为空")
	}
	saved, err := store.Load()
	if errors.Is(err, os.ErrClosed) {
		return nil, err
	}
	m := newModel(game.New(seed))
	m.save = store.Save
	if !errors.Is(err, os.ErrNotExist) {
		m.startup = &startupMenu{seed: seed, saved: saved, problem: err}
	}
	return m, nil
}
func (m *model) startupKey(key string) tea.Cmd {
	menu := m.startup
	if key == "q" || key == "ctrl+c" {
		return tea.Quit
	}
	if !m.fits() {
		return nil
	}
	if menu.confirming {
		m.confirmNewGame(key)
		return nil
	}
	switch key {
	case "up", "w":
		menu.selected = max(0, menu.selected-1)
	case "down", "s":
		menu.selected = min(2, menu.selected+1)
	case "enter":
		return m.chooseStartup()
	}
	return nil
}
func (m *model) chooseStartup() tea.Cmd {
	menu := m.startup
	switch menu.selected {
	case 0:
		if menu.saved != nil {
			m.startGame(menu.saved, menu.saved.Calendar.Elapsed)
		}
	case 1:
		menu.confirming = true
	case 2:
		return tea.Quit
	}
	return nil
}

func (m *model) confirmNewGame(key string) {
	if key == "y" {
		m.startGame(game.New(m.startup.seed), -1)
	} else if cancelsConfirmation(key, endDialog) {
		m.startup.confirming = false
	}
}
func (m *model) startGame(s *game.State, savedMonth int) {
	m.state = s
	m.cursor = s.Tribes[0].Capital
	m.startup = nil
	m.savedMonth = savedMonth
	m.messages = []string{"只在月底结算成功后自动存档；本月未结算操作和军令不会保留。"}
	m.record(max(0, len(s.Events)-messageLimit), nil)
}
func (m *model) startupScreen() string {
	menu := m.startup
	lines := []string{"部落纪元 / TRIBAL AGE", "", "选择继续游戏，或开始新的部落。", ""}
	for i, name := range []string{"继续游戏", "新游戏", "退出"} {
		prefix := "  "
		if menu.selected == i {
			prefix = "> "
		}
		lines = append(lines, prefix+name)
	}
	if menu.saved != nil {
		lines = append(lines, "", fmt.Sprintf("存档：%s", menu.saved.Calendar), fmt.Sprintf("人口%d / 种子%d", menu.saved.Tribes[0].Population, menu.saved.Seed))
	}
	if menu.problem != nil {
		lines = append(lines, "", fitRow("无法读取存档："+menu.problem.Error(), min(m.width, 160)), "原档保留；可确认新游戏，在首月结算后替换。")
	}
	if menu.confirming {
		lines = append(lines, "", "确认开始新游戏并替换旧存档？", "新局第一次月结成功才覆盖；此前旧档仍保留。", confirmHint)
	} else {
		lines = append(lines, "", "上下键选择，Enter确认；Q退出。")
	}
	return fitScreen(wrapLines(lines, min(m.width, 160)), min(m.width, 160), min(m.height, 48))
}
func (m *model) endMonth() {
	from := len(m.state.Events)
	month := m.state.Calendar.Elapsed
	err := savegame.EndMonth(m.state, m.save)
	if errors.Is(err, savegame.ErrAutosave) {
		m.saveFailed = true
		m.record(from, nil)
	} else if err == nil {
		m.saveFailed = false
		m.savedMonth = m.state.Calendar.Elapsed
	}
	m.record(from, err)
	if m.state.Calendar.Elapsed != month {
		if r, ok := m.settledReport(); ok && r.Destroyed() {
			m.openReports()
		}
	}
}
func (m *model) saveHint() string {
	if m.save == nil {
		return "当前会话不写存档"
	}
	if m.saveFailed {
		return "自动存档失败！退出不会重试保存"
	}
	if m.savedMonth < 0 {
		return "月末自动存档；首月前不写档"
	}
	return "月末自动存档；本月操作未保存"
}
