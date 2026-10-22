# AGENTS.md

Guidance for coding agents working in this repository.

This file is in English on purpose: coding agents read it. User-facing docs are in
Russian.

## Project

Go monolith for family expenses: a Telegram bot, a REST API and PostgreSQL.
The LLM is only a parser: it turns Russian text into a typed `llm.Intent`.
Validation, ownership checks and all database writes stay in Go.

## Layout

- `cmd/main.go` — entry point, `.env`, graceful shutdown.
- `internal/app` — dependency wiring, HTTP + Telegram startup.
- `internal/api` — HTTP router, middleware, panel REST API.
- `internal/api/web` — embedded panel assets (`go:embed`).
- `internal/bot` — Telegram polling, handler, `LLMProcessor`, pending state.
- `internal/llm` — Groq client, strict JSON Schema, intent, prompt, eval.
- `internal/config` — environment configuration.
- `internal/model` — domain types (money is `decimal`, dates are `time.Time`).
- `internal/service`, `internal/repository` — business logic and PostgreSQL.
- `internal/migrator`, `migrations` — goose migrations.
- `internal/testdb` — temporary PostgreSQL schemas for integration tests.
- `docs/` — AI notes, screenshots and the change log.
- `README.md` — quick start, architecture, API and checks.

## Rules

1. Work in small iterations. Plan first, then code; one change should be easy to verify.
2. The LLM never writes to the database. Providers return `llm.Intent`; actions go
   through `service.ExpenseService`.
3. Money is `shopspring/decimal`. Do not introduce `float64` for amounts.
4. Ownership is enforced in the service layer. Never expose another user's expenses.
5. Dates are calendar dates (`YYYY-MM-DD`); "today" comes from `TIMEZONE`.
6. Bot deletion requires confirmation and must go through `DeleteExpenseIfUnchanged`,
   which checks the row snapshot.
7. Never read or print secrets. Do not output `.env`, tokens or API keys.
8. Do not commit, push or create a repository without explicit permission.
9. Only add migrations. Do not rewrite files that have already been applied.
10. Keep docs in sync with code. If behavior changes, update `docs/AI_SYSTEM.md` and
    anything else affected.
11. After API, schema, cross-layer or infrastructure changes, add a short
    `docs/IMPLEMENTATION_LOG.md` entry: what changed, how it was checked, what is left.
    Skip formatting and small refactors.

## Checks

```bash
gofmt -l -w .
go vet ./...
go test ./...
go test -race ./internal/bot ./internal/repository ./internal/llm ./internal/config
go build ./...
```

Integration tests need PostgreSQL:

```bash
TEST_DATABASE_URL='postgres://expenses:expenses@127.0.0.1:55432/expenses?sslmode=disable' \
  go test ./... -count=1
```

Live Groq tests are opt-in:

```bash
# end-to-end create → list → update → delete
LLM_INTEGRATION=1 TEST_DATABASE_URL='...' \
  go test ./internal/bot -run TestLLMProcessorWithGroq -v

# small eval sample
LLM_EVAL=1 LLM_EVAL_LIMIT=10 LLM_EVAL_DELAY=3s \
  go test ./internal/llm -run TestModelEvaluation -v

# full eval on the free tier
LLM_EVAL=1 LLM_EVAL_DELAY=18s \
  go test ./internal/llm -run TestModelEvaluation -v -timeout 20m
```

## Docker Compose

```bash
docker compose up --build -d
docker compose ps
docker compose logs -f app
docker compose down          # keeps the data volume
```

- `DATABASE_URL` in `.env` is for local `go run` and points at the host.
- Inside Compose the app uses `COMPOSE_DATABASE_URL` with host `postgres`. If you
  change `POSTGRES_*`, update it too.
- Compose sets `SEED_DEMO_USER=true` by default. On an empty database it creates
  "Демо-пользователь" so the panel works without Telegram; if users exist, the seed
  does nothing.
- Only loopback ports are published: `APP_HOST_PORT` (8083) and `POSTGRES_HOST_PORT`
  (55432).
- Use `docker compose -p <test-name>` with a temporary env file for isolated checks.
  `docker compose down -v` deletes data and is fine only for a test project.

## Known limitations

See the limitations section in `README.md`.
