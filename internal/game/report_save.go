package game

import "fmt"

func (r savedGame) restoreReports(s *State) error {
	if r.Version == 1 {
		if r.ReportsSince != nil || len(r.Reports) > 0 {
			return fmt.Errorf("版本1不能包含新格式战报")
		}
		s.ReportsSince = s.Calendar.Elapsed
		return nil
	}
	if r.ReportsSince == nil {
		return fmt.Errorf("存档缺少战报起始月份")
	}
	s.ReportsSince = *r.ReportsSince
	s.Reports = r.Reports
	return nil
}
