package cli

import (
	"strings"
	"tribal-age/internal/game"
)

func renderMilitaryReports(s *game.State) string {
	var lines []string
	if s.ReportsSince > 0 {
		lines = append(lines, "旧档更早没有详细战报，不补造战果；原始可见消息：")
		lines = append(lines, s.VisibleEvents(0, 0)...)
	}
	history := s.MilitaryHistory(0)
	if len(history) == 0 {
		lines = append(lines, "暂无军事记录。")
	}
	for _, r := range history {
		if r.Month == s.Calendar.Elapsed {
			lines = append(lines, "本月未结算，退出不会保存这些操作记录。")
		}
		lines = append(lines, r.Lines()...)
	}
	return strings.Join(lines, "\n") + "\n"
}
