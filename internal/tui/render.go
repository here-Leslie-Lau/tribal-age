package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
	"tribal-age/internal/game"
)

const mapCellWidth = 4

type layout struct {
	width, height, leftWidth, rightWidth int
	left, top, cols, rows, panelRows     int
	inset                                int
	gap                                  string
	roomy                                bool
}

func screenLayout(width, height int, cursor game.Point) layout {
	// Bound rendering work even if a terminal reports implausibly large sizes.
	width, height = min(width, 160), min(height, 48)
	inset, rightMin, extra := 0, 26, 0
	gap := "  |  "
	roomy := height >= 32
	if roomy {
		extra = 4
	}
	if width >= 100 {
		inset, rightMin = 2, 30
		gap = "   |   "
	}
	usable := width - 2*inset
	leftWidth := min(4+game.Width*mapCellWidth, usable-rightMin-len(gap))
	cols, rows := min(game.Width, (leftWidth-4)/mapCellWidth), min(game.Height, height-13-extra)
	return layout{width: width, height: height, leftWidth: leftWidth, rightWidth: usable - leftWidth - len(gap),
		left: min(max(0, cursor.X-cols/2), game.Width-cols), top: min(max(0, cursor.Y-rows/2), game.Height-rows),
		cols: cols, rows: rows, panelRows: height - 9 - extra - rows, inset: inset, gap: gap, roomy: roomy}
}

func (m *model) screen() string {
	if !m.fits() {
		return m.smallScreen()
	}
	if m.dialog == territoryDialog {
		return m.territoryScreen()
	}
	l := screenLayout(m.width, m.height, m.cursor)
	v := m.playerView()
	f := game.ForecastFor(v)
	lines := []string{
		fmt.Sprintf("部落纪元 | 行动点：%d/%d | %s", v.Self.AP, game.MonthlyAP, v.Calendar),
		m.mapView.legend(),
	}
	if l.roomy {
		lines = []string{lines[0], "", lines[1], ""}
	}
	left, right := mapLines(v, m.cursor, l, m.mapView), m.sidebarLines(v, f, l)
	for i := 0; i < l.rows+2; i++ {
		lines = append(lines, fitRow(lineAt(left, i), l.leftWidth)+l.gap+fitRow(lineAt(right, i), l.rightWidth))
	}
	lines = append(lines, strings.Repeat("-", l.width), selectedInfo(v, m.cursor), actionHints(v, m.cursor))
	if l.roomy {
		lines = append(lines, "")
	}
	panel := m.panelLines(v, f, l.width, l.panelRows)
	for i := 0; i < l.panelRows; i++ {
		lines = append(lines, lineAt(panel, i))
	}
	if l.roomy {
		lines = append(lines, "")
	}
	lines = append(lines, "方向键/WASD移动 | E探索 C占领 B建田 | T训练 P生产 L领地",
		"Enter结束本月(需Y确认) | V切换视图 ?帮助 | Q退出(无存档)")
	return l.frame(lines)
}

func (m *model) smallScreen() string {
	lines := []string{fmt.Sprintf("终端太小：当前%d列%d行", m.width, m.height), "请放大到至少80列24行；游戏已暂停操作。", "Q退出（无存档）；放大窗口后继续。"}
	if m.dialog == quitDialog {
		lines = []string{"退出游戏？尚不支持存档。", "[Y]确认退出 [N/Esc/Enter]取消"}
	}
	return fitScreen(lines, min(m.width, 160), min(m.height, 48))
}

func lineAt(lines []string, i int) string {
	if i < len(lines) {
		return lines[i]
	}
	return ""
}

func fitRow(line string, width int) string {
	line = ansi.Truncate(line, width, "")
	return line + strings.Repeat(" ", max(0, width-ansi.StringWidth(line)))
}

func fitScreen(lines []string, width, height int) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	out := make([]string, 0, height)
	for i := 0; i < height; i++ {
		out = append(out, fitRow(lineAt(lines, i), width))
	}
	return strings.Join(out, "\n")
}

func wrapLines(lines []string, width int) []string {
	var out []string
	for _, line := range lines {
		out = append(out, strings.Split(ansi.Hardwrap(line, width, true), "\n")...)
	}
	return out
}

func (l layout) frame(lines []string) string {
	if l.inset > 0 {
		for i, line := range lines {
			lines[i] = strings.Repeat(" ", l.inset) + ansi.Truncate(line, l.width-2*l.inset, "")
		}
	}
	return fitScreen(lines, l.width, l.height)
}
