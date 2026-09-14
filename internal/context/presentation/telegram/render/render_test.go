package render

import (
	"strings"
	"testing"
	"time"

	"ojirun/internal/context/domain"
)

func TestMealResultUsesMarkdownTable(t *testing.T) {
	out := MealResult(domain.MealResultView{
		Analysis: domain.NutritionAnalysis{
			Summary: "Яйцо, творог и банан",
			Items: []domain.NutritionItem{
				{Name: "Яйцо | вареное", CaloriesKCal: 85, ProteinG: 7, FatG: 6, CarbsG: 0},
			},
			Total: domain.NutritionTotal{CaloriesKCal: 85, ProteinG: 7, FatG: 6, CarbsG: 0},
		},
		Profile: domain.Profile{
			DailyCaloriesKCal: 2100,
			DailyProteinG:     140,
			DailyFatG:         70,
			DailyCarbsG:       240,
		},
		ConsumedToday: 85,
	})

	if strings.Contains(out, "![](") {
		t.Fatalf("expected no rich markdown image block, got:\n%s", out)
	}
	if !strings.Contains(out, "| **Позиция** | **Белки** | **Углеводы** | **Жиры** | **Калории** |") {
		t.Fatalf("expected meal table header, got:\n%s", out)
	}
	if !strings.Contains(out, "| **Итого** | **7** | **0** | **6** | **85** |") {
		t.Fatalf("expected recognized totals only, got:\n%s", out)
	}
	if !strings.Contains(out, "📊 Сегодня: **85** / **2100** ккал\n⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜ 4%") {
		t.Fatalf("expected calorie progress bar, got:\n%s", out)
	}
	if !strings.Contains(out, "Яйцо \\| вареное") {
		t.Fatalf("expected escaped item pipe, got:\n%s", out)
	}
	if strings.Contains(out, "- Яйцо") {
		t.Fatalf("expected no bullet list, got:\n%s", out)
	}
}

func TestMealDraftUsesThinkingBlock(t *testing.T) {
	out := MealStageDraft(domain.StageAnalyzing)
	if !strings.Contains(out, "<tg-thinking>") {
		t.Fatalf("expected tg-thinking block, got:\n%s", out)
	}
	if !strings.Contains(out, "🧠") || !strings.Contains(out, "⚖️") {
		t.Fatalf("expected decorated draft status, got:\n%s", out)
	}
}

