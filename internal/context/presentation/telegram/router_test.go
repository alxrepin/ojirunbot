package telegram

import "testing"

func TestSplitCommand(t *testing.T) {
	cases := []struct {
		in      string
		command string
		args    string
	}{
		{"/add гречка с курицей", "/add", "гречка с курицей"},
		{"/add@my_bot салат", "/add", "салат"},
		{"/Profile", "/profile", ""},
		{"просто текст", "", "просто текст"},
		{"  /register  ", "/register", ""},
	}
	for _, tt := range cases {
		command, args := splitCommand(tt.in)
		if command != tt.command || args != tt.args {
			t.Errorf("splitCommand(%q) = (%q, %q), want (%q, %q)", tt.in, command, args, tt.command, tt.args)
		}
	}
}

func TestParseCommandIgnoresOtherBots(t *testing.T) {
	r := &Router{botUsername: "Ojirun_Bot"}
	cases := []struct {
		in      string
		command string
		foreign bool
	}{
		{"/add", "/add", false},
		{"/add@ojirun_bot салат", "/add", false},
		{"/add@other_bot салат", "", true},
		{"гречка", "", false},
	}
	for _, tt := range cases {
		command, _, foreign := r.parseCommand(tt.in)
		if command != tt.command || foreign != tt.foreign {
			t.Errorf("parseCommand(%q) = (%q, foreign %v), want (%q, %v)", tt.in, command, foreign, tt.command, tt.foreign)
		}
	}
}
