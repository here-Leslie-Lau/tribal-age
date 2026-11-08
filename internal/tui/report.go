package tui

import (
	"fmt"
	"tribal-age/internal/game"
)

func (m *model) openReports() {
	m.dialog = reportDialog
	m.reportIndex = len(m.state.MilitaryHistory(0)) - 1
	m.reportOffset = 0
}
func (m *model) reportBody() (string, []string) {
	history := m.state.MilitaryHistory(0)
	if m.reportIndex < 0 || len(history) == 0 {
		lines := []string{"暂无军事记录：仅记录本部落参战及部队变化，普通行军不生成战报。"}
		if m.state.ReportsSince > 0 {
			date := game.Calendar{Elapsed: m.state.ReportsSince}
			lines = append(lines, fmt.Sprintf("详细记录从第%d年%d月开始；更早不补造战果。", date.Year(), date.Month()), "原始可见消息（不是完整战报）：")
			lines = append(lines, m.state.VisibleEvents(0, 0)...)
		}
		return "原始记录", lines
	}
	index := min(m.reportIndex, len(history)-1)
	r := history[index]
	status := "已结算"
	if r.Month == m.state.Calendar.Elapsed {
		status = "本月未结算，退出不保留"
	}
	return fmt.Sprintf("%d/%d · %s", index+1, len(history), status), r.Lines()
}
func (m *model) reportRows() ([]string, int) {
	l := screenLayout(m.width, m.height, m.cursor)
	_, body := m.reportBody()
	return wrapLines(body, l.width-2*l.inset), l.height - 5
}
func (m *model) reportKey(key string) {
	if m.selectReport(key) {
		return
	}
	switch key {
	case "esc", "enter", "r":
		m.dialog = noDialog
		return
	case "up", "w":
		m.reportOffset--
	case "down", "s":
		m.reportOffset++
	case "pageup":
		m.reportOffset -= max(1, min(m.height, 48)-5)
	case "pagedown":
		m.reportOffset += max(1, min(m.height, 48)-5)
	case "home":
		m.reportOffset = 0
	case "end":
		m.reportOffset = int(^uint(0) >> 1)
	}
	lines, rows := m.reportRows()
	m.reportOffset = min(max(0, m.reportOffset), max(0, len(lines)-rows))
}

func (m *model) selectReport(key string) bool {
	history := m.state.MilitaryHistory(0)
	first := 0
	if m.state.ReportsSince > 0 || len(history) == 0 {
		first = -1
	}
	switch key {
	case "left", "a":
		m.reportIndex = max(first, m.reportIndex-1)
	case "right", "d":
		m.reportIndex = min(len(history)-1, m.reportIndex+1)
	default:
		return false
	}
	m.reportOffset = 0
	return true
}
func (m *model) reportScreen() string {
	l := screenLayout(m.width, m.height, m.cursor)
	title, _ := m.reportBody()
	body, rows := m.reportRows()
	start := min(max(0, m.reportOffset), max(0, len(body)-rows))
	lines := []string{"部落纪元 | 军事战报 | " + title, fmt.Sprintf("行%d–%d / %d；查看不花行动点、不推进月份。", start+1, min(start+rows, len(body)), len(body)), ""}
	for i := 0; i < rows; i++ {
		lines = append(lines, lineAt(body, start+i))
	}
	lines = append(lines, "←/→切报 ↑/↓滚动 PgUp/PgDn翻页 Enter/Esc/R返回", m.saveHint())
	return l.frame(lines)
}
func (m *model) settledReport() (game.MilitaryReport, bool) {
	history := m.state.MilitaryHistory(0)
	for i := len(history) - 1; i >= 0; i-- {
		if history[i].Month == m.state.Calendar.Elapsed-1 {
			return history[i], true
		}
	}
	return game.MilitaryReport{}, false
}
func (m *model) militarySummary() string {
	r, ok := m.settledReport()
	if !ok {
		return ""
	}
	date := game.Calendar{Elapsed: r.Month}
	return fmt.Sprintf("军事摘要 [R战报] 第%d年%d月：%s", date.Year(), date.Month(), r.Summary())
}
func (m *model) armyEndNote(slot int) string {
	history := m.state.MilitaryHistory(0)
	for i := len(history) - 1; i >= 0; i-- {
		for j := len(history[i].Changes) - 1; j >= 0; j-- {
			c := history[i].Changes[j]
			if c.Slot != slot {
				continue
			}
			if c.After != 0 {
				return "未编组"
			}
			status := "已覆灭"
			if c.Cause == game.Reorganized {
				status = "已解编"
			}
			if c.Cause == game.DefeatDisbanded {
				status = "战败解散"
			}
			date := game.Calendar{Elapsed: history[i].Month}
			return fmt.Sprintf("%s：%s（第%d年%d月，R查详情）", status, c.Cause, date.Year(), date.Month())
		}
	}
	return "未编组"
}
