package bootstrap

import "testing"

func TestNormalizeChannel(t *testing.T) {
	cases := map[string]string{
		"":                     "",
		"ojirun":               "@ojirun",
		"@ojirun":              "@ojirun",
		"https://t.me/ojirun/": "@ojirun",
		"-1001234567890":       "-1001234567890",
	}
	for in, want := range cases {
		if got := normalizeChannel(in); got != want {
			t.Errorf("normalizeChannel(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestChannelTarget(t *testing.T) {
	if target := (TelegramConfig{}).ChannelTarget(); target != nil {
		t.Fatalf("no channel should disable the requirement, got %v", target)
	}
	if target := (TelegramConfig{RequiredChannel: "-100123"}).ChannelTarget(); target != int64(-100123) {
		t.Fatalf("numeric channel should be an id, got %#v", target)
	}
	named := TelegramConfig{RequiredChannel: "@ojirun", ChannelURL: "https://t.me/ojirun"}
	if named.ChannelTarget() != "@ojirun" || named.ChannelLabel() != "@ojirun" {
		t.Fatalf("named channel: target %v label %q", named.ChannelTarget(), named.ChannelLabel())
	}
}
