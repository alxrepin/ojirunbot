package service

import "testing"

func TestSetRegistrationChoice(t *testing.T) {
	cases := []struct {
		step, value string
		wantNext    string
		wantOK      bool
		check       func(registrationPayload) bool
	}{
		{"sex", "male", "age", true, func(p registrationPayload) bool { return p.Sex == "male" }},
		{"activity", "moderate", "goal", true, func(p registrationPayload) bool { return p.ActivityLevel == "moderate" }},
		{"goal", "lose", "", true, func(p registrationPayload) bool { return p.Goal == "lose" }},
		{"sex", "bogus", "", false, nil},
		{"age", "30", "", false, nil},
	}
	for _, tt := range cases {
		var p registrationPayload
		next, ok := setRegistrationChoice(tt.step, tt.value, &p)
		if ok != tt.wantOK || next != tt.wantNext {
			t.Errorf("setRegistrationChoice(%q,%q) = (%q,%v), want (%q,%v)", tt.step, tt.value, next, ok, tt.wantNext, tt.wantOK)
		}
		if tt.check != nil && !tt.check(p) {
			t.Errorf("setRegistrationChoice(%q,%q) did not set payload: %+v", tt.step, tt.value, p)
		}
	}
}
