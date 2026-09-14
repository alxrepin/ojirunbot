package render

import (
	"fmt"
	"html"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"ojirun/internal/context/domain"
)

func MealStageStatus(stage domain.MealStage) string {
	switch stage {
	case domain.StageReceived:
		return "🧾 Принял запись. Готовлю разбор…"
	case domain.StageDownloadingPhoto:
		return "📸 Скачиваю фото и сохраняю его для анализа…"
	case domain.StageAnalyzing:
		return "🧠 Анализирую питание и считаю КБЖУ…"
	case domain.StageReanalyzing:
		return "✏️ Пересчитываю с учетом корректировки…"
	default:
		return "Обрабатываю…"
	}
}

func MealStageDraft(stage domain.MealStage) string {
	switch stage {
	case domain.StageReceived:
		return mealThinkingDraft("🧾 Принял запись", "✨ Готовлю разбор…", "Шаг 1/3 • собираю описание и вложения")
	case domain.StageDownloadingPhoto:
		return mealThinkingDraft("📸 Забираю фото", "⬇️ Сохраняю изображение…", "Шаг 2/3 • проверяю файл и метаданные")
	case domain.StageAnalyzing:
		return mealThinkingDraft("🧠 Считаю КБЖУ", "⚖️ Сопоставляю порции и ингредиенты…", "Шаг 3/3 • скоро покажу таблицу")
	case domain.StageReanalyzing:
		return mealThinkingDraft("✏️ Уточняю результат", "🔄 Пересчитываю с корректировкой…", "Сохраню новую ревизию и обновлю кнопки")
	default:
		return mealThinkingDraft("⏳ Обрабатываю", "…", "")
	}
}

func MealFailureText(failure domain.MealFailure) string {
	switch failure {
	case domain.FailDownloadPhoto:
		return "Не удалось скачать фото."
	case domain.FailSavePhoto:
		return "Не удалось сохранить фото."
	case domain.FailAnalyze:
		return "Не удалось проанализировать питание."
	case domain.FailPersistResult:
		return "Не удалось сохранить результат."
	case domain.FailReanalyze:
		return "Не удалось пересчитать питание."
	case domain.FailPersistCorrection:
		return "Не удалось сохранить корректировку."
	default:
		return "Не удалось обработать запись."
	}
}

func mealThinkingDraft(title, thinking, detail string) string {
	return fmt.Sprintf(
		"<b>%s</b>\n<tg-thinking>%s</tg-thinking>\n<i>%s</i>",
		html.EscapeString(title),
		html.EscapeString(thinking),
		html.EscapeString(detail),
	)
}

func NeedPrivateMessage() string {
	return "Напишите мне в ЛС."
}

func HowItWorks() string {
	return "💡 <b>Как это работает</b>\n\n" +
		"1️⃣ <b>Добавьте еду</b> — кнопка «🍽 Добавить еду» или /add, затем опишите блюдо и порцию. Можно просто прислать фото: подпись станет описанием.\n\n" +
		"2️⃣ <b>Проверьте разбор</b> — я посчитаю КБЖУ. Нажмите «Принять», «Редактировать» (и напишите, что поправить) или «Удалить».\n\n" +
		"3️⃣ <b>Следите за нормой</b> — «📊 Статистика» показывает, сколько съедено за сегодня. Зелёный бар значит, что вы попали в норму.\n\n" +
		"4️⃣ <b>Утренний отчёт</b> — если вчера были записи, утром пришлю разбор дня с рекомендациями.\n\n" +
		"👥 <b>В группах</b> тоже работаю: добавьте меня в чат. Подсказки и черновики видите только вы, а группа — итог принятой еды и статистику.\n\n" +
		"⚙️ Норму КБЖУ, вес и пол можно поменять в «Настройках» — например, если норму назначил тренер."
}

func MenuReady() string {
	return "👇 Меню всегда под рукой — кнопки внизу экрана."
}

func NeedRegister(botUsername string) string {
	return registerHint("👋 Сначала зарегистрируйтесь в ЛС —", botUsername)
}

func FinishRegister(botUsername string) string {
	return registerHint("✏️ Сначала завершите регистрацию в ЛС —", botUsername)
}

func registerHint(prefix, botUsername string) string {
	if botUsername == "" {
		return prefix + " /start"
	}
	return prefix + ` <a href="https://t.me/` + botUsername + `?start=register">/start</a>`
}

