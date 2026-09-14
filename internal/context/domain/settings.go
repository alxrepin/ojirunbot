package domain

import "fmt"

const (
	MinWeightKG = 30
	MaxWeightKG = 300
)

const (
	MinDailyCaloriesKCal = 800
	MaxDailyCaloriesKCal = 10000
	MaxDailyMacroG       = 1000
)

const ManualFormulaVersion = "manual"

type ProfileSettings struct {
	Sex          string
	WeightKG     float64
	CaloriesKCal int
	ProteinG     int
	FatG         int
	CarbsG       int
}

func (s ProfileSettings) Validate() error {
	if Sex(s.Sex) != SexMale && Sex(s.Sex) != SexFemale {
		return fmt.Errorf("unsupported sex %q", s.Sex)
	}
	if s.WeightKG < MinWeightKG || s.WeightKG > MaxWeightKG {
		return fmt.Errorf("weight out of range")
	}
	if s.CaloriesKCal < MinDailyCaloriesKCal || s.CaloriesKCal > MaxDailyCaloriesKCal {
		return fmt.Errorf("daily calories out of range")
	}
	for _, grams := range []int{s.ProteinG, s.FatG, s.CarbsG} {
		if grams < 0 || grams > MaxDailyMacroG {
			return fmt.Errorf("daily macro out of range")
		}
	}
	return nil
}

func (s ProfileSettings) TargetsDiffer(other ProfileSettings) bool {
	return s.CaloriesKCal != other.CaloriesKCal ||
		s.ProteinG != other.ProteinG ||
		s.FatG != other.FatG ||
		s.CarbsG != other.CarbsG
}

func (p Profile) Settings() ProfileSettings {
	return ProfileSettings{
		Sex:          p.Sex,
		WeightKG:     p.WeightKG,
		CaloriesKCal: p.DailyCaloriesKCal,
		ProteinG:     p.DailyProteinG,
		FatG:         p.DailyFatG,
		CarbsG:       p.DailyCarbsG,
	}
}

func (p Profile) ApplySettings(s ProfileSettings) (Profile, error) {
	if err := s.Validate(); err != nil {
		return Profile{}, err
	}
	if s.TargetsDiffer(p.Settings()) {
		p.FormulaVersion = ManualFormulaVersion
	}
	p.Sex = s.Sex
	p.WeightKG = s.WeightKG
	p.DailyCaloriesKCal = s.CaloriesKCal
	p.DailyProteinG = s.ProteinG
	p.DailyFatG = s.FatG
	p.DailyCarbsG = s.CarbsG
	return p, nil
}
