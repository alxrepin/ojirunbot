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