func PrivateHelp() string {
	return "Команды:\n• /add — добавить приём пищи (или просто пришлите фото)\n• /stats — статистика за сегодня\n• /profile — профиль и дневная норма\n• /settings — изменить вес, пол и норму КБЖУ\n• /help — как это работает\n• /start — пройти регистрацию заново"
}

func MealNeedContent() string {
	return "Не вижу, что добавить. Пришлите описание еды или фото блюда."
}

func MealDailyLimit(limit int) string {
	return fmt.Sprintf("🚫 Лимит на сегодня исчерпан: %d из %d записей. Новые записи можно будет добавить завтра.", limit, limit)
}

func ReaddNeedReply() string {
	return "Ответьте командой /readd на исходное сообщение с едой, чтобы повторить распознавание."
}

func YesterdayNeedReply() string {
	return "Ответьте командой /yesterday на сообщение с едой или на карточку бота, чтобы перенести запись на вчера."
}

func YesterdayNoEntry() string {
	return "Не нашёл запись для этого сообщения. Ответьте на сообщение с едой или на карточку бота."
}

func YesterdayNotAuthor() string {
	return "Перенести дату записи может только её автор."
}

func YesterdayNotMovable() string {
	return "Эту запись нельзя перенести: она удалена или ещё не распознана."
}

func YesterdayMoved(date time.Time) string {
	return "Готово. Запись перенесена на " + date.Format("2006-01-02") + "."
}

func MealNothingRecognized() string {
	return "Не смог распознать еду. Уточните описание (что и сколько) или пришлите фото поразборчивее."
}

var registrationSteps = []struct {
	key, label string
}{
	{"sex", "Пол"},
	{"age", "Возраст"},
	{"height", "Рост"},
	{"weight", "Вес"},
	{"activity", "Активность"},
	{"goal", "Цель"},
}

func RegistrationError() string {
	return "⚠️ Не удалось обработать регистрацию. Попробуйте позже — отправьте /start."
}

func RegistrationCard(step string, d domain.RegistrationDraft, invalid bool) string {
	var b strings.Builder
	b.WriteString("🥗 <b>Регистрация профиля</b>\n")
	b.WriteString("<i>Заполним данные — посчитаю вашу дневную норму КБЖУ.</i>\n\n")
	b.WriteString(registrationChecklist(step, d))
	b.WriteString("\n")
	if invalid {
		b.WriteString("⚠️ <i>Не понял ответ. Попробуйте ещё раз.</i>\n\n")
	}
	b.WriteString(registrationPrompt(step))
	return b.String()
}

func registrationChecklist(step string, d domain.RegistrationDraft) string {
	var b strings.Builder
	for _, f := range registrationSteps {
		value := registrationFieldValue(f.key, d)
		switch {
		case value != "":
			fmt.Fprintf(&b, "✅ %s: <b>%s</b>\n", f.label, value)
		case f.key == step:
			fmt.Fprintf(&b, "👉 <b>%s</b>\n", f.label)
		default:
			fmt.Fprintf(&b, "▫️ %s\n", f.label)
		}
	}
	return b.String()
}

func registrationFieldValue(key string, d domain.RegistrationDraft) string {
	switch key {
	case "sex":
		return sexLabel(d.Sex)
	case "age":
		if d.Age == 0 {
			return ""
		}
		return fmt.Sprintf("%d лет", d.Age)
	case "height":
		if d.HeightCM == 0 {
			return ""
		}
		return formatNumber(d.HeightCM) + " см"
	case "weight":
		if d.WeightKG == 0 {
			return ""
		}
		return formatNumber(d.WeightKG) + " кг"
	case "activity":
		return activityLabel(d.ActivityLevel)
	case "goal":
		return goalLabel(d.Goal)
	default:
		return ""
	}
}

func registrationPrompt(step string) string {
	switch step {
	case "sex":
		return "Выберите пол 👇"
	case "age":
		return "✏️ Сколько вам полных лет? <i>(например, 30)</i>"
	case "height":
		return "✏️ Какой у вас рост в сантиметрах? <i>(например, 178)</i>"
	case "weight":
		return "✏️ Какой у вас вес в килограммах? <i>(например, 72)</i>"
	case "activity":
		return "Какой у вас уровень активности? 👇"
	case "goal":
		return "Какая у вас цель? 👇"
	default:
		return "Отправьте /start, чтобы начать заново."
	}
}

