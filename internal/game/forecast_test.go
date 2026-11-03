package game

import (
	"reflect"
	"strings"
	"testing"
)

func TestForecastShowsFoodWoodAndGrowthTradeoff(t *testing.T) {
	for _, tt := range []struct {
		name                                                      string
		priority, stock, hunger, food, wood, net, after, shortage int
		canGrow                                                   bool
	}{
		{"automatic", 0, 30, 0, 12, 6, 4, 34, 0, true},
		{"more wood", 4, 30, 0, 6, 8, -2, 28, 0, true},
		{"growth stops before starvation", 4, 25, 0, 6, 8, -2, 23, 0, false},
		{"food shortage", 4, 1, 0, 6, 8, -2, 0, 1, false},
		{"full payment recovers from hunger", 4, 30, 1, 6, 8, -2, 28, 0, true},
		{"exact payment is not shortage", 4, 2, 1, 6, 8, -2, 0, 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s := productionScenario()
			s.Calendar.Elapsed = 3
			s.Tribes[0].Population, s.Tribes[0].Food, s.Tribes[0].HungerMonths = 31, tt.stock, tt.hunger
			if err := s.SetForestPriority(0, tt.priority); err != nil {
				t.Fatal(err)
			}
			v, err := s.View(0)
			if err != nil {
				t.Fatal(err)
			}
			f := ForecastFor(v)
			if f.Production.Food != tt.food || f.Production.Wood != tt.wood || f.Demand != 8 || f.NetFood != tt.net || f.FoodAfter != tt.after || f.Shortage != tt.shortage || (f.Growth.Reason == "") != tt.canGrow {
				t.Fatalf("wrong monthly preview: %+v", f)
			}
			if err := s.EndMonth(Interactive); err != nil {
				t.Fatal(err)
			}
			if s.Tribes[0].Food != f.FoodAfter || s.Tribes[0].Wood != 20+f.Production.Wood || (s.Tribes[0].GrowthProgress == 1) != tt.canGrow {
				t.Fatalf("preview differs from settlement: preview %+v, actual %+v", f, s.Tribes[0])
			}
		})
	}
}

func TestForecastIncludesCompletingFarmsWithoutMutation(t *testing.T) {
	s := growthScenario()
	s.Tiles[3][2].Farm = FarmReady - 1
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	before := v
	before.Cells = append([]Cell(nil), v.Cells...)
	before.Legal = append([]Action(nil), v.Legal...)
	stateBefore := *s
	f := ForecastFor(v)
	if f.Production.Food != 8 || f.NetFood != 1 || f.FoodAfter != 24 || f.Growth.Reason != "" {
		t.Fatalf("preview missed this month's completed farm: %+v", f)
	}
	if !reflect.DeepEqual(v, before) || !reflect.DeepEqual(*s, stateBefore) || !reflect.DeepEqual(ForecastFor(v), f) {
		t.Fatal("preview mutated state, view, construction or growth progress")
	}
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Food != f.FoodAfter || s.Tribes[0].GrowthProgress != 1 {
		t.Fatal("completion preview differed from settlement")
	}
	if s.rng.Int63() != growthScenario().rng.Int63() {
		t.Fatal("forecast or settlement consumed randomness")
	}
}

func TestForecastKeepsCurrentSeasonAndIncompleteFarms(t *testing.T) {
	for _, tt := range []struct{ elapsed, food int }{{2, 8}, {5, 12}, {8, 16}, {11, 0}} {
		s := productionScenario()
		s.Calendar.Elapsed = tt.elapsed
		v, err := s.View(0)
		if err != nil {
			t.Fatal(err)
		}
		f := ForecastFor(v)
		if f.Production.Food != tt.food {
			t.Fatalf("preview used next month's season: %+v", f)
		}
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Food != f.FoodAfter {
			t.Fatal("season boundary preview differed from settlement")
		}
	}
	s := productionScenario()
	s.Tiles[3][2].Farm = FarmPlanned
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	if f := ForecastFor(v); f.Production.Food != 4 || f.Production.Wood != 8 {
		t.Fatalf("preview finished farm too early: %+v", f)
	}
}

func TestForecastRespectsCapacityAndExtinction(t *testing.T) {
	s := productionScenario()
	s.Tribes[0].Population, s.Tribes[0].Food = 55, 100
	v, err := s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	if f := ForecastFor(v); !strings.Contains(f.Growth.Reason, "承载力") {
		t.Fatalf("preview ignored capacity: %+v", f)
	}
	s.Tribes[0].Population, s.Tribes[0].Soldiers = 0, 0
	s.clearExtinct(0)
	v, err = s.View(0)
	if err != nil {
		t.Fatal(err)
	}
	f := ForecastFor(v)
	if f.Production.Food != 0 || f.Production.Wood != 0 || f.Demand != 0 || f.Shortage != 0 || f.FoodAfter != 0 || !strings.Contains(f.Growth.Reason, "灭亡") {
		t.Fatalf("extinct tribe has an active forecast: %+v", f)
	}
}
