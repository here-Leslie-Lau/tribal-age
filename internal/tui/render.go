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
		">光标 @首都 P我方 2/3他部落 .草原 f森林 ^山 =河 ~海 ?未知",
	}
	if l.roomy {
		lines = []string{lines[0], "", lines[1], ""}
	}
	sidebar := m.sidebar(v, f)
	if l.roomy {
		sidebar = spacedSidebar(sidebar)
	}
	left, right := mapLines(v, m.cursor, l), wrapLines(sidebar, l.rightWidth)
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
		"Enter结束本月(需Y确认) | ?帮助 | Q退出(无存档)")
	return l.frame(lines)
}

func (m *model) sidebar(v game.View, f game.Forecast) []string {
	t, p := v.Self, f.Production
	name, policy := t.Name, "生产：自动（农田优先）"
	if !t.Alive() {
		name += " [灭亡]"
	}
	if m.dialog == productionDialog {
		name += " [草稿]"
	}
	if t.ForestPriority > 0 {
		policy = fmt.Sprintf("生产：优先采木%d格", t.ForestPriority)
	}
	growth := "增长：本月可积累"
	if f.Growth.Reason != "" {
		growth = "增长：" + f.Growth.Reason
	}
	hunger := fmt.Sprintf("连续缺粮：%d月", t.HungerMonths)
	if f.Shortage > 0 {
		hunger = fmt.Sprintf("缺粮预警 %d粮 / 连缺%d月", f.Shortage, t.HungerMonths)
	}
	return []string{name, fmt.Sprintf("人口：%d/%d", t.Population, f.Growth.Capacity),
		fmt.Sprintf("士兵%d / 平民%d", t.Soldiers, t.Population-t.Soldiers),
		fmt.Sprintf("粮食：%d (上月%+d)", t.Food, t.LastChange.Food), fmt.Sprintf("收支 %+d / 余粮 %d", f.NetFood, f.FoodAfter),
		fmt.Sprintf("木材：%d (上月%+d)", t.Wood, t.LastChange.Wood), fmt.Sprintf("本月预计 %+d", p.Wood),
		fmt.Sprintf("人手%d格：田%d 林%d", p.Slots, len(p.Farms), len(p.Forests)), policy,
		fmt.Sprintf("增长%d/%d / 安全粮%d", t.GrowthProgress, game.GrowthMonths, f.Growth.Reserve), hunger, growth}
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

func spacedSidebar(lines []string) []string {
	var out []string
	for i, line := range lines {
		out = append(out, line)
		if i == 0 || i == 2 || i == 6 {
			out = append(out, "")
		}
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
