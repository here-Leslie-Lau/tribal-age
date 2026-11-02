package game

import "fmt"

func (s *State) emitOwn(id int, text string) {
	s.Events = append(s.Events, Event{Month: s.Calendar.Elapsed, Audience: 1 << id, Text: text})
}

func (s *State) emitAt(id int, p Point, text string) {
	audience := uint8(1 << id)
	for observer, t := range s.Tribes {
		if t.Explored[p.Y][p.X] {
			audience |= 1 << observer
		}
	}
	s.Events = append(s.Events, Event{Month: s.Calendar.Elapsed, Audience: audience, Text: text})
}

func (s *State) VisibleEvents(id, from int) []string {
	if id < 0 || id >= TribeCount {
		return nil
	}
	if from < 0 {
		from = 0
	}
	if from > len(s.Events) {
		return nil
	}
	var result []string
	for _, e := range s.Events[from:] {
		if e.Audience&(1<<id) != 0 {
			date := Calendar{Elapsed: e.Month}
			result = append(result, fmt.Sprintf("第%d年%d月：%s", date.Year(), date.Month(), e.Text))
		}
	}
	return result
}
