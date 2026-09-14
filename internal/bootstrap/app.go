package bootstrap

import (
	"context"
	"sync"
	"time"

	"ojirun/internal/context/application/service"
	tgclient "ojirun/internal/context/infrastructure/telegram"
	presentation "ojirun/internal/context/presentation/telegram"
)

type App struct {
	reg    *registry
	router *presentation.Router

	updateSem chan struct{}
	wg        sync.WaitGroup
}

func New(ctx context.Context) (*App, error) {
	cfg, err := Load()
	if err != nil {
		return nil, err
	}
	log := newLogger(cfg.LogLevel)

	prompts, err := LoadPrompts("prompts")
	if err != nil {
		return nil, err
	}

	reg := new(registry)
	if err := reg.load(ctx, cfg, log, prompts); err != nil {
		return nil, err
	}
	if err := reg.store.Migrate(ctx, "migrations"); err != nil {
		_ = reg.cleanUp()
		return nil, err
	}

	app := &App{
		reg:       reg,
		updateSem: make(chan struct{}, cfg.MaxConcurrentUpdates),
	}
	app.router = presentation.NewRouter(presentation.Deps{
		API:          reg.tg,
		Messenger:    reg.messenger,
		AddMeal:      reg.addMeal,
		Actions:      reg.mealActions,
		Profiles:     reg.getProfile,
		Stats:        reg.getStats,
		Registration: reg.registration,
		Settings:     reg.settings,
		MealInput:    reg.mealInput,
		MealJobs:     reg.mealJobs,
		Pipeline:     reg.pipeline,
		Subscription: reg.subscription,
		Channel:      presentation.Channel{Label: cfg.Telegram.ChannelLabel(), URL: cfg.Telegram.ChannelURL},
		BotUsername:  reg.botUsername,
		Loc:          reg.loc,
		Log:          log,
	})

	log.Info("bot initialized",
		"required_channel", cfg.Telegram.RequiredChannel,
		"meal_workers", cfg.MaxConcurrentAI,
		"report_workers", cfg.ReportWorkers,
		"max_meals_per_day", cfg.MaxMealsPerDay,
		"log_level", cfg.LogLevel,
	)
	return app, nil
}

func (a *App) Run(ctx context.Context) error {
	a.configureBotCommands(ctx)

	for _, workers := range []*service.JobWorkers{a.reg.mealWorkers, a.reg.reportWorkers} {
		a.wg.Add(1)
		go func(workers *service.JobWorkers) {
			defer a.wg.Done()
			workers.Run(ctx)
		}(workers)
	}

	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		a.runDailyReports(ctx)
	}()

	a.reg.log.Info("telegram polling started")

	var offset int64
	for ctx.Err() == nil {
		updates, err := a.reg.tg.GetUpdates(ctx, offset, 30)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			a.reg.log.Error("telegram getUpdates failed", "error", err)
			if sleepCtx(ctx, 2*time.Second) != nil {
				break
			}
			continue
		}
		for _, update := range updates {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
			a.dispatch(ctx, update)
		}
	}

	a.wg.Wait()
	return ctx.Err()
}

func (a *App) Stop() {
	_ = a.reg.cleanUp()
}

func (a *App) dispatch(ctx context.Context, update tgclient.Update) {
	select {
	case a.updateSem <- struct{}{}:
	case <-ctx.Done():
		return
	}
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		defer func() { <-a.updateSem }()
		a.router.HandleUpdate(ctx, update)
	}()
}

func (a *App) runDailyReports(ctx context.Context) {
	hour, minute, _ := ParseClock(a.reg.cfg.DailyReportTime)
	now := time.Now().In(a.reg.loc)
	today := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, a.reg.loc)
	if now.After(today) {
		a.scheduleReports(ctx, today.AddDate(0, 0, -1))
	}

	for {
		now := time.Now().In(a.reg.loc)
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, a.reg.loc)
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			a.scheduleReports(ctx, next.AddDate(0, 0, -1))
		}
	}
}

func (a *App) scheduleReports(ctx context.Context, reportDate time.Time) {
	queued, err := a.reg.report.Schedule(ctx, reportDate)
	if err != nil {
		a.reg.log.Error("schedule daily reports failed", "error", err, "report_date", reportDate.Format("2006-01-02"))
		return
	}
	a.reg.log.Info("daily reports scheduled", "report_date", reportDate.Format("2006-01-02"), "queued", queued)
	a.reg.reportWorkers.Wake()
}

func (a *App) configureBotCommands(ctx context.Context) {
	privateCommands := []tgclient.BotCommand{
		{Command: "add", Description: "добавить приём пищи"},
		{Command: "stats", Description: "статистика за сегодня"},
		{Command: "profile", Description: "профиль и дневная норма"},
		{Command: "settings", Description: "изменить вес, пол и норму КБЖУ"},
		{Command: "yesterday", Description: "перенести запись на вчера (ответом на сообщение)"},
		{Command: "readd", Description: "повторить распознавание (ответом на сообщение)"},
		{Command: "help", Description: "как это работает"},
		{Command: "start", Description: "регистрация и расчёт нормы"},
	}
	groupCommands := []tgclient.BotCommand{
		{Command: "add", Description: "добавить приём пищи"},
		{Command: "stats", Description: "статистика за сегодня (можно /stats @имя)"},
		{Command: "help", Description: "как это работает"},
		{Command: "yesterday", Description: "перенести запись на вчера (ответом на сообщение)"},
		{Command: "readd", Description: "повторить распознавание (ответом на сообщение)"},
	}
	if err := a.reg.tg.SetMyCommands(ctx, privateCommands, &tgclient.BotCommandScope{Type: "all_private_chats"}); err != nil {
		a.reg.log.Warn("set private bot commands failed", "error", err)
	}
	if err := a.reg.tg.SetMyCommands(ctx, groupCommands, &tgclient.BotCommandScope{Type: "all_group_chats"}); err != nil {
		a.reg.log.Warn("set group bot commands failed", "error", err)
	}
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
