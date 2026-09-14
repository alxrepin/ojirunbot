package telegram

import (
	"context"

	tg "ojirun/internal/context/infrastructure/telegram"
	"ojirun/internal/context/presentation/telegram/render"
)

type menuButton struct {
	label   string
	command string
	style   string
}

var mainMenu = [][]menuButton{
	{{label: "🍽 Добавить еду", command: "/add", style: tg.ButtonStyleSuccess}, {label: "📊 Статистика", command: "/stats"}},
	{{label: "👤 Профиль", command: "/profile"}, {label: "⚙️ Настройки", command: "/settings"}},
	{{label: "💡 Как это работает?", command: "/help"}},
}

func menuCommand(text string) (string, bool) {
	for _, row := range mainMenu {
		for _, button := range row {
			if button.label == text {
				return button.command, true
			}
		}
	}
	return "", false
}

func menuKeyboard() tg.ReplyKeyboardMarkup {
	rows := make([][]tg.KeyboardButton, 0, len(mainMenu))
	for _, row := range mainMenu {
		buttons := make([]tg.KeyboardButton, 0, len(row))
		for _, button := range row {
			buttons = append(buttons, tg.KeyboardButton{Text: button.label, Style: button.style})
		}
		rows = append(rows, buttons)
	}
	return tg.ReplyKeyboardMarkup{
		Keyboard:              rows,
		IsPersistent:          true,
		ResizeKeyboard:        true,
		InputFieldPlaceholder: "Выберите действие или пришлите фото еды",
	}
}

func withMenu(opts *tg.SendOptions) *tg.SendOptions {
	copied := copyOptions(opts)
	copied.ReplyMarkup = menuKeyboard()
	return copied
}

func (r *Router) sendMenu(ctx context.Context, chatID int64, text string) {
	_, _ = r.api.SendMessage(ctx, chatID, text, withMenu(nil))
}

func (r *Router) showHelp(ctx context.Context, msg tg.Message) {
	opts := htmlOptions()
	if msg.Chat.Type == "private" {
		opts = withMenu(opts)
	}
	_, _ = r.notify(ctx, msg, render.HowItWorks(), opts)
}
