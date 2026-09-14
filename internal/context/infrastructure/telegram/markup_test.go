package telegram

import "testing"

func TestMealCorrectionKeyboard(t *testing.T) {
	keyboard := MealCorrectionKeyboard("meal-1")
	if len(keyboard.InlineKeyboard) != 1 || len(keyboard.InlineKeyboard[0]) != 1 {
		t.Fatalf("expected one cancel button, got %#v", keyboard.InlineKeyboard)
	}
	button := keyboard.InlineKeyboard[0][0]
	if button.Text != "Отменить" {
		t.Fatalf("button text = %q, want %q", button.Text, "Отменить")
	}
	if button.CallbackData != "meal:cancel:meal-1" {
		t.Fatalf("callback data = %q, want %q", button.CallbackData, "meal:cancel:meal-1")
	}
}
