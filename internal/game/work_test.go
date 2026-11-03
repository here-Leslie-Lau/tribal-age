package game

import (
	"reflect"
	"strings"
	"testing"
)

func TestForestPriorityChangesProductionAndPersists(t *testing.T) {
	s := productionScenario()
	s.Calendar.Elapsed = 3 // June: one farm produces six food.
	s.Tribes[0].Population = 31
	if err := s.SetForestPriority(0, 4); err != nil {
		t.Fatal(err)
	}
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	p := ProductionFor(v, Summer)
	if p.Food != 6 || p.Wood != 8 || len(p.Farms) != 1 || len(p.Forests) != 4 {
		t.Fatalf("priority should trade one farm for one forest: %+v", p)
	}
	if !reflect.DeepEqual(p.Farms, []Point{{2, 3}}) || !reflect.DeepEqual(p.Unstaffed, []Point{{3, 3}}) {
		t.Fatalf("coordinate order changed: %+v", p)
	}
	for month := 0; month < 2; month++ {
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].ForestPriority != 4 || s.Tribes[0].Food != 60-(month+1)*2 || s.Tribes[0].Wood != 20+(month+1)*8 {
			t.Fatalf("monthly settlement ignored persistent priority: %+v", s.Tribes[0])
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.SetForestPriority(0, 0); err != nil {
		t.Fatal(err)
	}
	v, err = s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	p = ProductionFor(v, Summer)
	if p.Food != 12 || p.Wood != 6 {
		t.Fatalf("automatic allocation was not restored: %+v", p)
	}
}

func TestForestPriorityUsesOnlyAvailablePlotsAndWorkers(t *testing.T) {
	for _, tt := range []struct {
		name                           string
		priority, population, soldiers int
		season                         Season
		food, wood                     int
	}{
		{"minimum not maximum", 1, 30, 5, Spring, 8, 6},
		{"missing forests release workers", 6, 30, 5, Spring, 4, 8},
		{"largest future target", Width * Height, 30, 5, Spring, 4, 8},
		{"all workers reserved", 4, 30, 10, Spring, 0, 8},
		{"no civilian workers", 4, 5, 5, Spring, 0, 0},
		{"winter still uses every forest", 1, 30, 5, Winter, 0, 8},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := productionScenario()
			s.Tribes[0].Population, s.Tribes[0].Soldiers = tt.population, tt.soldiers
			if err := s.SetForestPriority(0, tt.priority); err != nil {
				t.Fatal(err)
			}
			v, err := s.View(0)
			if err != nil {
				t.Fatal(err)
			}
			before := v
			before.Cells = append([]Cell(nil), v.Cells...)
			p := ProductionFor(v, tt.season)
			if p.Food != tt.food || p.Wood != tt.wood || len(p.Farms)+len(p.Forests) > v.Self.WorkSlots() {
				t.Fatalf("wrong constrained production: %+v", p)
			}
			if !reflect.DeepEqual(v, before) {
				t.Fatal("allocation mutated the view")
			}
		})
	}
}

func TestForestPriorityIsFreeIdempotentAndPrivate(t *testing.T) {
	s := productionScenario()
	s.Tribes[0].AP = 0
	before := *s
	if err := s.SetForestPriority(0, 4); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].ForestPriority != 4 {
		t.Fatal("setting was not saved")
	}
	changed := *s
	changed.Tribes[0].ForestPriority = before.Tribes[0].ForestPriority
	changed.Events = before.Events
	if !reflect.DeepEqual(changed, before) {
		t.Fatal("setting consumed resources, time or AP")
	}
	if !strings.Contains(strings.Join(s.VisibleEvents(0, 0), "\n"), "4 格森林") || len(s.VisibleEvents(1, 0)) != 0 {
		t.Fatal("setting confirmation was missing or leaked")
	}
	before = *s
	before.Events = append([]Event(nil), s.Events...)
	if err := s.SetForestPriority(0, 4); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*s, before) {
		t.Fatal("repeating the setting was not idempotent")
	}
	if s.rng.Int63() != productionScenario().rng.Int63() {
		t.Fatal("setting consumed randomness")
	}
}

