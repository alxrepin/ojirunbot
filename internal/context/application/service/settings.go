package service

import (
	"context"
	"encoding/json"
	"math"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"ojirun/internal/context/domain"
)

const (
	SettingsMenuStep      = "menu"
	SettingsChoiceAll     = "all"
	SettingsChoiceTargets = "targets"
)

var (
	settingsSteps       = []string{"sex", "weight", "calories", "protein", "fat", "carbs"}
	settingsTargetSteps = []string{"calories", "protein", "fat", "carbs"}
)

type settingsUsers interface {
	GetByTelegramID(ctx context.Context, telegramUserID int64) (domain.User, error)
}

type settingsProfiles interface {
	GetByUserID(ctx context.Context, userID string) (domain.Profile, error)
	Save(ctx context.Context, p domain.Profile) error
}

type SettingsOutcome struct {
	Step      string
	Fields    []string
	Invalid   bool
	Saved     bool
	Cancelled bool
	Current   domain.ProfileSettings
	Draft     domain.ProfileSettings
}

type settingsPayload struct {
	Current domain.ProfileSettings `json:"current"`
	Draft   domain.ProfileSettings `json:"draft"`
	Fields  []string               `json:"fields,omitempty"`
}

func (p settingsPayload) fields() []string {
	if len(p.Fields) == 0 {
		return settingsSteps
	}
	return p.Fields
}

type Settings struct {
	sessions sessionStore
	users    settingsUsers
	profiles settingsProfiles
}

func NewSettings(sessions sessionStore, users settingsUsers, profiles settingsProfiles) *Settings {
	return &Settings{sessions: sessions, users: users, profiles: profiles}
}

func (s *Settings) Begin(ctx context.Context, telegramUserID, chatID int64) (SettingsOutcome, error) {
	user, err := s.users.GetByTelegramID(ctx, telegramUserID)
	if err != nil {
		return SettingsOutcome{}, err
	}
	profile, err := s.profiles.GetByUserID(ctx, user.ID)
	if err != nil {
		return SettingsOutcome{}, err
	}
	current := profile.Settings()
	payload := settingsPayload{Current: current, Draft: current}
	if _, err := s.sessions.Start(ctx, domain.SessionSettings, telegramUserID, chatID, SettingsMenuStep, payload); err != nil {
		return SettingsOutcome{}, err
	}
	return SettingsOutcome{Step: SettingsMenuStep, Current: current, Draft: current}, nil
}

func (s *Settings) ActiveSession(ctx context.Context, telegramUserID, chatID int64) (domain.Session, error) {
	return s.sessions.GetActive(ctx, domain.SessionSettings, telegramUserID, chatID)
}

func (s *Settings) BindMessage(ctx context.Context, telegramUserID, chatID, botMessageID int64) error {
	return s.sessions.SetBotMessage(ctx, domain.SessionSettings, telegramUserID, chatID, botMessageID)
}

func (s *Settings) Select(ctx context.Context, session domain.Session, choice string) (SettingsOutcome, error) {
	fields, ok := settingsChoiceFields(choice)
	if session.Step != SettingsMenuStep || !ok {
		return SettingsOutcome{}, nil
	}
	payload := decodeSettingsPayload(session)
	payload.Fields = fields
	return s.moveTo(ctx, session, payload, fields[0])
}

func (s *Settings) SubmitText(ctx context.Context, session domain.Session, text string) (SettingsOutcome, error) {
	payload := decodeSettingsPayload(session)
	if session.Step == SettingsMenuStep || !applySettingsText(session.Step, text, &payload.Draft) {
		return SettingsOutcome{Step: session.Step, Fields: payload.fields(), Invalid: true, Current: payload.Current, Draft: payload.Draft}, nil
	}
	return s.advance(ctx, session, payload)
}

func (s *Settings) Choose(ctx context.Context, session domain.Session, step, value string) (SettingsOutcome, error) {
	if session.Step != step {
		return SettingsOutcome{}, nil
	}
	payload := decodeSettingsPayload(session)
	sex, ok := domain.ParseSex(value)
	if step != "sex" || !ok {
		return SettingsOutcome{}, nil
	}
	payload.Draft.Sex = string(sex)
	return s.advance(ctx, session, payload)
}

func (s *Settings) Keep(ctx context.Context, session domain.Session, step string) (SettingsOutcome, error) {
	if session.Step != step || step == SettingsMenuStep {
		return SettingsOutcome{}, nil
	}
	return s.advance(ctx, session, decodeSettingsPayload(session))
}

