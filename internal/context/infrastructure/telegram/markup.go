package telegram

type SendOptions struct {
	ParseMode           string                      `json:"parse_mode,omitempty"`
	DisableNotification bool                        `json:"disable_notification,omitempty"`
	ReplyMarkup         any                         `json:"reply_markup,omitempty"`
	ReplyParameters     *ReplyParameters            `json:"reply_parameters,omitempty"`
	Ephemeral           *EphemeralMessageParameters `json:"ephemeral_message_parameters,omitempty"`
}

type ReplyParameters struct {
	MessageID          int64 `json:"message_id,omitempty"`
	ChatID             any   `json:"chat_id,omitempty"`
	EphemeralMessageID int64 `json:"ephemeral_message_id,omitempty"`
}

func ReplyTo(messageID int64) *SendOptions {
	if messageID == 0 {
		return nil
	}
	return &SendOptions{
		ReplyParameters: &ReplyParameters{MessageID: messageID},
	}
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
	Style        string `json:"style,omitempty"`
}

const (
	ButtonStyleSuccess = "success"
	ButtonStyleDanger  = "danger"
)

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type BotCommandScope struct {
	Type   string `json:"type"`
	ChatID any    `json:"chat_id,omitempty"`
}

func MealKeyboard(mealID string) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "Принять", CallbackData: EncodeMealCallback(ActionAccept, mealID), Style: ButtonStyleSuccess},
				{Text: "Редактировать", CallbackData: EncodeMealCallback(ActionEdit, mealID)},
			},
			{
				{Text: "Удалить", CallbackData: EncodeMealCallback(ActionDelete, mealID)},
			},
		},
	}
}

func MealCorrectionKeyboard(mealID string) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "Отменить", CallbackData: EncodeMealCallback(ActionCancel, mealID)},
			},
		},
	}
}

type regOption struct{ Label, Value string }

func RegistrationKeyboard(step string) (InlineKeyboardMarkup, bool) {
	perRow := 1
	var opts []regOption
	switch step {
	case "sex":
		perRow = 2
		opts = []regOption{{"Мужской", "male"}, {"Женский", "female"}}
	case "activity":
		opts = []regOption{
			{"Низкая", "low"}, {"Легкая", "light"}, {"Средняя", "moderate"},
			{"Высокая", "high"}, {"Спорт", "athlete"},
		}
	case "goal":
		opts = []regOption{{"Похудение", "lose"}, {"Поддержание", "maintain"}, {"Набор", "gain"}}
	default:
		return InlineKeyboardMarkup{}, false
	}

	var rows [][]InlineKeyboardButton
	var row []InlineKeyboardButton
	for _, o := range opts {
		row = append(row, InlineKeyboardButton{Text: o.Label, CallbackData: EncodeRegCallback(step, o.Value)})
		if len(row) == perRow {
			rows = append(rows, row)
			row = nil
		}
	}
	if len(row) > 0 {
		rows = append(rows, row)
	}
	return InlineKeyboardMarkup{InlineKeyboard: rows}, true
}

func MealInputKeyboard(sessionID string) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "Отменить", CallbackData: EncodeMealInputCancel(sessionID), Style: ButtonStyleDanger}},
		},
	}
}

func SettingsKeyboard(step, keepLabel string) InlineKeyboardMarkup {
	var rows [][]InlineKeyboardButton
	if step == "sex" {
		rows = append(rows, []InlineKeyboardButton{
			{Text: "Мужской", CallbackData: EncodeSettingsCallback(step, "male")},
			{Text: "Женский", CallbackData: EncodeSettingsCallback(step, "female")},
		})
	}
	rows = append(rows,
		[]InlineKeyboardButton{{Text: keepLabel, CallbackData: EncodeSettingsCallback(step, SettingsKeep), Style: ButtonStyleSuccess}},
		[]InlineKeyboardButton{{Text: "Отмена", CallbackData: EncodeSettingsCallback(step, SettingsCancel), Style: ButtonStyleDanger}},
	)
	return InlineKeyboardMarkup{InlineKeyboard: rows}
}

func SettingsMenuKeyboard() InlineKeyboardMarkup {
	choice := func(text, value, style string) InlineKeyboardButton {
		return InlineKeyboardButton{Text: text, CallbackData: EncodeSettingsCallback(SettingsMenuStep, value), Style: style}
	}
	return InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{
		{choice("Пол", "sex", ""), choice("Вес", "weight", "")},
		{choice("Калории", "calories", ""), choice("Белки", "protein", "")},
		{choice("Жиры", "fat", ""), choice("Углеводы", "carbs", "")},
		{choice("🎯 Вся норма КБЖУ", SettingsTargets, "")},
		{choice("Настроить всё", SettingsAll, ButtonStyleSuccess)},
		{choice("Отмена", SettingsCancel, ButtonStyleDanger)},
	}}
}

func ProfileKeyboard() InlineKeyboardMarkup {
	return InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{
		{{Text: "⚙️ Изменить данные", CallbackData: ProfileSettingsCallback}},
		{{Text: "♻️ Пересчитать заново", CallbackData: ProfileRestartCallback}},
	}}
}

func SubscribeKeyboard(channelURL string) InlineKeyboardMarkup {
	return InlineKeyboardMarkup{InlineKeyboard: [][]InlineKeyboardButton{
		{{Text: "📢 Подписаться на канал", URL: channelURL}},
		{{Text: "Я подписался", CallbackData: SubscriptionCheckCallback, Style: ButtonStyleSuccess}},
	}}
}

type ReplyKeyboardMarkup struct {
	Keyboard              [][]KeyboardButton `json:"keyboard"`
	IsPersistent          bool               `json:"is_persistent,omitempty"`
	ResizeKeyboard        bool               `json:"resize_keyboard,omitempty"`
	InputFieldPlaceholder string             `json:"input_field_placeholder,omitempty"`
}

type KeyboardButton struct {
	Text  string `json:"text"`
	Style string `json:"style,omitempty"`
}