func TestInvalidForestPriorityIsAtomic(t *testing.T) {
	for _, tt := range []struct{ id, priority int }{{-1, 2}, {TribeCount, 2}, {0, -1}, {0, Width*Height + 1}} {
		s := productionScenario()
		before := *s
		if err := s.SetForestPriority(tt.id, tt.priority); err == nil {
			t.Fatal("invalid setting accepted")
		}
		if !reflect.DeepEqual(*s, before) {
			t.Fatal("rejected setting changed state")
		}
	}
	for _, priority := range []int{-1, Width*Height + 1} {
		s := productionScenario()
		s.Tribes[0].ForestPriority = priority
		if err := s.Validate(); err == nil {
			t.Fatal("invalid saved setting accepted")
		}
	}
}

func TestForestPriorityClearsOnExtinction(t *testing.T) {
	s := productionScenario()
	s.Tribes[0].Population, s.Tribes[0].Soldiers, s.Tribes[0].Food = 1, 1, 0
	if err := s.SetForestPriority(0, 4); err != nil {
		t.Fatal(err)
	}
	for month := 0; month < 2; month++ {
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
	}
	if s.Tribes[0].Alive() || s.Tribes[0].ForestPriority != 0 {
		t.Fatal("extinct tribe retained its work setting")
	}
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	before := *s
	before.Events = append([]Event(nil), s.Events...)
	if err := s.SetForestPriority(0, 0); err == nil || !reflect.DeepEqual(before, *s) {
		t.Fatal("dead tribe accepted a setting")
	}
	s.Tribes[0].ForestPriority = 1
	if err := s.Validate(); err == nil {
		t.Fatal("extinct tribe's saved setting was not checked")
	}
}

func TestForestPriorityIsIndependentForEveryTribe(t *testing.T) {
	s := productionScenario()
	before, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetForestPriority(1, 4); err != nil {
		t.Fatal(err)
	}
	after, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	other, err := s.View(1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) || other.Self.ForestPriority != 4 || len(s.VisibleEvents(0, 0)) != 0 || len(s.VisibleEvents(1, 0)) != 1 {
		t.Fatal("one tribe's work policy affected or leaked to another tribe")
	}
}

func TestForestPriorityForecastAcrossLongReplay(t *testing.T) {
	for _, seed := range []int64{0, 42} {
		stepped, automatic := New(seed), New(seed)
		for month := 0; month < 120; month++ {
			for _, s := range []*State{stepped, automatic} {
				if s.Tribes[0].Alive() {
					if err := s.SetForestPriority(0, []int{0, 4, 1, 5}[(month/4)%4]); err != nil {
						t.Fatal(err)
					}
				}
			}
			// A manual player uses the same legal choices as automatic tribe zero.
			for action := 0; action < MonthlyAP; action++ {
				v, err := stepped.View(0)
				if err != nil {
					t.Fatal(err)
				}
				a, ok := ChooseAction(v)
				if !ok {
					break
				}
				if err := stepped.Apply(0, a); err != nil {
					t.Fatal(err)
				}
			}
			v, err := stepped.View(0)
			if err != nil {
				t.Fatal(err)
			}
			f := ForecastFor(v)
			for _, s := range []*State{stepped, automatic} {
				if err := s.EndMonth(Simulation); err != nil {
					t.Fatal(err)
				}
				if err := s.Validate(); err != nil {
					t.Fatal(err)
				}
			}
			if !reflect.DeepEqual(stepped, automatic) || stepped.Tribes[0].Food != f.FoodAfter {
				t.Fatalf("seed %d month %d: replay or food forecast diverged", seed, month)
			}
			if stepped.Tribes[0].Alive() && stepped.Tribes[0].Wood != v.Self.Wood+f.Production.Wood {
				t.Fatalf("seed %d month %d: wood forecast diverged", seed, month)
			}
		}
		if stepped.Calendar.Elapsed != 120 {
			t.Fatal("custom-policy replay stopped early")
		}
	}
}