func (s *Settings) Cancel(ctx context.Context, session domain.Session) (SettingsOutcome, error) {
	payload := decodeSettingsPayload(session)
	cancelled, err := s.sessions.Cancel(ctx, session.ID, session.TelegramUserID)
	if err != nil || !cancelled {
		return SettingsOutcome{}, err
	}
	return SettingsOutcome{Cancelled: true, Current: payload.Current, Draft: payload.Current}, nil
}

func (s *Settings) advance(ctx context.Context, session domain.Session, payload settingsPayload) (SettingsOutcome, error) {
	next := nextSettingsStep(payload.fields(), session.Step)
	if next == "" {
		return s.save(ctx, session, payload)
	}
	return s.moveTo(ctx, session, payload, next)
}

func (s *Settings) moveTo(ctx context.Context, session domain.Session, payload settingsPayload, next string) (SettingsOutcome, error) {
	advanced, err := s.sessions.Advance(ctx, session.ID, session.Step, next, payload)
	if err != nil || !advanced {
		return SettingsOutcome{}, err
	}
	return SettingsOutcome{Step: next, Fields: payload.fields(), Current: payload.Current, Draft: payload.Draft}, nil
}

func (s *Settings) save(ctx context.Context, session domain.Session, payload settingsPayload) (SettingsOutcome, error) {
	closed, err := s.sessions.Close(ctx, session.ID, session.TelegramUserID)
	if err != nil || !closed {
		return SettingsOutcome{}, err
	}
	user, err := s.users.GetByTelegramID(ctx, session.TelegramUserID)
	if err != nil {
		return SettingsOutcome{}, err
	}
	profile, err := s.profiles.GetByUserID(ctx, user.ID)
	if err != nil {
		return SettingsOutcome{}, err
	}
	updated, err := profile.ApplySettings(payload.Draft)
	if err != nil {
		return SettingsOutcome{}, err
	}
	if err := s.profiles.Save(ctx, updated); err != nil {
		return SettingsOutcome{}, err
	}
	return SettingsOutcome{Saved: true, Fields: payload.fields(), Current: payload.Current, Draft: updated.Settings()}, nil
}

func settingsChoiceFields(choice string) ([]string, bool) {
	switch choice {
	case SettingsChoiceAll:
		return settingsSteps, true
	case SettingsChoiceTargets:
		return settingsTargetSteps, true
	}
	if slices.Contains(settingsSteps, choice) {
		return []string{choice}, true
	}
	return nil, false
}

func nextSettingsStep(fields []string, step string) string {
	for i, candidate := range fields {
		if candidate == step && i+1 < len(fields) {
			return fields[i+1]
		}
	}
	return ""
}

func applySettingsText(step, text string, draft *domain.ProfileSettings) bool {
	if step == "sex" {
		sex, ok := domain.ParseSex(text)
		if ok {
			draft.Sex = string(sex)
		}
		return ok
	}
	value, ok := parseAmount(text)
	if !ok {
		return false
	}
	switch step {
	case "weight":
		if value < domain.MinWeightKG || value > domain.MaxWeightKG {
			return false
		}
		draft.WeightKG = value
		return true
	case "calories":
		kcal := int(math.Round(value))
		if kcal < domain.MinDailyCaloriesKCal || kcal > domain.MaxDailyCaloriesKCal {
			return false
		}
		draft.CaloriesKCal = kcal
		return true
	case "protein":
		return setMacroGrams(&draft.ProteinG, value)
	case "fat":
		return setMacroGrams(&draft.FatG, value)
	case "carbs":
		return setMacroGrams(&draft.CarbsG, value)
	default:
		return false
	}
}

func setMacroGrams(field *int, value float64) bool {
	grams := int(math.Round(value))
	if grams < 0 || grams > domain.MaxDailyMacroG {
		return false
	}
	*field = grams
	return true
}

func parseAmount(text string) (float64, bool) {
	compact := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, text)
	compact = strings.TrimRightFunc(compact, func(r rune) bool { return unicode.IsLetter(r) || r == '.' })
	value, err := strconv.ParseFloat(strings.ReplaceAll(compact, ",", "."), 64)
	if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
		return 0, false
	}
	return value, true
}

func decodeSettingsPayload(session domain.Session) settingsPayload {
	var payload settingsPayload
	if len(session.PayloadJSON) > 0 {
		_ = json.Unmarshal(session.PayloadJSON, &payload)
	}
	return payload
}
