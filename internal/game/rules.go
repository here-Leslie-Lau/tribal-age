package game

// Numeric game rules live here; the engine, AI and CLI share them.
const (
	MonthsPerYear     = 12
	MonthsPerSeason   = 3
	StartingMonth     = 3
	Width             = 24
	Height            = 16
	TribeCount        = 3
	MinSpawnDistance  = 6
	InitialPopulation = 30
	InitialSoldiers   = 5
	InitialFood       = 60
	InitialWood       = 20
	MonthlyAP         = 2
	ActionAP          = 1
	ClaimWood         = 5
	FarmWood          = 10
	FarmMonths        = 3
	TrainingFood      = 5
	TrainingSize      = 5
	ForestYield       = 2
	RationGroup       = 5
	Unowned           = -1
)

func FarmYield(season Season) int {
	switch season {
	case Spring:
		return 4
	case Summer:
		return 6
	case Autumn:
		return 8
	default:
		return 0
	}
}

func FoodDemand(population, soldiers int) int {
	return roundUpRations(population) + roundUpRations(soldiers)
}

func roundUpRations(n int) int {
	result := n / RationGroup
	if n%RationGroup != 0 {
		result++
	}
	return result
}
