# Ожирун (ojirun)

Telegram-бот — дневник питания. Пришлите фото или описание еды: Ожирун посчитает КБЖУ через AI, сравнит с вашей дневной нормой и утром пришлет разбор вчерашнего дня. Работает в личных сообщениях и в любых группах, пользоваться им могут подписчики канала автора.

## Возможности

- Регистрация в ЛС через `/start`: пол, возраст, рост, вес, активность и цель превращаются в дневную норму КБЖУ по Mifflin-St Jeor.
- Меню в ЛС — постоянная клавиатура: «🍽 Добавить еду», «📊 Статистика», «👤 Профиль», «⚙️ Настройки», «💡 Как это работает?».
- Добавление еды: `/add` открывает режим ввода с кнопкой отмены, `/add` с текстом обрабатывается сразу, фото с подписью — без команды.
- Анализ через OpenRouter со structured output и цепочкой резервных vision-моделей; карточка КБЖУ с кнопками «Принять», «Редактировать», «Удалить».
- Корректировка текстом создает новую ревизию анализа, история сохраняется.
- `/settings` — ручная правка пола, веса и нормы КБЖУ, например нормы от тренера.
- `/stats` — прогресс за сегодня по каждому макросу. Цвет бара показывает близость к норме: 🟩 в пределах ±10%, 🟧 ±25%, 🟥 дальше. В группе `/stats @имя` показывает участника, который ведет дневник в этой группе.
- `/yesterday` и `/readd` ответом на сообщение: перенести запись на вчера или повторить распознавание.
- Утренний отчет в ЛС с таблицей приемов пищи и AI-рекомендациями — только тем, у кого вчера были подтвержденные записи.
- Группы без флуда: статусы, подсказки и карточка с кнопками приходят эфемерными сообщениями, которые видит только автор; всей группе видны итог принятой еды и статистика.
- Доступ только для подписчиков канала `TELEGRAM_REQUIRED_CHANNEL`.
- Антифрод: не больше `MAX_MEALS_PER_DAY` (по умолчанию 10) добавлений еды в сутки на человека, суммарно в ЛС и группах.
- Очередь задач в PostgreSQL: анализ еды и отчеты разбирают пулы воркеров, работа переживает перезапуск, несколько инстансов делят нагрузку.

## Стек

Go 1.26, PostgreSQL 17 (pgx v5), Telegram Bot API 10.3 через собственный HTTP-клиент (Rich Messages, эфемерные сообщения, цветные кнопки), OpenRouter, Docker Compose.

## Настройка

