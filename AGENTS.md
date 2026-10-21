# AGENTS.md

Правила для coding-агентов, работающих с этим репозиторием.

## Что это за проект

Go-монолит для семейных расходов: Telegram-бот, REST API и PostgreSQL.
LLM используется только как парсер текста в типизированное намерение. Бизнес-логика,
валидация и запись в БД — на Go.

## Структура

- `cmd/main.go` — точка входа, `.env`, graceful shutdown.
- `internal/app` — сборка зависимостей и запуск HTTP + Telegram.
- `internal/api` — HTTP-роутер, middleware, REST панели.
- `internal/bot` — Telegram polling, handler, LLMProcessor, pending-состояния.
- `internal/llm` — Groq-клиент, strict JSON Schema, intent, prompt, eval.
- `internal/config` — env-конфигурация.
- `internal/model` — доменные типы (деньги — `decimal`, дата — `time.Time`).
- `internal/service` / `internal/repository` — бизнес-логика и PostgreSQL.
- `internal/migrator`, `migrations` — goose-миграции.
- `internal/testdb` — временные схемы PostgreSQL для интеграционных тестов.
- `docs/` — AI-документация и журнал крупных изменений.

## Главные правила

1. **Маленькие итерации.** Один этап — одна проверяемая правка. Сначала план, потом код.
2. **LLM не пишет в БД.** Провайдер возвращает `llm.Intent`; выполнение действий идёт
   только через `service.ExpenseService`.
3. **Деньги — только `shopspring/decimal`.** Не использовать `float64`.
4. **Владелец траты проверяется на уровне сервиса.** Чужие траты не должны быть доступны.
5. **Даты трат** — календарные, `YYYY-MM-DD`; бизнес-дата берётся из `TIMEZONE`.
6. **Удаление в боте — только после подтверждения** и только через
   `DeleteExpenseIfUnchanged` (атомарная проверка снимка).
7. **Секреты не читать и не печатать.** Не выводить содержимое `.env`, токены и ключи.
8. **Git:** не коммитить, не пушить и не создавать репозиторий без явного разрешения.
9. **Миграции только добавлять.** Не переписывать уже применённые файлы.
10. **Документация должна соответствовать коду.** Если поведение изменилось — обновить
    `docs/AI_SYSTEM.md`, а при необходимости и остальные документы.
11. **Журнал значимых изменений.** После изменений API, схемы БД, межслойного flow или
    инфраструктуры добавить короткую запись в `docs/IMPLEMENTATION_LOG.md`:
    задача → что изменилось → как работает теперь → что проверено → ограничения.
    Мелкий рефакторинг и форматирование в журнал не заносить.

## Проверки

```bash
gofmt -l -w .
go vet ./...
go test ./...
go test -race ./internal/bot ./internal/repository ./internal/llm ./internal/config
go build ./...
```

Интеграционные тесты требуют PostgreSQL:

```bash
TEST_DATABASE_URL='postgres://expenses:expenses@127.0.0.1:55432/expenses?sslmode=disable' \
  go test ./... -count=1
```

Живые тесты с Groq запускаются только явно:

```bash
# сквозной сценарий create → list → update → delete
LLM_INTEGRATION=1 TEST_DATABASE_URL='...' \
  go test ./internal/bot -run TestLLMProcessorWithGroq -v

# оценка качества модели
LLM_EVAL=1 LLM_EVAL_LIMIT=10 LLM_EVAL_DELAY=3s \
  go test ./internal/llm -run TestModelEvaluation -v
```

## Docker Compose

```bash
docker compose up --build -d
docker compose ps
docker compose logs -f app
docker compose down          # volume с данными сохраняется
```

- `DATABASE_URL` в `.env` нужен для локального `go run` и указывает на хост.
- В Compose приложение получает отдельный `COMPOSE_DATABASE_URL` с хостом `postgres`.
  Если меняешь `POSTGRES_*`, обнови и `COMPOSE_DATABASE_URL`.
- Проверено на изолированном проекте: миграции, CRUD, graceful stop и сохранение
  данных после перезапуска.
- Compose по умолчанию включает `SEED_DEMO_USER=true`: на пустой БД создаётся
  «Демо-пользователь», чтобы панель была рабочей без Telegram. Если пользователи уже
  есть, seed не срабатывает.
- Наружу публикуются только loopback-порты: `APP_HOST_PORT` (8083) и
  `POSTGRES_HOST_PORT` (55432).
- Для изолированных проверок используй `docker compose -p <test-name>` и отдельный
  временный env-файл. `docker compose down -v` удаляет данные и допустим только для
  явно тестового project name.

## Текущие ограничения

- Веб-панель и README ещё не сделаны; Compose уже проверен на изолированном запуске.
- Если локальный `go run` занимает порт 8083, останови его перед `docker compose up`.
- Pending-состояния (подтверждение удаления/уточнения) хранятся в памяти и теряются
  при рестарте.
- Несколько трат в одном сообщении пока не поддерживаются.
- REST API панели работает без авторизации — осознанное решение для локального демо;
  сам веб-интерфейс ещё не сделан.
