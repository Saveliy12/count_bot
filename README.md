# Семейные расходы

Telegram-бот, который понимает траты обычным текстом, и веб-панель для всей семьи.
Go-монолит, PostgreSQL, строгая JSON-схема для LLM и минималистичный интерфейс без
сборщика и фреймворков.

| Десктоп | Мобильный |
|---|---|
| ![Панель расходов](docs/screenshots/panel-desktop.png) | ![Панель на телефоне](docs/screenshots/panel-mobile.png) |

## Что умеет

- **Бот:** «кофе 350», «такси 1200 вчера», «сколько потратил на еду в этом месяце»,
  «последнее было не 350, а 450», «удали такси».
- **Права:** каждый член семьи видит и меняет только свои траты.
- **Панель:** все траты семьи, фильтры по участнику, периоду и категории, создание,
  редактирование и удаление. Без авторизации — как в тестовом задании.
- **AI:** модель только разбирает текст в типизированное намерение. Все проверки и
  действия выполняет Go.
- **Запуск:** приложение и PostgreSQL поднимаются одним `docker compose`.

## Стек

- Go 1.26, `net/http`, `log/slog`
- PostgreSQL 17, миграции goose
- Telegram Bot API (`go-telegram/bot`)
- Groq + GPT-OSS-120B, strict JSON Schema
- Vanilla HTML/CSS/JS, встроенная статика через `go:embed`
- Docker, Docker Compose

## Архитектура

```mermaid
flowchart LR
    TG[Telegram] --> BOT[Bot polling]
    BOT --> HANDLER[Handler + регистрация]
    HANDLER --> PROC[LLMProcessor]
    PROC --> LLM[Groq: strict JSON]
    PROC --> SERVICE[ExpenseService]
    WEB[Веб-панель] --> API[REST API]
    API --> SERVICE
    SERVICE --> REPO[Репозитории]
    REPO --> DB[(PostgreSQL)]
```

LLM не подключается к БД и не выполняет действия. Она возвращает `llm.Intent`,
а Go проверяет владельца, сумму, категорию, дату, цель изменения и подтверждение
удаления. Подробно — в [`docs/AI_SYSTEM.md`](docs/AI_SYSTEM.md).

## Быстрый старт

Нужны Docker и ключи Telegram/Groq.

```bash
cp .env.example .env
# заполни TELEGRAM_BOT_TOKEN и GROQ_API_KEY
docker compose up --build -d
docker compose ps
```

Открой **http://127.0.0.1:8083**.

На пустой базе Compose по умолчанию создаёт демо-участника, чтобы панель работала даже
без Telegram. Если ключей нет, приложение всё равно поднимется: бот отключится,
LLM-парсер будет заглушкой.

Остановка:

```bash
docker compose down          # данные в volume сохраняются
```

`docker compose down -v` удаляет базу — только если данные не нужны.

### Локально без Docker

Нужен запущенный PostgreSQL и переменные из `.env`:

```bash
go run ./cmd
```

`DATABASE_URL` должен указывать на локальный PostgreSQL. В Compose используется
отдельный `COMPOSE_DATABASE_URL` с хостом `postgres`, поэтому локальный и
контейнерный запуск не конфликтуют.

## Конфигурация

| Переменная | Назначение |
|---|---|
| `DATABASE_URL` | PostgreSQL для `go run` |
| `COMPOSE_DATABASE_URL` | PostgreSQL внутри Compose |
| `HTTP_ADDR` | адрес HTTP, по умолчанию `:8083` |
| `TELEGRAM_BOT_TOKEN` | токен бота; без него бот отключён |
| `GROQ_API_KEY` | ключ Groq; без него LLM на заглушке |
| `LLM_MODEL` | модель, по умолчанию `openai/gpt-oss-120b` |
| `LLM_BASE_URL` | OpenAI-совместимый endpoint Groq |
| `LLM_TIMEOUT` | общий таймаут LLM-запроса, по умолчанию `20s` |
| `TIMEZONE` | бизнес-зона для «сегодня», по умолчанию `Europe/Moscow` |
| `POSTGRES_DB/USER/PASSWORD` | БД Compose |
| `POSTGRES_HOST_PORT` | порт PostgreSQL на хосте, по умолчанию `55432` |
| `APP_HOST_PORT` | порт панели на хосте, по умолчанию `8083` |
| `SEED_DEMO_USER` | демо-участник на пустой БД; Compose включает по умолчанию |

Модель можно заменить без правок кода: `LLM_MODEL=qwen/qwen3.8-27b`.

## Как работает AI

1. Telegram-сообщение попадает в `LLMProcessor`.
2. Текст и текущая дата уходят в Groq со strict JSON Schema.
3. Модель возвращает поля: действие, сумму, категорию, дату, цель, период,
   комментарий или уточняющий вопрос.
4. Go проверяет данные и права, затем вызывает `ExpenseService`.
5. Для изменения/удаления цель ищется по «последнее», ID или описанию с учётом даты и
   категории. Удаление требует подтверждения «да/нет».
6. Итоги списков считаются SQL-агрегатом, а не по загруженной странице.

Безопасность:

- owner-scoped доступ на уровне сервиса;
- удаление атомарно проверяет снимок записи, чтобы не удалить изменённую трату;
- точное совпадение суммы (`350` не матчит `1350`);
- `old_amount` не даёт изменить не ту трату;
- prompt injection не обходит схему и проверки, но может повлиять на выбранное
  действие — поэтому удаление подтверждается.

