package game

import (
	"reflect"
	"testing"
)

func TestArmyOperationBoundaryReasonsAreAtomic(t *testing.T) {
	for _, tt := range []struct {
		name  string
		setup func(*State)
		act   func(*State) error
	}{
		{"unknown order", nil, func(s *State) error { return s.SetArmyOrder(0, 0, ArmyOrder{Kind: 99}) }},
		{"friendly occupation", nil, func(s *State) error { return s.SetArmyOrder(0, 0, ArmyOrder{Kind: Occupy}) }},
		{"neutral occupation", func(s *State) { s.Tribes[0].Armies[0].Position = Point{2, 2} }, func(s *State) error { return s.SetArmyOrder(0, 0, ArmyOrder{Kind: Occupy}) }},
		{"unknown route", func(s *State) { s.Tribes[0].Explored[2][2] = false }, func(s *State) error { return s.SetArmyOrder(0, 0, ArmyOrder{Kind: March, Target: Point{2, 2}}) }},
		{"mountain route", func(s *State) { s.Tiles[2][2].Terrain = Mountain }, func(s *State) error { return s.SetArmyOrder(0, 0, ArmyOrder{Kind: March, Target: Point{2, 2}}) }},
		{"empty attack", nil, func(s *State) error { return s.SetArmyOrder(0, 0, ArmyOrder{Kind: Attack, Target: Point{2, 2}}) }},
		{"enemy march", func(s *State) { s.Tribes[1].Armies[0] = Army{People: 5, Position: Point{2, 2}}; s.revealArmies(1) }, func(s *State) error { return s.SetArmyOrder(0, 0, ArmyOrder{Kind: March, Target: Point{2, 2}}) }},
		{"no resize AP", func(s *State) { s.Tribes[0].AP = 0 }, func(s *State) error { return s.SetArmySize(0, 0, 4) }},
		{"remote resize", func(s *State) { s.Tribes[0].Armies[0].Position = Point{2, 2} }, func(s *State) error { return s.SetArmySize(0, 0, 4) }},
		{"occupied capital", func(s *State) {
			s.Tribes[0].Soldiers = 10
			s.Tribes[0].Armies[0].Position = Point{2, 2}
			s.Tribes[1].Armies[0] = Army{People: 5, Position: Point{2, 3}}
			s.revealArmies(1)
		}, func(s *State) error { return s.SetArmySize(0, 1, 3) }},
		{"inactive actor", func(s *State) {
			s.Tribes[0].Defeated = true
			s.Tribes[0].Armies = [MaxArmies]Army{}
			s.Tribes[0].Soldiers = 0
			s.Tribes[0].Food = 0
			s.Tribes[0].Wood = 0
			s.Tribes[0].AP = 0
			s.Tiles[3][2].Owner = Unowned
		}, func(s *State) error { return s.SetArmySize(0, 0, 1) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := scenarioState()
			if err := s.SetArmySize(0, 0, 5); err != nil {
				t.Fatal(err)
			}
			if tt.setup != nil {
				tt.setup(s)
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
			before := *s
			before.Events = append([]Event(nil), s.Events...)
			if tt.act(s) == nil || !reflect.DeepEqual(before, *s) {
				t.Fatal("invalid command changed military state, resources, fog or events")
			}
		})
	}
}
func TestArmyCancelDisbandAndDuplicateOccupation(t *testing.T) {
	s := scenarioState()
	if err := s.SetArmySize(0, 0, 5); err != nil {
		t.Fatal(err)
	}
	for _, o := range []ArmyOrder{{Kind: March, Target: Point{2, 2}}, {Kind: Hold}} {
		if err := s.SetArmyOrder(0, 0, o); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetArmySize(0, 0, 0); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].ReserveSoldiers() != 5 || s.Tribes[0].Armies[0] != (Army{}) || s.Tribes[0].Population != 30 {
		t.Fatal("disbanding killed or duplicated soldiers")
	}
	for _, k := range []ArmyOrderKind{Hold, March, Attack, Occupy, 255} {
		if k.String() == "" {
			t.Fatal("missing order explanation")
		}
	}
	s = invaderOnCapital()
	p := s.Tribes[1].Capital
	s.Tribes[0].Armies[0].People = 2
	s.Tribes[0].Armies[1] = Army{People: 3, Position: p}
	for slot := 0; slot < 2; slot++ {
		if err := s.SetArmyOrder(0, slot, ArmyOrder{Kind: Occupy}); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tiles[p.Y][p.X].Owner != 0 || !s.Tribes[1].Defeated {
		t.Fatal("joint occupation failed")
	}
	v, _ := s.View(1)
	if len(v.Armies) != 0 {
		t.Fatal("defeated observer retained live troop surveillance")
	}
}
func FuzzMilitaryOrdersPreserveStateInvariants(f *testing.F) {
	f.Add([]byte{1, 2, 3, 4, 5, 0, 8, 19, 2, 0})
	f.Add([]byte{9, 9, 9, 9, 9, 9, 9, 9, 9})
	f.Fuzz(func(t *testing.T, data []byte) {
		s := warScenario(42)
		for _, value := range data[:min(len(data), 64)] {
			if s.AllExtinct() {
				break
			}
			id, slot := int(value)%TribeCount, int(value/3)%MaxArmies
			before := *s
			before.Events = append([]Event(nil), s.Events...)
			var err error
			switch value % 5 {
			case 0:
				err = s.EndMonth(Simulation)
			case 1:
				err = s.SetArmySize(id, slot, int(value%40))
			default:
				p := s.Tribes[id].Armies[slot].Position
				target := p.Neighbors()[int(value)%4]
				err = s.SetArmyOrder(id, slot, ArmyOrder{Kind: ArmyOrderKind(value % 6), Target: target})
			}
			if err != nil && !reflect.DeepEqual(before, *s) {
				t.Fatal("rejected military input partially mutated state")
			}
			if err := s.Validate(); err != nil {
				t.Fatal(err)
			}
		}
	})
}
