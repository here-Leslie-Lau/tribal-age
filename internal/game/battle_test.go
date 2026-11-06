package game

import (
	"reflect"
	"testing"
)

func TestSimultaneousBattleHandExamples(t *testing.T) {
	for _, tt := range []struct {
		name         string
		forest       bool
		left, right  int
		lossA, lossB int
	}{
		{"equal grass", false, 10, 10, 3, 3}, {"forest defense", true, 10, 10, 2, 3}, {"outnumbered", false, 5, 10, 3, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := []fighter{{Owner: 0, People: tt.left, From: Point{2, 3}, To: Point{2, 3}, Supply: Supplied, Forest: tt.forest}, {Owner: 1, People: tt.right, From: Point{3, 3}, To: Point{2, 3}, Supply: Supplied}}
			before := append([]fighter(nil), f...)
			r := resolveClashes(f)
			if r.Losses[0] != tt.lossA || r.Losses[1] != tt.lossB || !r.Engaged[0] || !reflect.DeepEqual(f, before) {
				t.Fatalf("wrong simultaneous clash: %+v", r)
			}
			f[0], f[1] = f[1], f[0]
			swapped := resolveClashes(f)
			if swapped.Losses[1] != tt.lossA || swapped.Losses[0] != tt.lossB {
				t.Fatal("iteration order changed the result")
			}
		})
	}
}
func TestMeetingCrossingAndBlockedDeparture(t *testing.T) {
	cases := [][]fighter{
		{{Owner: 0, People: 10, From: Point{1, 2}, To: Point{2, 2}, Supply: Supplied}, {Owner: 1, People: 10, From: Point{3, 2}, To: Point{2, 2}, Supply: Supplied}},
		{{Owner: 0, People: 10, From: Point{1, 2}, To: Point{2, 2}, Supply: Supplied}, {Owner: 1, People: 10, From: Point{2, 2}, To: Point{1, 2}, Supply: Supplied}},
		{{Owner: 0, People: 10, From: Point{1, 2}, To: Point{2, 2}, Supply: Supplied}, {Owner: 1, People: 10, From: Point{2, 2}, To: Point{3, 2}, Supply: Supplied}, {Owner: 2, People: 10, From: Point{4, 2}, To: Point{3, 2}, Supply: Supplied}},
	}
	for _, f := range cases {
		r := resolveClashes(f)
		for i := range f {
			if !r.Engaged[i] || r.Positions[i] != f[i].From {
				t.Fatal("opposing forces passed through each other or entered a blocked departure")
			}
		}
	}
}
func TestMilitaryMarchExecutesOnlyAtMonthEnd(t *testing.T) {
	s := scenarioState()
	if err := s.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	target := Point{2, 2}
	if err := s.SetArmyOrder(0, 0, ArmyOrder{Kind: March, Target: target}); err != nil {
		t.Fatal(err)
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	a := s.Tribes[0].Armies[0]
	if a.Position != target || a.Order.Kind != Hold || s.Calendar.Elapsed != 1 || s.Tiles[2][2].Owner != Unowned {
		t.Fatal("march failed, repeated its order or automatically painted territory")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
}
