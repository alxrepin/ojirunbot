package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"ojirun/internal/context/application/service"
	"ojirun/internal/context/application/usecase"
	"ojirun/internal/context/domain"
	"ojirun/internal/context/infrastructure/db/postgres"
	jobrepo "ojirun/internal/context/infrastructure/db/postgres/job"
	mealrepo "ojirun/internal/context/infrastructure/db/postgres/meal"
	photorepo "ojirun/internal/context/infrastructure/db/postgres/photo"
	profilerepo "ojirun/internal/context/infrastructure/db/postgres/profile"
	reportrepo "ojirun/internal/context/infrastructure/db/postgres/report"
	sessionrepo "ojirun/internal/context/infrastructure/db/postgres/session"
	userrepo "ojirun/internal/context/infrastructure/db/postgres/user"
	"ojirun/internal/context/infrastructure/files"
	"ojirun/internal/context/infrastructure/openrouter"
	tgclient "ojirun/internal/context/infrastructure/telegram"
	presentation "ojirun/internal/context/presentation/telegram"
)

const (
	mealJobLease   = 5 * time.Minute
	reportJobLease = 5 * time.Minute
)

type registry struct {
	cfg     Config
	log     *slog.Logger
	loc     *time.Location
	prompts Prompts

	store *postgres.Client
	tg    *tgclient.Client

	botUsername string

	subscription *service.Subscription
	registration *service.Registration
	settings     *service.Settings
	mealInput    *service.MealInput
	messenger    *presentation.MealMessenger
	pipeline     *service.MealPipeline
	mealJobs     *service.MealJobs
	report       *service.Report

	mealWorkers   *service.JobWorkers
	reportWorkers *service.JobWorkers

	addMeal     *usecase.AddMeal
	mealActions *usecase.MealActions
	getProfile  *usecase.GetProfile
	getStats    *usecase.GetStats
}

func (r *registry) load(ctx context.Context, cfg Config, log *slog.Logger, prompts Prompts) error {
	r.cfg = cfg
	r.log = log
	r.prompts = prompts

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return fmt.Errorf("load timezone %q: %w", cfg.Timezone, err)
	}
	r.loc = loc

	store, err := postgres.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("connect postgres: %w", err)
	}
	r.store = store
	tg, err := tgclient.NewClient(cfg.Telegram.APIBaseURL, cfg.Telegram.BotToken, cfg.ProxyURL)
	if err != nil {
		return fmt.Errorf("build telegram client: %w", err)
	}
	r.tg = tg

	var botUserID int64
	if me, err := r.tg.GetMe(ctx); err != nil {
		log.Warn("resolve bot identity via getMe failed", "error", err)
	} else {
		r.botUsername = me.Username
		botUserID = me.ID
		log.Info("bot identity resolved", "bot_username", me.Username)
	}

	users := userrepo.NewRepository(store)
	profiles := profilerepo.NewRepository(store)
	sessions := sessionrepo.NewRepository(store)
	meals := mealrepo.NewRepository(store)
	photos := photorepo.NewRepository(store)
	reports := reportrepo.NewRepository(store)
	jobs := jobrepo.NewRepository(store)

	archiver := files.NewArchiver(files.NewPhotoStorage(cfg.PhotoStorageDir, cfg.MaxPhotoBytes), r.tg)
	aiCfg := cfg.OpenRouterClientConfig()
	aiCfg.Logger = log.With("component", "openrouter")
	analyzer, err := openrouter.NewClient(aiCfg)
	if err != nil {
		return fmt.Errorf("build openrouter client: %w", err)
	}

	memberStatus := service.ChatMemberStatusFunc(func(ctx context.Context, chat any, telegramUserID int64) (string, error) {
		member, err := r.tg.GetChatMember(ctx, chat, telegramUserID)
		if err == nil && member.Status == "restricted" && !member.IsMember {
			return "left", nil
		}
		return member.Status, err
	})
	r.subscription = service.NewSubscription(memberStatus, cfg.Telegram.ChannelTarget(), log)
	switch {
	case !r.subscription.Enabled():
		log.Warn("TELEGRAM_REQUIRED_CHANNEL is empty: the bot is open to everyone")
	case botUserID != 0:
		if err := r.subscription.Verify(ctx, botUserID); err != nil {
			log.Error("cannot check channel subscriptions: add the bot to the channel as an administrator",
				"error", err, "channel", cfg.Telegram.RequiredChannel)
		}
	}

	r.registration = service.NewRegistration(sessions, users, profiles, log)
	r.settings = service.NewSettings(sessions, users, profiles)
	r.mealInput = service.NewMealInput(sessions)
	r.messenger = presentation.NewMealMessenger(r.tg, meals, log)
	r.pipeline = service.NewMealPipeline(
		analyzer, archiver, meals, photos, profiles, reports, r.messenger,
		prompts.NutritionAnalysis, prompts.NutritionCorrection, loc, log,
	)
	r.mealJobs = service.NewMealJobs(jobs, meals, profiles, r.pipeline)
	r.mealWorkers = service.NewJobWorkers(jobs, domain.QueueMeal, cfg.MaxConcurrentAI, mealJobLease, r.mealJobs.Handle, log)
	r.mealJobs.OnEnqueue(r.mealWorkers.Wake)

	r.report = service.NewReport(
		reports, analyzer, presentation.NewReportNotifier(r.tg, log), r.subscription,
		prompts.DailyRecommendation, loc, log,
	)
	r.reportWorkers = service.NewJobWorkers(jobs, domain.QueueReport, cfg.ReportWorkers, reportJobLease, r.report.Handle, log)

	r.addMeal = usecase.NewAddMeal(users, profiles, meals, domain.MealLimits{
		PerUserPerDay: cfg.MaxMealsPerDay,
		PerChatPerDay: cfg.MaxMealsPerChatPerDay,
		InFlight:      cfg.MaxMealsInFlight,
	}, loc)
	r.mealActions = usecase.NewMealActions(meals)
	r.getProfile = usecase.NewGetProfile(users, profiles)
	r.getStats = usecase.NewGetStats(users, profiles, reports, loc)

	return nil
}

func (r *registry) cleanUp() error {
	if r.store != nil {
		r.store.Close()
	}
	return nil
}