### Качество модели

Opt-in eval: 33 русские фразы на create/list/update/delete/clarify, даты, цели,
категории и неоднозначности.

- GPT-OSS-120B: **33/33**;
- запуск на free tier: `LLM_EVAL_DELAY=18s`, чтобы не упираться в TPM;
- live-тесты не входят в обычный `go test` и требуют явных флагов.

## REST API

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/healthz` | liveness |
| `GET` | `/readyz` | готовность БД |
| `GET` | `/api/v1/users` | участники семьи |
| `GET` | `/api/v1/categories` | категории и русские подписи |
| `GET` | `/api/v1/expenses` | список + `total/count/has_more` |
| `POST` | `/api/v1/expenses` | создать трату |
| `GET` | `/api/v1/expenses/{id}` | одна трата |
| `PATCH` | `/api/v1/expenses/{id}` | изменить |
| `DELETE` | `/api/v1/expenses/{id}` | удалить |

Фильтры списка: `user_id`, `from`, `to`, `category`, `limit`, `offset`.
Суммы передаются строками, не float.

Пример:

```bash
curl 'http://127.0.0.1:8083/api/v1/expenses?limit=5'

curl -X POST http://127.0.0.1:8083/api/v1/expenses \
  -H 'Content-Type: application/json' \
  -d '{"user_id":900000000001,"amount":"350.50","category":"food","spent_at":"2026-10-21","comment":"кофе"}'
```

Ошибки имеют вид `{"error":{"code":"...","message":"...","field":"..."}}`.

## Как проверить руками

### Бот

1. Открой бота, отправь `/start`.
2. `кофе 350` → бот отвечает «Записал…».
3. `последнее было не 350, а 450` → пока единственная трата, сумма меняется.
4. `такси 1200 вчера` → вторая трата с вчерашней датой.
5. `сколько потратил на еду в этом месяце` → сумма и количество по еде.
6. `удали такси` → бот показывает трату и просит «да/нет».
7. `да` → запись удалена; `нет` → остаётся.
8. Отправь `/start` со второго Telegram-аккаунта: каждый видит только свои траты,
   а панель — обе записи.

Если в базе уже есть старые траты, «последнее» может относиться к другой записи.
Для чистого сценария используй отдельный тестовый Compose-проект или новую базу.

### Панель

1. Открой http://127.0.0.1:8083 — демо-участник уже есть.
2. Проверь фильтры по участнику, периоду и категории.
3. Нажми «Добавить трату», создай запись с суммой `123,45`.
4. Нажми «Изменить», поменяй сумму и комментарий.
5. Нажми «Удалить», проверь отмену и подтверждение.
6. Создай больше 50 записей и проверь «Показать ещё».
7. Сузь окно до телефона — таблица станет карточками.

Быстрая проверка ошибок: сумма `0`, `abc` или `10.999` должна дать понятную ошибку,
а не создать запись.

## Тесты

```bash
gofmt -l .
go vet ./...
go test ./...
go test -race ./internal/bot ./internal/repository ./internal/llm ./internal/config
go build ./...
```

С PostgreSQL (сервис Compose уже поднят):

```bash
TEST_DATABASE_URL='postgres://expenses:expenses@127.0.0.1:55432/expenses?sslmode=disable' \
  go test ./... -count=1
```

Живой LLM и полный eval:

```bash
LLM_INTEGRATION=1 TEST_DATABASE_URL='...' \
  go test ./internal/bot -run TestLLMProcessorWithGroq -v

LLM_EVAL=1 LLM_EVAL_DELAY=18s \
  go test ./internal/llm -run TestModelEvaluation -v -timeout 20m
```

Обычный `go test` не ходит в сеть. `TEST_DATABASE_URL` создаёт временные схемы и
удаляет их после тестов.

## Ограничения

- Pending-состояния (подтверждение удаления, уточнения) хранятся в памяти и теряются
  при рестарте.
- Несколько трат в одном сообщении пока не поддерживаются.
- Нет дедупликации повторно доставленных Telegram update.
- Подтверждение удаления — нативный `confirm`; владельца траты в панели изменить нельзя.
- Панель без авторизации — требование локального демо, не production-решение.
- Free tier Groq имеет TPM/TPD-лимиты; клиент делает retry, но при исчерпании квоты
  нужно подождать.
- Локальный запуск по HTTP; TLS и reverse proxy не входят в демо.

## Структура

```text
cmd/main.go
internal/
  api/            HTTP, REST панели, встроенная статика
  app/            DI и запуск HTTP + Telegram
  bot/            polling, handler, LLMProcessor, pending
  config/         env-конфигурация
  llm/            Groq-клиент, intent, prompt, eval
  model/          доменные типы
  repository/     PostgreSQL
  service/        бизнес-логика
  migrator/       goose
  testdb/         временные схемы для тестов
migrations/
docs/
  AI_SYSTEM.md
  AI_EXPLAINED_for_me_only.md
  AI_AGENT_WORKFLOW.md
  IMPLEMENTATION_LOG.md
compose.yaml
Dockerfile
```

## Документация

- [AI в проекте: техническое описание](docs/AI_SYSTEM.md)
- [AI простыми словами](docs/AI_EXPLAINED_for_me_only.md)
- [Как я работаю с ИИ-агентами](docs/AI_AGENT_WORKFLOW.md)
- [Журнал значимых изменений](docs/IMPLEMENTATION_LOG.md)