func RegistrationDone(d domain.RegistrationDraft, targets domain.Targets) string {
	var b strings.Builder
	b.WriteString("🎉 <b>Профиль готов!</b>\n\n")
	b.WriteString("👤 <b>Ваши данные</b>\n")
	b.WriteString(profileInputLines(d.Sex, d.Age, d.HeightCM, d.WeightKG, d.ActivityLevel, d.Goal))
	b.WriteString("\n🔥 <b>Дневная норма</b>\n")
	b.WriteString(targetLines(targets.CaloriesKCal, targets.ProteinG, targets.FatG, targets.CarbsG))
	b.WriteString("\n🍽 Теперь добавляйте еду командой /add или просто присылайте фото — здесь или в группе с ботом.")
	return b.String()
}

func Profile(profile domain.Profile) string {
	var b strings.Builder
	b.WriteString("👤 <b>Ваш профиль</b>\n")
	b.WriteString(profileInputLines(profile.Sex, profile.Age, profile.HeightCM, profile.WeightKG, profile.ActivityLevel, profile.Goal))
	b.WriteString("\n🔥 <b>Дневная норма</b>")
	if profile.FormulaVersion == domain.ManualFormulaVersion {
		b.WriteString(" <i>(задана вручную)</i>")
	}
	b.WriteString("\n")
	b.WriteString(targetLines(profile.DailyCaloriesKCal, profile.DailyProteinG, profile.DailyFatG, profile.DailyCarbsG))
	return b.String()
}

func profileInputLines(sex string, age int, heightCM, weightKG float64, activity, goal string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "• Пол: <b>%s</b>\n", sexLabel(sex))
	fmt.Fprintf(&b, "• Возраст: <b>%d</b>\n", age)
	fmt.Fprintf(&b, "• Рост: <b>%s см</b>\n", formatNumber(heightCM))
	fmt.Fprintf(&b, "• Вес: <b>%s кг</b>\n", formatNumber(weightKG))
	fmt.Fprintf(&b, "• Активность: <b>%s</b>\n", activityLabel(activity))
	fmt.Fprintf(&b, "• Цель: <b>%s</b>\n", goalLabel(goal))
	return b.String()
}

func targetLines(calories, protein, fat, carbs int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "• Калории: <b>%d</b> ккал\n", calories)
	fmt.Fprintf(&b, "• Белки: <b>%d</b> г\n", protein)
	fmt.Fprintf(&b, "• Жиры: <b>%d</b> г\n", fat)
	fmt.Fprintf(&b, "• Углеводы: <b>%d</b> г\n", carbs)
	return b.String()
}

func MealInputPrompt() string {
	return "✍️ Что вы съели? Опишите блюдо и порцию следующим сообщением или пришлите фото."
}

func MealInputCancelled() string {
	return "Добавление отменено."
}

type settingsField struct {
	key, label string
}

var settingsFields = []settingsField{
	{"sex", "Пол"},
	{"weight", "Вес"},
	{"calories", "Калории"},
	{"protein", "Белки"},
	{"fat", "Жиры"},
	{"carbs", "Углеводы"},
}

func selectedSettingsFields(keys []string) []settingsField {
	if len(keys) == 0 {
		return settingsFields
	}
	selected := make([]settingsField, 0, len(keys))
	for _, f := range settingsFields {
		if slices.Contains(keys, f.key) {
			selected = append(selected, f)
		}
	}
	return selected
}

func SettingsMenu(current domain.ProfileSettings, invalid bool) string {
	var b strings.Builder
	b.WriteString("⚙️ <b>Настройки профиля</b>\n\n")
	for _, f := range settingsFields {
		fmt.Fprintf(&b, "• %s: <b>%s</b>\n", f.label, settingsFieldValue(f.key, current))
	}
	b.WriteString("\n")
	if invalid {
		b.WriteString("⚠️ <i>Сначала выберите кнопкой, что изменить.</i>\n\n")
	}
	b.WriteString("Что изменить? Выберите поле или настройте всё сразу 👇")
	return b.String()
}