func TestCalorieProgressBarColors(t *testing.T) {
	cases := []struct {
		name     string
		consumed float64
		limit    float64
		want     string
	}{
		{"empty day", 0, 1000, "⬜⬜⬜⬜⬜⬜⬜⬜⬜⬜ 0%"},
		{"far below the norm is red", 700, 1000, "🟥🟥🟥🟥🟥🟥🟥⬜⬜⬜ 70%"},
		{"approaching the norm is orange", 800, 1000, "🟧🟧🟧🟧🟧🟧🟧🟧⬜⬜ 80%"},
		{"within ten percent below is green", 900, 1000, "🟩🟩🟩🟩🟩🟩🟩🟩🟩⬜ 90%"},
		{"hitting the norm is green", 1000, 1000, "🟩🟩🟩🟩🟩🟩🟩🟩🟩🟩 100%"},
		{"within ten percent above is green", 1100, 1000, "🟩🟩🟩🟩🟩🟩🟩🟩🟩🟩 110%"},
		{"slightly over is orange", 1200, 1000, "🟧🟧🟧🟧🟧🟧🟧🟧🟧🟧 120%"},
		{"far over the norm is red", 1500, 1000, "🟥🟥🟥🟥🟥🟥🟥🟥🟥🟥 150%"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := calorieProgressBar(tt.consumed, tt.limit); got != tt.want {
				t.Fatalf("calorieProgressBar() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRegistrationCardChecklist(t *testing.T) {
	draft := domain.RegistrationDraft{Sex: "male", Age: 30}
	card := RegistrationCard("height", draft, false)

	for _, want := range []string{"✅ Пол: <b>Мужской</b>", "✅ Возраст: <b>30 лет</b>", "👉 <b>Рост</b>", "▫️ Цель"} {
		if !strings.Contains(card, want) {
			t.Fatalf("card missing %q, got:\n%s", want, card)
		}
	}
	if strings.Contains(card, "⚠️") {
		t.Fatalf("valid step should not show the invalid warning:\n%s", card)
	}

	if invalid := RegistrationCard("age", domain.RegistrationDraft{Sex: "male"}, true); !strings.Contains(invalid, "⚠️") {
		t.Fatalf("invalid answer should show a warning, got:\n%s", invalid)
	}
}

func TestRegistrationDoneShowsTargets(t *testing.T) {
	out := RegistrationDone(
		domain.RegistrationDraft{Sex: "female", Age: 28, HeightCM: 165, WeightKG: 58.5, ActivityLevel: "moderate", Goal: "lose"},
		domain.Targets{CaloriesKCal: 1800, ProteinG: 120, FatG: 60, CarbsG: 180},
	)
	for _, want := range []string{"Профиль готов", "Женский", "58.5 кг", "Похудение", "<b>1800</b> ккал", "/add"} {
		if !strings.Contains(out, want) {
			t.Fatalf("done summary missing %q, got:\n%s", want, out)
		}
	}
}

func TestStatsCardShowsBarsForEachMacro(t *testing.T) {
	out := StatsCard(
		"Алекс",
		2,
		domain.DailyTotals{CaloriesKCal: 900, ProteinG: 60, FatG: 30, CarbsG: 90},
		domain.Profile{DailyCaloriesKCal: 1800, DailyProteinG: 120, DailyFatG: 60, DailyCarbsG: 180},
	)
	for _, want := range []string{"Статистика за сегодня", "Алекс", "приёмов пищи: <b>2</b>",
		"Калории", "Белки", "Жиры", "Углеводы", "900 / 1800 ккал", "50%"} {
		if !strings.Contains(out, want) {
			t.Fatalf("stats card missing %q, got:\n%s", want, out)
		}
	}
	if got := strings.Count(out, "🟥🟥🟥🟥🟥⬜⬜⬜⬜⬜"); got != 4 {
		t.Fatalf("expected 4 half-filled bars, got %d in:\n%s", got, out)
	}
}

func TestStatsCardEmptyDay(t *testing.T) {
	out := StatsCard("Кто-то", 0, domain.DailyTotals{}, domain.Profile{DailyCaloriesKCal: 2000})
	if !strings.Contains(out, "ещё нет подтверждённых записей") {
		t.Fatalf("expected empty-day hint, got:\n%s", out)
	}
}

func TestRegisterHintLinksToBot(t *testing.T) {
	withBot := NeedRegister("fat_ojirun_bot")
	if !strings.Contains(withBot, `<a href="https://t.me/fat_ojirun_bot?start=register">/start</a>`) {
		t.Fatalf("expected deep link to the bot, got:\n%s", withBot)
	}

	plain := NeedRegister("")
	if strings.Contains(plain, "<a ") || !strings.Contains(plain, "/start") {
		t.Fatalf("expected plain fallback, got:\n%s", plain)
	}
}

func TestReportMarkdownUsesTable(t *testing.T) {
	user := domain.ReportUser{
		User: domain.User{DisplayName: "Иван"},
		Profile: domain.Profile{
			DailyCaloriesKCal: 2100,
			DailyProteinG:     140,
			DailyFatG:         70,
			DailyCarbsG:       240,
		},
	}
	meals := []domain.DailyMeal{
		{
			CreatedAt:    time.Date(2026, 6, 28, 9, 12, 0, 0, time.UTC),
			Summary:      "Омлет | тост",
			CaloriesKCal: 430,
			ProteinG:     24,
			FatG:         21,
			CarbsG:       32,
		},
	}
	out := ReportMarkdown(user, time.Date(2026, 6, 28, 0, 0, 0, 0, time.UTC), meals, domain.DailyTotals{
		CaloriesKCal: 430,
		ProteinG:     24,
		FatG:         21,
		CarbsG:       32,
	}, nil)
	if !strings.Contains(out, "| Время | Еда | Ккал | Б | Ж | У |") {
		t.Fatalf("expected table header, got:\n%s", out)
	}
	if !strings.Contains(out, "Омлет \\| тост") {
		t.Fatalf("expected escaped pipe, got:\n%s", out)
	}
}

func TestSettingsCardShowsProgressAndCurrentValues(t *testing.T) {
	current := domain.ProfileSettings{Sex: "male", WeightKG: 80, CaloriesKCal: 2600, ProteinG: 128, FatG: 81, CarbsG: 350}
	draft := current
	draft.WeightKG = 78.5
	card := SettingsCard("calories", allSettingsFields, current, draft, false)
	for _, want := range []string{
		"✅ Пол: <b>Мужской</b>\n",
		"✅ Вес: <b>78.5 кг</b> <i>(было 80 кг)</i>",
		"👉 <b>Калории</b>: сейчас 2600 ккал",
		"▫️ Углеводы: 350 г",
		"дневную норму калорий",
	} {
		if !strings.Contains(card, want) {
			t.Fatalf("settings card missing %q, got:\n%s", want, card)
		}
	}
	if got := SettingsKeepLabel("calories", current); got != "Оставить текущее: 2600 ккал" {
		t.Fatalf("keep label = %q", got)
	}
}

var allSettingsFields = []string{"sex", "weight", "calories", "protein", "fat", "carbs"}

func TestSettingsMenuListsCurrentValues(t *testing.T) {
	current := domain.ProfileSettings{Sex: "female", WeightKG: 61.5, CaloriesKCal: 1900, ProteinG: 110, FatG: 60, CarbsG: 210}
	menu := SettingsMenu(current, false)
	for _, want := range []string{"• Пол: <b>Женский</b>", "• Вес: <b>61.5 кг</b>", "• Углеводы: <b>210 г</b>", "Что изменить?"} {
		if !strings.Contains(menu, want) {
			t.Fatalf("settings menu missing %q, got:\n%s", want, menu)
		}
	}
	if strings.Contains(menu, "⚠️") || !strings.Contains(SettingsMenu(current, true), "⚠️") {
		t.Fatal("only an invalid menu should carry the warning")
	}
}

func TestSettingsSingleFieldCardAndSummary(t *testing.T) {
	before := domain.ProfileSettings{Sex: "male", WeightKG: 80, CaloriesKCal: 2600, ProteinG: 128, FatG: 81, CarbsG: 350}
	card := SettingsCard("weight", []string{"weight"}, before, before, false)
	if !strings.Contains(card, "👉 <b>Вес</b>: сейчас 80 кг") || strings.Contains(card, "Калории") || strings.Contains(card, "по очереди") {
		t.Fatalf("single field card should show only weight, got:\n%s", card)
	}
	after := before
	after.WeightKG = 79
	saved := SettingsSaved([]string{"weight"}, before, after)
	if !strings.Contains(saved, "Вес: <b>79 кг</b> <i>(было 80 кг)</i>") || strings.Contains(saved, "Пол") {
		t.Fatalf("summary should list only the edited field, got:\n%s", saved)
	}
}

func TestSettingsSavedNotesManualTargets(t *testing.T) {
	before := domain.ProfileSettings{Sex: "male", WeightKG: 80, CaloriesKCal: 2600, ProteinG: 128, FatG: 81, CarbsG: 350}
	bodyOnly := before
	bodyOnly.WeightKG = 79
	if out := SettingsSaved(allSettingsFields, before, bodyOnly); strings.Contains(out, "вручную") {
		t.Fatalf("unchanged targets should not be flagged manual:\n%s", out)
	}
	coach := before
	coach.ProteinG = 160
	out := SettingsSaved(allSettingsFields, before, coach)
	if !strings.Contains(out, "Белки: <b>160 г</b> <i>(было 128 г)</i>") || !strings.Contains(out, "вручную") {
		t.Fatalf("expected changed protein and manual note, got:\n%s", out)
	}
}
