package telegram

import "strings"

const (
	callbackPrefix = "meal"

	ActionAccept = "accept"
	ActionEdit   = "edit"
	ActionCancel = "cancel"
	ActionDelete = "delete"
)

type MealCallback struct {
	Action string
	MealID string
}

func EncodeMealCallback(action, mealID string) string {
	return callbackPrefix + ":" + action + ":" + mealID
}

func DecodeMealCallback(data string) (MealCallback, bool) {
	action, mealID, ok := splitCallback(data, callbackPrefix)
	return MealCallback{Action: action, MealID: mealID}, ok
}

const regPrefix = "reg"

type RegCallback struct {
	Step  string
	Value string
}

func EncodeRegCallback(step, value string) string {
	return regPrefix + ":" + step + ":" + value
}

func DecodeRegCallback(data string) (RegCallback, bool) {
	step, value, ok := splitCallback(data, regPrefix)
	return RegCallback{Step: step, Value: value}, ok
}

const (
	settingsPrefix = "set"

	SettingsKeep     = "keep"
	SettingsCancel   = "cancel"
	SettingsMenuStep = "menu"
	SettingsAll      = "all"
	SettingsTargets  = "targets"
)

type SettingsCallback struct {
	Step  string
	Value string
}

func EncodeSettingsCallback(step, value string) string {
	return settingsPrefix + ":" + step + ":" + value
}

func DecodeSettingsCallback(data string) (SettingsCallback, bool) {
	step, value, ok := splitCallback(data, settingsPrefix)
	return SettingsCallback{Step: step, Value: value}, ok
}

const mealInputCancelPrefix = "add:cancel:"

func EncodeMealInputCancel(sessionID string) string {
	return mealInputCancelPrefix + sessionID
}

func DecodeMealInputCancel(data string) (string, bool) {
	sessionID, ok := strings.CutPrefix(data, mealInputCancelPrefix)
	return sessionID, ok && sessionID != "" && !strings.Contains(sessionID, ":")
}

func splitCallback(data, prefix string) (string, string, bool) {
	parts := strings.Split(data, ":")
	if len(parts) != 3 || parts[0] != prefix || parts[1] == "" || parts[2] == "" {
		return "", "", false
	}
	return parts[1], parts[2], true
}

const SubscriptionCheckCallback = "sub:check"

const (
	ProfileSettingsCallback = "profile:settings"
	ProfileRestartCallback  = "profile:restart"
)