func SettingsCard(step string, fields []string, current, draft domain.ProfileSettings, invalid bool) string {
	selected := selectedSettingsFields(fields)
	var b strings.Builder
	b.WriteString("⚙️ <b>Настройки профиля</b>\n")
	if len(selected) > 1 {
		b.WriteString("<i>Идём по полям по очереди: отправьте новое значение или оставьте текущее кнопкой.</i>\n\n")
	} else {
		b.WriteString("<i>Отправьте новое значение или оставьте текущее кнопкой.</i>\n\n")
	}
	reached := false
	for _, f := range selected {
		now := settingsFieldValue(f.key, current)
		switch {
		case f.key == step:
			reached = true
			fmt.Fprintf(&b, "👉 <b>%s</b>: сейчас %s\n", f.label, now)
		case reached:
			fmt.Fprintf(&b, "▫️ %s: %s\n", f.label, now)
		default:
			b.WriteString("✅ " + settingsChange(f.label, now, settingsFieldValue(f.key, draft)) + "\n")
		}
	}
	b.WriteString("\n")
	if invalid {
		b.WriteString("⚠️ <i>Не понял значение. Попробуйте ещё раз.</i>\n\n")
	}
	b.WriteString(settingsPrompt(step))
	return b.String()
}

func SettingsKeepLabel(step string, current domain.ProfileSettings) string {
	return "Оставить текущее: " + settingsFieldValue(step, current)
}

func SettingsSaved(fields []string, before, after domain.ProfileSettings) string {
	var b strings.Builder
	b.WriteString("✅ <b>Настройки сохранены</b>\n\n")
	for _, f := range selectedSettingsFields(fields) {
		b.WriteString("• " + settingsChange(f.label, settingsFieldValue(f.key, before), settingsFieldValue(f.key, after)) + "\n")
	}
	if after.TargetsDiffer(before) {
		b.WriteString("\n✍️ Норма КБЖУ теперь задана вручную. Вернуть расчёт по формуле — /start.")
	}
	return b.String()
}

func SettingsCancelled() string {
	return "Настройки не изменены."
}

func SettingsError() string {
	return "⚠️ Не удалось обновить настройки. Попробуйте ещё раз — /settings."
}

func NeedRegisterPrivate() string {
	return "Сначала зарегистрируйтесь — отправьте /start."
}

func MealQueued(ahead int) string {
	return fmt.Sprintf("⏳ Запись в очереди — перед вами %d. Разберу, как только освободится место.", ahead)
}

func SubscribeRequiredPrivate(label, url string) string {
	return "📢 Бот работает для подписчиков канала " + channelLink(label, url) +
		".\n\nПодпишитесь и нажмите «Я подписался» — сразу продолжим."
}

func SubscribeRequiredGroup(label, url string) string {
	return "📢 Бот работает для подписчиков канала " + channelLink(label, url) + ". Подпишитесь и повторите команду."
}

func SubscribeAlert(label string) string {
	return "Бот работает для подписчиков канала " + label + ". Подпишитесь и попробуйте снова."
}

func SubscriptionNotFound() string {
	return "Подписка пока не видна. Подпишитесь на канал и нажмите ещё раз."
}

func SubscriptionConfirmed() string {
	return "✅ Подписка подтверждена — спасибо! Можно пользоваться ботом."
}

func channelLink(label, url string) string {
	return `<a href="` + html.EscapeString(url) + `">` + html.EscapeString(label) + `</a>`
}

func settingsChange(label, before, after string) string {
	if before == after {
		return fmt.Sprintf("%s: <b>%s</b>", label, after)
	}
	return fmt.Sprintf("%s: <b>%s</b> <i>(было %s)</i>", label, after, before)
}

func settingsFieldValue(key string, s domain.ProfileSettings) string {
	switch key {
	case "sex":
		return sexLabel(s.Sex)
	case "weight":
		return formatNumber(s.WeightKG) + " кг"
	case "calories":
		return fmt.Sprintf("%d ккал", s.CaloriesKCal)
	case "protein":
		return fmt.Sprintf("%d г", s.ProteinG)
	case "fat":
		return fmt.Sprintf("%d г", s.FatG)
	case "carbs":
		return fmt.Sprintf("%d г", s.CarbsG)
	default:
		return ""
	}
}

