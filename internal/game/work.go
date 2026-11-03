package game

import "fmt"

// SetForestPriority changes monthly staffing, not the action or production clock.
// Zero restores automatic farm-first allocation; repeats do not add events.
func (s *State) SetForestPriority(id, forests int) error {
	if id < 0 || id >= TribeCount {
		return fmt.Errorf("部落编号无效")
	}
	t := &s.Tribes[id]
	if !t.Alive() {
		return fmt.Errorf("部落已经灭亡")
	}
	if !validForestPriority(forests) {
		return fmt.Errorf("优先采木数量必须在 0 到 %d 之间", Width*Height)
	}
	if t.ForestPriority == forests {
		return nil
	}
	t.ForestPriority = forests
	message := "生产安排：恢复自动，先种田，再采木。"
	if forests > 0 {
		message = fmt.Sprintf("生产安排：优先经营 %d 格森林，剩余人手自动安排。", forests)
	}
	s.emitOwn(id, message)
	return nil
}

func validForestPriority(forests int) bool {
	return forests >= 0 && forests <= Width*Height
}

func validateWorkPlan(t Tribe) error {
	if !validForestPriority(t.ForestPriority) || (!t.Alive() && t.ForestPriority != 0) {
		return fmt.Errorf("生产安排无效或灭亡后未清理")
	}
	return nil
}
