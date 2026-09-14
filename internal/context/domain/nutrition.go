package domain

import (
	"fmt"
	"math"
	"strings"
)

const FormulaVersion = "mifflin-st-jeor-v1"

type Sex string

const (
	SexMale   Sex = "male"
	SexFemale Sex = "female"
)

type ActivityLevel string

const (
	ActivityLow      ActivityLevel = "low"
	ActivityLight    ActivityLevel = "light"
	ActivityModerate ActivityLevel = "moderate"
	ActivityHigh     ActivityLevel = "high"
	ActivityAthlete  ActivityLevel = "athlete"
)

type Goal string

const (
	GoalLose     Goal = "lose"
	GoalMaintain Goal = "maintain"
	GoalGain     Goal = "gain"
)

type ProfileInput struct {
	Sex           Sex
	Age           int
	HeightCM      float64
	WeightKG      float64
	ActivityLevel ActivityLevel
	Goal          Goal
}

type Targets struct {
	CaloriesKCal int
	ProteinG     int
	FatG         int
	CarbsG       int
}

func CalculateTargets(input ProfileInput) (Targets, error) {
	if err := input.Validate(); err != nil {
		return Targets{}, err
	}

	bmr := 10*input.WeightKG + 6.25*input.HeightCM - 5*float64(input.Age)
	switch input.Sex {
	case SexMale:
		bmr += 5
	case SexFemale:
		bmr -= 161
	}

	calories := bmr * activityFactor(input.ActivityLevel)
	switch input.Goal {
	case GoalLose:
		calories *= 0.85
	case GoalGain:
		calories *= 1.10
	}

	proteinPerKG := 1.6
	if input.Goal == GoalLose || input.Goal == GoalGain {
		proteinPerKG = 1.8
	}
	protein := input.WeightKG * proteinPerKG
	fatCalories := calories * 0.28
	fat := fatCalories / 9
	carbs := (calories - protein*4 - fat*9) / 4
	if carbs < 0 {
		carbs = 0
	}

	return Targets{
		CaloriesKCal: roundToInt(calories),
		ProteinG:     roundToInt(protein),
		FatG:         roundToInt(fat),
		CarbsG:       roundToInt(carbs),
	}, nil
}

func (i ProfileInput) Validate() error {
	if i.Sex != SexMale && i.Sex != SexFemale {
		return fmt.Errorf("unsupported sex %q", i.Sex)
	}
	if i.Age < 10 || i.Age > 120 {
		return fmt.Errorf("age out of range")
	}
	if i.HeightCM < 100 || i.HeightCM > 250 {
		return fmt.Errorf("height out of range")
	}
	if i.WeightKG < MinWeightKG || i.WeightKG > MaxWeightKG {
		return fmt.Errorf("weight out of range")
	}
	switch i.ActivityLevel {
	case ActivityLow, ActivityLight, ActivityModerate, ActivityHigh, ActivityAthlete:
	default:
		return fmt.Errorf("unsupported activity level %q", i.ActivityLevel)
	}
	switch i.Goal {
	case GoalLose, GoalMaintain, GoalGain:
	default:
		return fmt.Errorf("unsupported goal %q", i.Goal)
	}
	return nil
}

func ParseSex(value string) (Sex, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "м", "муж", "мужской", "male":
		return SexMale, true
	case "ж", "жен", "женский", "female":
		return SexFemale, true
	default:
		return "", false
	}
}

func ParseActivity(value string) (ActivityLevel, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "низкая", "low":
		return ActivityLow, true
	case "2", "легкая", "light":
		return ActivityLight, true
	case "3", "средняя", "moderate":
		return ActivityModerate, true
	case "4", "высокая", "high":
		return ActivityHigh, true
	case "5", "спорт", "athlete":
		return ActivityAthlete, true
	default:
		return "", false
	}
}

func ParseGoal(value string) (Goal, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "похудение", "похудеть", "lose":
		return GoalLose, true
	case "2", "поддержание", "держать", "maintain":
		return GoalMaintain, true
	case "3", "набор", "набрать", "gain":
		return GoalGain, true
	default:
		return "", false
	}
}

func activityFactor(level ActivityLevel) float64 {
	switch level {
	case ActivityLow:
		return 1.2
	case ActivityLight:
		return 1.375
	case ActivityModerate:
		return 1.55
	case ActivityHigh:
		return 1.725
	case ActivityAthlete:
		return 1.9
	default:
		return 1.2
	}
}

func roundToInt(v float64) int {
	return int(math.Round(v))
}