func settingsPrompt(step string) string {
	switch step {
	case "sex":
		return "Выберите пол 👇"
	case "weight":
		return "✏️ Отправьте вес в килограммах <i>(например, 72.5)</i>"
	case "calories":
		return "✏️ Отправьте дневную норму калорий <i>(например, 2100)</i>"
	case "protein":
		return "✏️ Отправьте норму белков в граммах <i>(например, 140)</i>"
	case "fat":
		return "✏️ Отправьте норму жиров в граммах <i>(например, 70)</i>"
	case "carbs":
		return "✏️ Отправьте норму углеводов в граммах <i>(например, 240)</i>"
	default:
		return "Отправьте /settings, чтобы начать заново."
	}
}

func StatsCard(name string, mealCount int, totals domain.DailyTotals, p domain.Profile) string {
	var b strings.Builder
	b.WriteString("📊 <b>Статистика за сегодня</b>\n")
	fmt.Fprintf(&b, "<i>%s</i> · приёмов пищи: <b>%d</b>\n\n", html.EscapeString(name), mealCount)
	b.WriteString(statMacro("🔥", "Калории", totals.CaloriesKCal, float64(p.DailyCaloriesKCal), "ккал"))
	b.WriteString(statMacro("🥩", "Белки", totals.ProteinG, float64(p.DailyProteinG), "г"))
	b.WriteString(statMacro("🧈", "Жиры", totals.FatG, float64(p.DailyFatG), "г"))
	b.WriteString(statMacro("🍞", "Углеводы", totals.CarbsG, float64(p.DailyCarbsG), "г"))
	if mealCount == 0 {
		b.WriteString("За сегодня ещё нет подтверждённых записей. Добавьте еду через /add.")
	}
	return b.String()
}

func statMacro(icon, label string, value, target float64, unit string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s <b>%s</b>  %.0f / %.0f %s\n", icon, label, value, target, unit)
	b.WriteString(progressBar(value, target))
	b.WriteString("\n\n")
	return b.String()
}

func sexLabel(code string) string {
	switch code {
	case "male":
		return "Мужской"
	case "female":
		return "Женский"
	default:
		return ""
	}
}

func activityLabel(code string) string {
	switch code {
	case "low":
		return "Низкая"
	case "light":
		return "Лёгкая"
	case "moderate":
		return "Средняя"
	case "high":
		return "Высокая"
	case "athlete":
		return "Спорт"
	default:
		return ""
	}
}

func goalLabel(code string) string {
	switch code {
	case "lose":
		return "Похудение"
	case "maintain":
		return "Поддержание"
	case "gain":
		return "Набор"
	default:
		return ""
	}
}

