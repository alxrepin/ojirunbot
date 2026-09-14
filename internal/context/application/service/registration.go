package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

	"ojirun/internal/context/domain"
)

type userWriter interface {
	Upsert(ctx context.Context, telegramUserID int64, username, displayName string) (domain.User, error)
}

type profileWriter interface {
	Save(ctx context.Context, p domain.Profile) error
}

type RegistrantInfo struct {
	TelegramUserID int64
	Username       string
	FirstName      string
	LastName       string
}

type RegistrationOutcome struct {
	Prompt  string
	Invalid bool
	Done    bool
	Draft   domain.RegistrationDraft
	Targets domain.Targets
}

type registrationPayload struct {
	Sex           string  `json:"sex"`
	Age           int     `json:"age"`
	HeightCM      float64 `json:"height_cm"`
	WeightKG      float64 `json:"weight_kg"`
	ActivityLevel string  `json:"activity_level"`
	Goal          string  `json:"goal"`
}

type Registration struct {
	sessions sessionStore
	users    userWriter
	profiles profileWriter
	log      *slog.Logger
}

func NewRegistration(sessions sessionStore, users userWriter, profiles profileWriter, log *slog.Logger) *Registration {
	return &Registration{sessions: sessions, users: users, profiles: profiles, log: log}
}

func (r *Registration) Begin(ctx context.Context, telegramUserID, chatID int64) error {
	_, err := r.sessions.Start(ctx, domain.SessionRegistration, telegramUserID, chatID, "sex", map[string]any{})
	return err
}

func (r *Registration) ActiveSession(ctx context.Context, telegramUserID, chatID int64) (domain.Session, error) {
	return r.sessions.GetActive(ctx, domain.SessionRegistration, telegramUserID, chatID)
}

func (r *Registration) BindMessage(ctx context.Context, telegramUserID, chatID, botMessageID int64) error {
	return r.sessions.SetBotMessage(ctx, domain.SessionRegistration, telegramUserID, chatID, botMessageID)
}

func (r *Registration) SubmitText(ctx context.Context, session domain.Session, who RegistrantInfo, text string) (RegistrationOutcome, error) {
	payload := decodePayload(session)
	nextStep, ok := applyTextAnswer(session.Step, text, &payload)
	if !ok {
		return RegistrationOutcome{Prompt: session.Step, Invalid: true, Draft: draftOf(payload)}, nil
	}
	return r.commit(ctx, session, who, nextStep, payload)
}

func (r *Registration) SubmitChoice(ctx context.Context, session domain.Session, who RegistrantInfo, step, value string) (RegistrationOutcome, error) {
	if session.Step != step {
		return RegistrationOutcome{}, nil
	}
	payload := decodePayload(session)
	nextStep, ok := setRegistrationChoice(step, value, &payload)
	if !ok {
		return RegistrationOutcome{}, nil
	}
	return r.commit(ctx, session, who, nextStep, payload)
}

func (r *Registration) commit(ctx context.Context, session domain.Session, who RegistrantInfo, nextStep string, payload registrationPayload) (RegistrationOutcome, error) {
	if nextStep == "" {
		return r.complete(ctx, session, who, payload)
	}
	advanced, err := r.sessions.Advance(ctx, session.ID, session.Step, nextStep, payload)
	if err != nil {
		return RegistrationOutcome{}, err
	}
	if !advanced {
		return RegistrationOutcome{}, nil
	}
	return RegistrationOutcome{Prompt: nextStep, Draft: draftOf(payload)}, nil
}

