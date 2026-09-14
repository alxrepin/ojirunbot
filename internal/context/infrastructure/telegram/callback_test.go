package telegram

import "testing"

func TestMealCallbackRoundTrip(t *testing.T) {
	data := EncodeMealCallback(ActionAccept, "meal-123")
	if data != "meal:accept:meal-123" {
		t.Fatalf("unexpected encoding: %q", data)
	}
	cb, ok := DecodeMealCallback(data)
	if !ok {
		t.Fatal("decode failed for valid data")
	}
	if cb.Action != ActionAccept || cb.MealID != "meal-123" {
		t.Fatalf("round trip mismatch: %+v", cb)
	}
}

func TestDecodeMealCallbackRejectsBadData(t *testing.T) {
	bad := []string{
		"",
		"meal",
		"meal:accept",
		"other:accept:id",
		"meal::id",
		"meal:accept:",
		"meal:accept:id:extra",
	}
	for _, data := range bad {
		if _, ok := DecodeMealCallback(data); ok {
			t.Errorf("expected %q to be rejected", data)
		}
	}
}

func TestKeyboardUsesEncodedCallbacks(t *testing.T) {
	kb := MealKeyboard("m1")
	got := kb.InlineKeyboard[0][0].CallbackData
	if cb, ok := DecodeMealCallback(got); !ok || cb.Action != ActionAccept || cb.MealID != "m1" {
		t.Fatalf("accept button callback not decodable: %q", got)
	}
}

func TestRegCallbackRoundTrip(t *testing.T) {
	data := EncodeRegCallback("sex", "male")
	if data != "reg:sex:male" {
		t.Fatalf("unexpected encoding: %q", data)
	}
	cb, ok := DecodeRegCallback(data)
	if !ok || cb.Step != "sex" || cb.Value != "male" {
		t.Fatalf("round trip mismatch: %+v ok=%v", cb, ok)
	}
	if _, ok := DecodeRegCallback("meal:accept:1"); ok {
		t.Error("meal callback wrongly decoded as reg")
	}
	if _, ok := DecodeMealCallback("reg:sex:male"); ok {
		t.Error("reg callback wrongly decoded as meal")
	}
}

func TestRegistrationKeyboardButtonsDecode(t *testing.T) {
	for _, step := range []string{"sex", "activity", "goal"} {
		kb, ok := RegistrationKeyboard(step)
		if !ok {
			t.Fatalf("%s should have a keyboard", step)
		}
		for _, row := range kb.InlineKeyboard {
			for _, btn := range row {
				cb, ok := DecodeRegCallback(btn.CallbackData)
				if !ok || cb.Step != step || cb.Value == "" {
					t.Errorf("%s button %q not decodable: %+v", step, btn.CallbackData, cb)
				}
			}
		}
	}
	if _, ok := RegistrationKeyboard("age"); ok {
		t.Error("age is a free-text step and must not have a keyboard")
	}
}