func formatNumber(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func MealResult(view domain.MealResultView) string {
	analysis := view.Analysis
	profile := view.Profile
	var b strings.Builder
	b.WriteString("🔍")
	b.WriteString(escapeInline(analysis.Summary))
	if strings.TrimSpace(analysis.Notes) != "" {
		b.WriteString(" ")
		b.WriteString(escapeInline(analysis.Notes))
	}
	b.WriteString("\n\n")
	b.WriteString("| **Позиция** | **Белки** | **Углеводы** | **Жиры** | **Калории** |\n")
	b.WriteString("|:--------|------:|---------:|-----:|--------:|\n")
	for _, item := range analysis.Items {
		fmt.Fprintf(&b,
			"| %s | %.0f | %.0f | %.0f | %.0f |\n",
			tableCell(item.Name, 28),
			item.ProteinG,
			item.CarbsG,
			item.FatG,
			item.CaloriesKCal,
		)
	}
	fmt.Fprintf(&b,
		"| **Итого** | **%.0f** | **%.0f** | **%.0f** | **%.0f** |\n\n",
		analysis.Total.ProteinG,
		analysis.Total.CarbsG,
		analysis.Total.FatG,
		analysis.Total.CaloriesKCal,
	)
	dailyLimit := float64(profile.DailyCaloriesKCal)
	fmt.Fprintf(&b, "📊 Сегодня: **%.0f** / **%.0f** ккал\n", view.ConsumedToday, dailyLimit)
	b.WriteString(calorieProgressBar(view.ConsumedToday, dailyLimit))

	return truncateRunes(b.String(), 1000)
}

func calorieProgressBar(consumedToday, dailyLimit float64) string {
	return progressBar(consumedToday, dailyLimit)
}

const (
	onTargetTolerancePct   = 10
	nearTargetTolerancePct = 25
)

func progressBar(value, target float64) string {
	const width = 10
	if target <= 0 {
		return strings.Repeat("⬜", width) + " 0%"
	}

	ratio := math.Max(value/target, 0)
	filled := int(math.Round(math.Min(ratio, 1) * width))
	percent := math.Round(ratio * 100)
	block := "🟥"
	switch deviation := math.Abs(percent - 100); {
	case deviation <= onTargetTolerancePct:
		block = "🟩"
	case deviation <= nearTargetTolerancePct:
		block = "🟧"
	}
	return strings.Repeat(block, filled) + strings.Repeat("⬜", width-filled) + fmt.Sprintf(" %.0f%%", percent)
}

func Deleted() string {
	return "Запись удалена."
}

func AwaitCorrection() string {
	return "Отправьте корректировку текстом в этот чат. Например: \"курицы было 200 г, без масла\"."
}

func ReportMarkdown(user domain.ReportUser, reportDate time.Time, meals []domain.DailyMeal, totals domain.DailyTotals, rec *domain.DailyRecommendation) string {
	var b strings.Builder
	b.WriteString("## Отчет за ")
	b.WriteString(reportDate.Format("2006-01-02"))
	b.WriteString("\n\n")
	b.WriteString("**")
	b.WriteString(escapeInline(user.DisplayName))
	b.WriteString("**\n\n")

	if len(meals) == 0 {
		b.WriteString("Записей за вчера нет.")
		return b.String()
	}

	b.WriteString("| Время | Еда | Ккал | Б | Ж | У |\n")
	b.WriteString("|:------|:----|-----:|--:|--:|--:|\n")
	for _, meal := range meals {
		fmt.Fprintf(&b,
			"| %s | %s | %.0f | %.0f | %.0f | %.0f |\n",
			meal.CreatedAt.Format("15:04"),
			tableCell(meal.Summary, 40),
			meal.CaloriesKCal,
			meal.ProteinG,
			meal.FatG,
			meal.CarbsG,
		)
	}
	fmt.Fprintf(&b,
		"| **Итого** |  | **%.0f** | **%.0f** | **%.0f** | **%.0f** |\n\n",
		totals.CaloriesKCal,
		totals.ProteinG,
		totals.FatG,
		totals.CarbsG,
	)

	fmt.Fprintf(&b,
		"Норма: **%d** ккал, Б **%d**, Ж **%d**, У **%d**.\n",
		user.Profile.DailyCaloriesKCal,
		user.Profile.DailyProteinG,
		user.Profile.DailyFatG,
		user.Profile.DailyCarbsG,
	)
	fmt.Fprintf(&b, "Калории: **%.0f / %d**.\n\n", totals.CaloriesKCal, user.Profile.DailyCaloriesKCal)

	if rec != nil {
		b.WriteString("### Рекомендации\n")
		b.WriteString(escapeInline(rec.Summary))
		b.WriteString("\n\n")
		if rec.MacroAdvice != "" {
			b.WriteString(escapeInline(rec.MacroAdvice))
			b.WriteString("\n\n")
		}
		writeBulletSection(&b, "Сохранить", rec.KeepDoing)
		writeBulletSection(&b, "Улучшить", rec.Improve)
		writeBulletSection(&b, "Идеи на сегодня", rec.FoodIdeas)
		if strings.TrimSpace(rec.WarningNotes) != "" {
			b.WriteString("\n⚠️ ")
			b.WriteString(escapeInline(rec.WarningNotes))
			b.WriteString("\n")
		}
	}
	return b.String()
}

func writeBulletSection(b *strings.Builder, title string, items []string) {
	if len(items) == 0 {
		return
	}
	b.WriteString("**")
	b.WriteString(title)
	b.WriteString("**\n")
	for _, item := range items {
		if strings.TrimSpace(item) == "" {
			continue
		}
		b.WriteString("- ")
		b.WriteString(escapeInline(item))
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func ReportFallback(markdown string) string {
	if len(markdown) <= 4000 {
		return markdown
	}
	return truncateRunes(markdown, 4000)
}

func tableCell(value string, max int) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.ReplaceAll(value, "|", "\\|")
	value = strings.TrimSpace(value)
	return truncateRunes(value, max)
}

func escapeInline(value string) string {
	value = strings.ReplaceAll(value, "|", "\\|")
	return strings.TrimSpace(value)
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 1 {
		return string(runes[:max])
	}
	return string(runes[:int(math.Max(0, float64(max-1)))]) + "…"
}