1. Создайте бота в [@BotFather](https://t.me/BotFather).
2. Отключите privacy mode (`/setprivacy` → Disable), чтобы в группах бот видел фото без команды.
3. Добавьте бота **администратором** в свой канал. Права на публикацию не нужны, но статус подписчиков Telegram отдает только администраторам канала.
4. Скопируйте `.env.example` в `.env` и заполните минимум:

```env
TELEGRAM_BOT_TOKEN=
TELEGRAM_REQUIRED_CHANNEL=@your_channel
OPENROUTER_API_KEY=
```

Добавить бота в свою группу может любой пользователь, отдельной настройки групп нет.

### Переменные окружения

| Переменная | По умолчанию | Назначение |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | — | токен бота |
| `TELEGRAM_REQUIRED_CHANNEL` | пусто | канал для обязательной подписки: `@username`, ссылка `t.me/...` или числовой id; пусто — бот открыт всем |
| `TELEGRAM_CHANNEL_URL` | из `@username` | ссылка для кнопки «Подписаться»; обязательна для числового id |
| `TELEGRAM_API_BASE_URL` | `https://api.telegram.org` | хост Bot API или свой reverse-proxy |
| `OPENROUTER_API_KEY` | — | ключ OpenRouter |
| `OPENROUTER_MODEL` | `google/gemini-3-flash-preview,google/gemini-2.5-flash,openai/gpt-5-mini` | vision-модели через запятую: первая основная, остальные резервные |
| `OPENROUTER_BASE_URL` | `https://openrouter.ai/api/v1` | адрес OpenRouter API |
| `OPENROUTER_HTTP_REFERER`, `OPENROUTER_APP_TITLE` | пусто, `Ojirun` | атрибуция приложения в OpenRouter |
| `DATABASE_URL` | — | строка подключения PostgreSQL |
| `PROXY_URL` | пусто | HTTP-прокси для всех исходящих запросов, креды внутри URL |
| `APP_TIMEZONE` | `Europe/Moscow` | таймзона дня, отчетов и лимита |
| `DAILY_REPORT_TIME` | `09:00` | время утренних отчетов |
| `PHOTO_STORAGE_DIR` | `./storage/photos` | каталог фото |
| `MAX_PHOTO_BYTES` | `10485760` | лимит размера скачиваемого фото |
| `MAX_CONCURRENT_UPDATES` | `32` | сколько updates обрабатывается одновременно |
| `MAX_CONCURRENT_AI` | `4` | воркеры очереди `meal`, то есть параллельные AI-анализы |
| `REPORT_WORKERS` | `2` | воркеры очереди `report` |
| `MAX_MEALS_PER_DAY` | `10` | лимит добавлений еды на пользователя в сутки; `0` — без лимита |
| `APP_ENV`, `LOG_LEVEL` | `local`, `info` | окружение и уровень логов |

## Запуск

```bash
docker compose up --build
```

Локально: PostgreSQL в контейнере, бот из исходников.

```bash
docker compose -f compose.local.yml up -d
go run ./cmd/bot
```

Миграции применяются при старте бота. Отдельный прогон:

```bash
DATABASE_URL="postgres://nutrition:nutrition@localhost:5432/nutrition?sslmode=disable" go run ./cmd/migrate
```

## Разработка

```bash
go test -race ./...
golangci-lint run ./...
```

Линтер — golangci-lint v2, конфигурация в `.golangci.yml`. Код пишется без комментариев.

### CI

GitHub Actions (`.github/workflows/ci.yml`) запускается на push в `main`/`master`, на pull request и вручную:

- **Lint** — golangci-lint v2.12;
- **Test** — `go mod tidy -diff`, `go vet`, сборка, тесты с `-race` и покрытием;
- **Migrations** — все миграции на пустом PostgreSQL 17 и повторный прогон для проверки идемпотентности;
- **Docker image** — сборка образа.

Dependabot (`.github/dependabot.yml`) раз в неделю предлагает обновления Go-модулей, GitHub Actions и базовых Docker-образов.

## Команды

Бот при старте регистрирует подсказки через `setMyCommands`:

- в ЛС: `/add`, `/stats`, `/profile`, `/settings`, `/yesterday`, `/readd`, `/help`, `/start`;
- в группах: `/add`, `/stats`, `/help`, `/yesterday`, `/readd`.

Telegram может обновлять меню команд с задержкой: если подсказки не появились, переоткройте чат.

## Диагностика

- `cannot check channel subscriptions` в логах при старте — бот не администратор канала, и Telegram не отдает ему статусы подписчиков. Пока это не исправлено, проверка подписки пропускает всех: ошибки API не блокируют пользователей.
- Бот отвечает в ЛС, но не видит фото в группе — не отключен privacy mode, либо сделайте бота администратором группы.
- Очередь задач — таблица `jobs`. Выполненные задачи удаляются, упавшие остаются со статусом `failed`:

```sql
SELECT queue, kind, status, attempts, last_error, run_at
FROM jobs
ORDER BY created_at DESC
LIMIT 20;
```

## Структура

Слоистая архитектура (DDD / Clean Architecture), зависимости направлены внутрь.

```text
cmd/bot/                            entrypoint бота
cmd/migrate/                        отдельный прогон миграций
internal/bootstrap/                 composition root: конфиг, сборка зависимостей, runtime
internal/context/
  domain/                           сущности, формула нормы, статусы, лимиты
  application/service/              регистрация, настройки, подписка, пайплайн еды, отчеты, очередь задач
  application/usecase/              добавление еды с лимитом, действия с записями, профиль, статистика
  infrastructure/telegram/          клиент Telegram Bot API
  infrastructure/openrouter/        клиент OpenRouter structured output
  infrastructure/files/             локальное хранилище фото
  infrastructure/db/postgres/       пул, миграции, репозитории по агрегатам
  presentation/telegram/            роутинг, хендлеры, callbacks, меню, эфемерные ответы
  presentation/telegram/render/     тексты и Rich Markdown
migrations/                         схема PostgreSQL
prompts/                            промпты для AI
.github/                            CI и Dependabot
```
