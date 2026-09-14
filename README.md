<div align="center">

<img src="docs/logo.png" width="140" alt="Ojirun logo" />

# Ojirun

**An AI food diary for Telegram**

Snap a photo of your meal or describe it in a few words — Ojirun counts calories, protein, fat and carbs,
tracks them against your daily target and sends you a breakdown of yesterday every morning.

[![CI](https://github.com/alxrepin/ojirunbot/actions/workflows/ci.yml/badge.svg)](https://github.com/alxrepin/ojirunbot/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
![Go 1.26](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![PostgreSQL 17](https://img.shields.io/badge/PostgreSQL-17-4169E1?logo=postgresql&logoColor=white)
![Telegram Bot API 10.3](https://img.shields.io/badge/Telegram_Bot_API-10.3-26A5E4?logo=telegram&logoColor=white)

[Русская версия](README.ru.md)

</div>

> The bot's interface is in Russian.

## Features

- **Photo or text** — send a picture of your plate, a caption, or `/add buckwheat with chicken and a salad`. No forms, no food databases to search.
- **Personal daily target** — a short onboarding (sex, age, height, weight, activity, goal) turns into calorie and macro targets using the Mifflin-St Jeor formula. Got a plan from a coach? Enter it manually in `/settings`.
- **You stay in control** — every analysis comes as a card with *Accept*, *Edit* and *Delete*. Nothing counts until you accept it; a text correction produces a new revision.
- **Honest progress bars** — `/stats` shows today's calories and macros. The color reflects how close you are to the target, not how full the bar is: 🟩 within ±10%, 🟧 within ±25%, 🟥 further off — undereating is as bad as overeating.
- **Morning report** — at 09:00 you get a table of yesterday's meals, totals against your target and AI recommendations for today. No meals — no report.
- **Works in groups** — track food together with friends. Statuses, hints and the action buttons are ephemeral messages only the author sees; the group sees just the accepted meal and `/stats` cards.
- **Fixes after the fact** — reply `/yesterday` to move a forgotten meal to the previous day, or `/readd` to run recognition again.
- **Subscribers only** — access is limited to subscribers of the author's Telegram channel.
- **Fair use** — up to 10 meals per person per day across all chats, so AI costs stay predictable.

## How it works

1. Send `/start` in a private chat and answer six questions — the bot edits a single card as you go.
2. Send a photo or a description of your meal — in a private chat or in a group.
3. The bot replies «Принял запись» (or shows your place in the queue) and analyzes the meal with a vision model.
4. You get a card: recognized items, a nutrition table and today's calorie progress. Accept it, correct it in plain text or delete it.
5. The next morning a report on yesterday lands in your private chat.

## Commands

| Command | Where | What it does |
|---|---|---|
| `/start` | DM | onboarding and daily target calculation |
| `/add` | DM, group | add a meal: with text or a photo — right away, without — waits for your next message |
| `/stats` | DM, group | today's progress; in a group `/stats @username` shows a member who keeps a diary there |
| `/profile` | DM | your data and daily target, with buttons to edit or recalculate |
| `/settings` | DM | pick what to change — sex, weight, one macro target, the whole target or everything |
| `/yesterday` | DM, group | reply to a meal to move it to the previous day |
| `/readd` | DM, group | reply to a meal message to analyze it again |
| `/help` | DM, group | how it works |

In a private chat the main actions are also on a persistent keyboard: «🍽 Добавить еду», «📊 Статистика», «👤 Профиль», «⚙️ Настройки», «💡 Как это работает?».

## Self-hosting

### 1. Prepare Telegram

1. Create a bot with [@BotFather](https://t.me/BotFather).
2. Disable privacy mode (`/setprivacy` → *Disable*) so the bot sees photos in groups without a command.
3. Add the bot to your channel as an **administrator**. It needs no posting rights, but Telegram only reveals subscriber status to channel admins.

### 2. Configure

```bash
cp .env.example .env
```

At minimum:

```env
TELEGRAM_BOT_TOKEN=
TELEGRAM_REQUIRED_CHANNEL=@your_channel
OPENROUTER_API_KEY=
```

### 3. Run

```bash
docker compose up --build
```

That's it: PostgreSQL starts alongside, and migrations are applied when the bot starts. Anyone can add the bot to their group — there is no per-group setup.

<details>
<summary><b>All environment variables</b></summary>

| Variable | Default | Purpose |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | — | bot token |
| `TELEGRAM_REQUIRED_CHANNEL` | empty | required channel: `@username`, a `t.me/...` link or a numeric id; empty — the bot is open to everyone |
| `TELEGRAM_CHANNEL_URL` | derived from `@username` | link for the «Подписаться» button; required for a numeric id |
| `TELEGRAM_API_BASE_URL` | `https://api.telegram.org` | Bot API host or your own reverse proxy |
| `OPENROUTER_API_KEY` | — | OpenRouter key |
| `OPENROUTER_MODEL` | `google/gemini-3-flash-preview,google/gemini-2.5-flash,openai/gpt-5-mini` | comma-separated vision models: the first is primary, the rest are fallbacks |
| `OPENROUTER_BASE_URL` | `https://openrouter.ai/api/v1` | OpenRouter API URL |
| `OPENROUTER_HTTP_REFERER`, `OPENROUTER_APP_TITLE` | empty, `Ojirun` | app attribution in OpenRouter |
| `DATABASE_URL` | — | PostgreSQL connection string |
| `PROXY_URL` | empty | HTTP proxy for all outgoing requests, credentials inside the URL |
| `APP_TIMEZONE` | `Europe/Moscow` | time zone for days, reports and the limit |
| `DAILY_REPORT_TIME` | `09:00` | morning report time |
| `PHOTO_STORAGE_DIR` | `./storage/photos` | photo directory |
| `MAX_PHOTO_BYTES` | `10485760` | max downloaded photo size |
| `MAX_CONCURRENT_UPDATES` | `32` | updates processed concurrently |
| `MAX_CONCURRENT_AI` | `4` | `meal` queue workers, i.e. parallel AI analyses |
| `REPORT_WORKERS` | `2` | `report` queue workers |
| `MAX_MEALS_PER_DAY` | `10` | meals per user per day; `0` disables the limit |
| `APP_ENV`, `LOG_LEVEL` | `local`, `info` | environment and log level |

</details>

## Under the hood

- **Go 1.26** with a layered DDD / Clean Architecture design; dependencies point inward.
- **Own Telegram Bot API client** — Rich Messages, ephemeral messages and styled buttons aren't available in existing libraries.
- **OpenRouter structured output** — responses are validated against a JSON Schema and normalized before they reach the database; if a model is down, the client falls over to the next one in the list.
- **Durable job queue in PostgreSQL** — meal analysis and reports run on separate worker pools (`FOR UPDATE SKIP LOCKED` with leases), survive restarts and can be shared across several instances.
- **Idempotency everywhere** — callback actions check the author and repeat safely, a daily report is never sent twice, the meal limit is enforced under an advisory lock.

### Project structure

```text
cmd/bot/                            bot entrypoint
cmd/migrate/                        standalone migration run
internal/bootstrap/                 composition root: config, wiring, runtime
internal/context/
  domain/                           entities, target formula, statuses, limits
  application/service/              onboarding, settings, subscription, meal pipeline, reports, job queue
  application/usecase/              add meal with limit, meal actions, profile, stats
  infrastructure/telegram/          Telegram Bot API client
  infrastructure/openrouter/        OpenRouter structured output client
  infrastructure/files/             local photo storage
  infrastructure/db/postgres/       pool, migrations, per-aggregate repositories
  presentation/telegram/            routing, handlers, callbacks, menu, ephemeral replies
  presentation/telegram/render/     texts and Rich Markdown
migrations/                         PostgreSQL schema
prompts/                            AI prompts
```

## Development

```bash
docker compose -f compose.local.yml up -d   # PostgreSQL only
go run ./cmd/bot

go test -race ./...
golangci-lint run ./...
```

Standalone migration run:

```bash
DATABASE_URL="postgres://nutrition:nutrition@localhost:5432/nutrition?sslmode=disable" go run ./cmd/migrate
```

CI (GitHub Actions) runs golangci-lint, `go mod tidy -diff`, `go vet`, tests with `-race`, all migrations on an empty PostgreSQL 17 twice to check idempotency, and a Docker image build. Dependabot proposes updates weekly. The code is written without comments — explanations live in this README and [CLAUDE.md](CLAUDE.md).

## Troubleshooting

- **`cannot check channel subscriptions` at startup** — the bot isn't a channel admin, so Telegram hides subscriber status. Until fixed, the check lets everyone through: API errors never block users.
- **The bot answers in DMs but ignores photos in a group** — privacy mode is still on; disable it or make the bot a group admin.
- **Something is stuck** — look at the `jobs` table. Completed jobs are deleted, failed ones stay with status `failed`:

  ```sql
  SELECT queue, kind, status, attempts, last_error, run_at
  FROM jobs
  ORDER BY created_at DESC
  LIMIT 20;
  ```

- **Command hints didn't appear** — Telegram updates the command menu with a delay; reopen the chat.

## License

[MIT](LICENSE) © Alex Repin