func (r *Registration) complete(ctx context.Context, session domain.Session, who RegistrantInfo, payload registrationPayload) (RegistrationOutcome, error) {
	input, valid := payload.profileInput()
	if !valid {
		return RegistrationOutcome{Prompt: "sex", Invalid: true, Draft: draftOf(payload)}, nil
	}
	profile, err := domain.NewProfile("", input)
	if err != nil {
		return RegistrationOutcome{}, err
	}

	user, err := r.users.Upsert(ctx, who.TelegramUserID, who.Username, domain.DisplayNameFor(who.FirstName, who.LastName, who.Username))
	if err != nil {
		return RegistrationOutcome{}, err
	}
	profile.UserID = user.ID
	if err := r.profiles.Save(ctx, profile); err != nil {
		return RegistrationOutcome{}, err
	}
	if err := r.sessions.Complete(ctx, session.ID, user.ID); err != nil {
		r.log.Warn("complete registration session failed", "error", err)
	}

	return RegistrationOutcome{
		Done:  true,
		Draft: draftOf(payload),
		Targets: domain.Targets{
			CaloriesKCal: profile.DailyCaloriesKCal,
			ProteinG:     profile.DailyProteinG,
			FatG:         profile.DailyFatG,
			CarbsG:       profile.DailyCarbsG,
		},
	}, nil
}

func draftOf(p registrationPayload) domain.RegistrationDraft {
	return domain.RegistrationDraft{
		Sex:           p.Sex,
		Age:           p.Age,
		HeightCM:      p.HeightCM,
		WeightKG:      p.WeightKG,
		ActivityLevel: p.ActivityLevel,
		Goal:          p.Goal,
	}
}

func applyTextAnswer(step, text string, payload *registrationPayload) (string, bool) {
	switch step {
	case "sex":
		sex, ok := domain.ParseSex(text)
		if !ok {
			return "", false
		}
		payload.Sex = string(sex)
		return "age", true
	case "age":
		age, err := strconv.Atoi(strings.TrimSpace(text))
		if err != nil || age < 10 || age > 120 {
			return "", false
		}
		payload.Age = age
		return "height", true
	case "height":
		height, err := strconv.ParseFloat(normalizeDecimal(text), 64)
		if err != nil || height < 100 || height > 250 {
			return "", false
		}
		payload.HeightCM = height
		return "weight", true
	case "weight":
		weight, err := strconv.ParseFloat(normalizeDecimal(text), 64)
		if err != nil || weight < domain.MinWeightKG || weight > domain.MaxWeightKG {
			return "", false
		}
		payload.WeightKG = weight
		return "activity", true
	case "activity":
		activity, ok := domain.ParseActivity(text)
		if !ok {
			return "", false
		}
		payload.ActivityLevel = string(activity)
		return "goal", true
	case "goal":
		goal, ok := domain.ParseGoal(text)
		if !ok {
			return "", false
		}
		payload.Goal = string(goal)
		return "", true
	default:
		return "sex", true
	}
}

func setRegistrationChoice(step, value string, payload *registrationPayload) (string, bool) {
	switch step {
	case "sex":
		sex, ok := domain.ParseSex(value)
		if !ok {
			return "", false
		}
		payload.Sex = string(sex)
		return "age", true
	case "activity":
		activity, ok := domain.ParseActivity(value)
		if !ok {
			return "", false
		}
		payload.ActivityLevel = string(activity)
		return "goal", true
	case "goal":
		goal, ok := domain.ParseGoal(value)
		if !ok {
			return "", false
		}
		payload.Goal = string(goal)
		return "", true
	default:
		return "", false
	}
}

func decodePayload(session domain.Session) registrationPayload {
	var payload registrationPayload
	if len(session.PayloadJSON) > 0 {
		_ = json.Unmarshal(session.PayloadJSON, &payload)
	}
	return payload
}

func normalizeDecimal(text string) string {
	return strings.ReplaceAll(strings.TrimSpace(text), ",", ".")
}

func (p registrationPayload) profileInput() (domain.ProfileInput, bool) {
	input := domain.ProfileInput{
		Sex:           domain.Sex(p.Sex),
		Age:           p.Age,
		HeightCM:      p.HeightCM,
		WeightKG:      p.WeightKG,
		ActivityLevel: domain.ActivityLevel(p.ActivityLevel),
		Goal:          domain.Goal(p.Goal),
	}
	return input, input.Validate() == nil
}
