package domain

type Profile struct {
	UserID            string
	Sex               string
	Age               int
	HeightCM          float64
	WeightKG          float64
	ActivityLevel     string
	Goal              string
	DailyCaloriesKCal int
	DailyProteinG     int
	DailyFatG         int
	DailyCarbsG       int
	FormulaVersion    string
}

func NewProfile(userID string, input ProfileInput) (Profile, error) {
	targets, err := CalculateTargets(input)
	if err != nil {
		return Profile{}, err
	}
	return Profile{
		UserID:            userID,
		Sex:               string(input.Sex),
		Age:               input.Age,
		HeightCM:          input.HeightCM,
		WeightKG:          input.WeightKG,
		ActivityLevel:     string(input.ActivityLevel),
		Goal:              string(input.Goal),
		DailyCaloriesKCal: targets.CaloriesKCal,
		DailyProteinG:     targets.ProteinG,
		DailyFatG:         targets.FatG,
		DailyCarbsG:       targets.CarbsG,
		FormulaVersion:    FormulaVersion,
	}, nil
}
