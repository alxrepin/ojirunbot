package telegram

import "testing"

func TestSettingsKeyboard(t *testing.T) {
	keyboard := SettingsKeyboard("sex", "Оставить текущее: Мужской")
	rows := keyboard.InlineKeyboard
	if len(rows) != 3 || len(rows[0]) != 2 {
		t.Fatalf("sex step wants choices, keep and cancel rows, got %#v", rows)
	}
	keep, cancel := rows[1][0], rows[2][0]
	if keep.Style != ButtonStyleSuccess || keep.Text != "Оставить текущее: Мужской" {
		t.Fatalf("keep button = %#v, want green with the current value", keep)
	}
	if cancel.Style != ButtonStyleDanger {
		t.Fatalf("cancel button = %#v, want red", cancel)
	}
	if got, ok := DecodeSettingsCallback(keep.CallbackData); !ok || got != (SettingsCallback{Step: "sex", Value: SettingsKeep}) {
		t.Fatalf("keep callback decoded to %#v, %v", got, ok)
	}

	if rows := SettingsKeyboard("weight", "Оставить текущее: 80 кг").InlineKeyboard; len(rows) != 2 {
		t.Fatalf("typed step wants only keep and cancel rows, got %#v", rows)
	}
}

func TestSettingsMenuKeyboard(t *testing.T) {
	rows := SettingsMenuKeyboard().InlineKeyboard
	var values []string
	for _, row := range rows {
		for _, button := range row {
			got, ok := DecodeSettingsCallback(button.CallbackData)
			if !ok || got.Step != SettingsMenuStep {
				t.Fatalf("menu button %#v decoded to %#v, %v", button, got, ok)
			}
			values = append(values, got.Value)
		}
	}
	want := []string{"sex", "weight", "calories", "protein", "fat", "carbs", SettingsTargets, SettingsAll, SettingsCancel}
	if len(values) != len(want) {
		t.Fatalf("menu values = %v, want %v", values, want)
	}
	for i := range want {
		if values[i] != want[i] {
			t.Fatalf("menu values = %v, want %v", values, want)
		}
	}
	all, cancel := rows[len(rows)-2][0], rows[len(rows)-1][0]
	if all.Style != ButtonStyleSuccess || cancel.Style != ButtonStyleDanger {
		t.Fatalf("configure-all should be green and cancel red, got %#v %#v", all, cancel)
	}
}

func TestProfileKeyboard(t *testing.T) {
	rows := ProfileKeyboard().InlineKeyboard
	if len(rows) != 2 || rows[0][0].CallbackData != ProfileSettingsCallback || rows[1][0].CallbackData != ProfileRestartCallback {
		t.Fatalf("profile keyboard = %#v", rows)
	}
}

func TestMealInputKeyboard(t *testing.T) {
	button := MealInputKeyboard("5f0c").InlineKeyboard[0][0]
	if button.Style != ButtonStyleDanger {
		t.Fatalf("cancel button = %#v, want red", button)
	}
	if id, ok := DecodeMealInputCancel(button.CallbackData); !ok || id != "5f0c" {
		t.Fatalf("decoded %q, %v; want 5f0c", id, ok)
	}
	for _, data := range []string{"add:cancel:", "meal:cancel:5f0c", "add:cancel:a:b"} {
		if _, ok := DecodeMealInputCancel(data); ok {
			t.Errorf("DecodeMealInputCancel(%q) should fail", data)
		}
	}
}
