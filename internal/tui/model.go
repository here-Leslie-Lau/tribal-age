package tui

import (
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
	"tribal-age/internal/game"
)

const (
	minWidth     = 80
	minHeight    = 24
	messageLimit = 64
)

type dialog uint8

const (
	noDialog dialog = iota
	endDialog
	quitDialog
	trainingDialog
	productionDialog
	helpDialog
	territoryDialog
	armyDialog
)

type tileBinding struct {
	key   string
	kind  game.ActionKind
	label string
	wood  int
}

// Read-only bindings shared by input and the contextual action hints.
var tileBindings = [...]tileBinding{
	{"e", game.Explore, "探索", 0},
	{"c", game.Claim, "占领", game.ClaimWood},
	{"b", game.BuildFarm, "建田", game.FarmWood},
}

// Only Update writes game state. Views and production drafts own copied data.
type model struct {
	state          *game.State
	cursor         game.Point
	width, height  int
	dialog         dialog
	workTarget     int
	territoryIndex int
	armySlot       int
	armyPeople     int
	mapView        mapStyle
	plainMap       bool
	messages       []string
	startup        *startupMenu
	save           func(*game.State) error
	savedMonth     int
	saveFailed     bool
}

func newModel(s *game.State) *model {
	return &model{state: s, cursor: s.Tribes[0].Capital, width: minWidth, height: minHeight, savedMonth: -1,
		plainMap: os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb",
		messages: []string{"方向键选地块；光标和菜单不花行动点，只有确认结束月份才推进。"}}
}

func (m *model) Init() tea.Cmd { return nil }

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = max(0, msg.Width), max(0, msg.Height)
	case tea.ColorProfileMsg:
		if !m.plainMap {
			m.mapView.profile = msg.Profile
		}
	case tea.KeyPressMsg:
		cmd = m.handleKey(strings.ToLower(msg.String()))
	}
	return m, cmd
}

func (m *model) fits() bool { return m.width >= minWidth && m.height >= minHeight }

func (m *model) handleKey(key string) tea.Cmd {
	if m.startup != nil {
		return m.startupKey(key)
	}
	if key == "q" || key == "ctrl+c" {
		m.dialog = quitDialog
		return nil
	}
	if m.dialog == quitDialog {
		return m.confirm(key)
	}
	if !m.fits() {
		return nil
	}
	if m.dialog == productionDialog {
		m.editProduction(key)
		return nil
	}
	if m.dialog == territoryDialog {
		m.territoryKey(key)
		return nil
	}
	if m.dialog == armyDialog {
		m.armyKey(key)
		return nil
	}
	if m.dialog != noDialog {
		return m.confirm(key)
	}
	m.gameKey(key)
	return nil
}

func (m *model) gameKey(key string) {
	if delta, ok := movement(key); ok {
		m.cursor.X = min(game.Width-1, max(0, m.cursor.X+delta.X))
		m.cursor.Y = min(game.Height-1, max(0, m.cursor.Y+delta.Y))
		return
	}
	if m.armyGameKey(key) {
		return
	}
	for _, binding := range tileBindings {
		if key == binding.key {
			m.apply(binding.kind)
			return
		}
	}
	m.controlKey(key)
}

func (m *model) controlKey(key string) {
	switch key {
	case "enter":
		m.dialog = endDialog
	case "t":
		m.dialog = trainingDialog
	case "p":
		m.dialog, m.workTarget = productionDialog, m.state.Tribes[0].ForestPriority
	case "?":
		m.dialog = helpDialog
	case "l":
		m.openTerritories()
	case "v":
		m.mapView.terrain = !m.mapView.terrain
	}
}

func movement(key string) (game.Point, bool) {
	switch key {
	case "up", "w":
		return game.Point{Y: -1}, true
	case "down", "s":
		return game.Point{Y: 1}, true
	case "left", "a":
		return game.Point{X: -1}, true
	case "right", "d":
		return game.Point{X: 1}, true
	}
	return game.Point{}, false
}

func (m *model) apply(kind game.ActionKind) {
	from := len(m.state.Events)
	m.record(from, m.state.Apply(0, game.Action{Kind: kind, Target: m.cursor}))
}

func (m *model) record(from int, err error) {
	if err != nil {
		m.messages = append(m.messages, "操作失败："+err.Error())
	} else {
		m.messages = append(m.messages, m.state.VisibleEvents(0, from)...)
	}
	if len(m.messages) > messageLimit {
		m.messages = m.messages[len(m.messages)-messageLimit:]
	}
}

func (m *model) playerView() game.View {
	v, _ := m.state.View(0) // The player ID is fixed and valid.
	if m.dialog == productionDialog {
		v.Self.ForestPriority = m.workTarget
	}
	return v
}

func (m *model) View() tea.View {
	v := tea.NewView(m.screen())
	v.AltScreen = true
	return v
}
