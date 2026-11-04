package game

import "testing"

func sizedGrowthState(t *testing.T, population, capacity, food, progress int) *State {
	t.Helper()
	s := scenarioState()
	remaining := (capacity - CapitalCapacity) / CapacityPerPlot
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			if remaining > 0 && s.Tiles[y][x].Owner == Unowned {
				ownTile(s, 0, Point{x, y}, Grassland, FarmNone)
				remaining--
			}
		}
	}
	if remaining != 0 {
		t.Fatal("growth fixture exhausted map")
	}
	s.Tribes[0].Population, s.Tribes[0].Food, s.Tribes[0].GrowthProgress = population, food, progress
	if err := s.Validate(); err != nil {
		t.Fatal(err)
	}
	return s
}

func TestNonlinearGrowthHandCalculatedMonthlyRates(t *testing.T) {
	for _, tt := range []struct{ population, capacity, progress, wantPopulation, wantProgress int }{
		{30, 35, 0, 30, 375}, {30, 40, 0, 30, 450}, {100, 120, 0, 101, 500},
		{200, 240, 0, 203, 0}, {100, 105, 0, 100, 375}, {100, 105, 200, 100, 575},
		{30, 30, 900, 30, 0}, {40, 35, 900, 40, 0}, {34, 35, 950, 35, 0},
	} {
		s := sizedGrowthState(t, tt.population, tt.capacity, 100000, tt.progress)
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		got := s.Tribes[0]
		if got.Population != tt.wantPopulation || got.GrowthProgress != tt.wantProgress {
			t.Fatalf("P/K=%d/%d: population=%d progress=%d, want %d/%d", tt.population, tt.capacity, got.Population, got.GrowthProgress, tt.wantPopulation, tt.wantProgress)
		}
		if got.Food != 100000-FoodDemand(tt.population, 5) || got.Soldiers != 5 {
			t.Fatal("births consumed food retroactively or changed soldiers")
		}
		if err := s.Validate(); err != nil {
			t.Fatal(err)
		}
	}
}

func TestLowReservePausesFractionButActualShortageClearsIt(t *testing.T) {
	s := sizedGrowthState(t, 30, 35, 30, 800) // Pay 7, reserve 23: no hunger, below safe 24.
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].GrowthProgress != 800 || s.Tribes[0].HungerMonths != 0 {
		t.Fatal("low reserve erased fractional growth")
	}
	s.Tribes[0].Food = 31
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Population != 31 || s.Tribes[0].GrowthProgress != 175 {
		t.Fatal("reserve recovery failed to resume fractional growth")
	}
	s.Tribes[0].Food = 0
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].GrowthProgress != 0 || s.Tribes[0].HungerMonths != 1 {
		t.Fatal("actual shortage retained progress")
	}
}

func TestMultipleBirthsRespectPostGrowthFoodReserve(t *testing.T) {
	s := sizedGrowthState(t, 104, 150, 88, 800) // Pay 22, 66 remains: supports 105, not 106.
	if err := s.EndMonth(Interactive); err != nil {
		t.Fatal(err)
	}
	if s.Tribes[0].Population != 105 || s.Tribes[0].Food != 66 || s.Tribes[0].GrowthProgress != 360 {
		t.Fatalf("batch bypassed reserve or banked blocked people: %+v", s.Tribes[0])
	}
}

func TestAnnualReserveDipDoesNotTrapLastVacantSlot(t *testing.T) {
	s := sizedGrowthState(t, 30, 35, 100, 0)
	fullAt := 0
	for month := 0; month < 120; month++ {
		p := s.Tribes[0].Population
		reserve := FoodDemand(p+1, 5) * 3
		s.Tribes[0].Food = FoodDemand(p, 5) + reserve
		if month%12 == 11 {
			s.Tribes[0].Food--
		}
		if err := s.EndMonth(Interactive); err != nil {
			t.Fatal(err)
		}
		if s.Tribes[0].Population == 35 && fullAt == 0 {
			fullAt = month + 1
		}
		if s.Tribes[0].HungerMonths != 0 {
			t.Fatal("controlled reserve dip caused real hunger")
		}
	}
	if fullAt != 32 || s.Tribes[0].GrowthProgress != 0 {
		t.Fatalf("last vacant slot stalled or banked progress: fullAt=%d tribe=%+v", fullAt, s.Tribes[0])
	}
}
