# TripGo

Сервис поездок для лабораторной работы по Go и PostgreSQL. Сервис хранит поездки, запрещает две активные поездки на одного водителя, умеет создавать, получать и завершать поездку, а также отдаёт health/ready статусы.

## Быстрый запуск

1. Поднять локальное окружение:

```bash
tripgoctl environment start
```

2. Скопировать пример переменных окружения и при необходимости поправить значения:

```bash
cp .env.example .env
```

3. Накатить миграции:

```bash
make migrate
```

4. Запустить сервис:

```bash
make run
```

По умолчанию сервер слушает `:8080`.

## Команды

```bash
make migrate      # apply goose migrations
make migrate_down # rollback migrations
make run          # start service locally
make generate     # generate OpenAPI code
make lint         # run golangci-lint via go tool
make test         # unit tests with race detector and coverage summary
make test-integration # PostgreSQL repository integration tests
```

`make test-integration` uses the PostgreSQL URL from `.env` and applies migrations first. Start the work environment with `tripgoctl environment start` before running it.

Также можно проверить сборку напрямую:

```bash
go build ./...
```

## Переменные окружения

Основные переменные перечислены в [.env.example](.env.example). Важные обязательные значения:

```bash
HTTP_ADDR=:8080
DATABASE_URL=postgres://tripgo:tripgo@localhost:5432/tripgo?sslmode=disable
SHUTDOWN_TIMEOUT=10s
READ_TIMEOUT=5s
READ_HEADER_TIMEOUT=5s
WRITE_TIMEOUT=5s
IDLE_TIMEOUT=30s

DATABASE_MAX_CONNS=10
DATABASE_MIN_CONNS=2
DATABASE_MAX_CONN_LIFETIME=30m
DATABASE_CONNECT_TIMEOUT=5s
DATABASE_QUERY_TIMEOUT=3s
```

Приложение валидирует обязательные переменные при старте и падает с понятной ошибкой, если их нет.

## API

Сервис реализует HTTP API по OpenAPI контракту:

- `GET /health` — базовый статус сервиса
- `GET /ready` — readiness, проверяет доступность PostgreSQL
- `POST /api/v1/trips` — создать поездку
- `GET /api/v1/trips/{tripId}` — получить поездку
- `POST /api/v1/trips/{tripId}/finish` — завершить поездку

Ошибки возвращаются в формате `application/problem+json` с кодами:

- `invalid_request` — 400
- `trip_not_found` — 404
- `driver_busy` — 409
- `trip_completed` — 409
- `internal_error` — 500

## Архитектура

Зависимости направлены внутрь: HTTP-адаптеры и PostgreSQL-репозиторий зависят от usecase/model/domain, usecase — от model/domain и собственных интерфейсов, домен зависит только от стандартной библиотеки Go.

- `cmd/trip-service` — загрузка конфигурации, запуск HTTP-сервера и обработка сигналов
- `internal/container` — единственное место создания пула, репозиториев, менеджера транзакций, usecase и HTTP-обработчиков
- `internal/config` — чтение env и проверка обязательных параметров
- `internal/domain/trip` — статусы, переход состояния поездки и доменные ошибки
- `internal/model` — структуры данных, передаваемые между слоями и адаптерами
- `internal/usecase/trip/{create,get,finish}` — сценарии приложения; интерфейсы зависимостей объявлены в `contract.go` соответствующего usecase
- `internal/repository` — PostgreSQL-реализации доступа к поездкам и истории статусов
- `internal/infra/postgres` — пул PostgreSQL и контекстный менеджер транзакций
- `internal/http/trip/{create,get,finish}` — HTTP-адаптеры по одному пакету на операцию; их интерфейсы находятся рядом в `contract.go`
- `internal/http/domainerror` и `internal/http/response` — преобразование доменных ошибок и запись HTTP-ответов
- `internal/httpserver` — только регистрация маршрутов
- `internal/generated` — типы и серверный контракт, сгенерированные из OpenAPI

`main` не создаёт репозитории и не знает конкретных реализаций usecase: он получает собранное приложение через `container.New` и управляет жизненным циклом сервера. Интерфейсы объявлены у потребителей: usecase формулирует нужные ему репозитории и транзакционный менеджер, HTTP-адаптеры — необходимые операции usecase.

Отклонение от примера раскладки: репозитории поездки и истории статусов находятся в общем пакете `internal/repository`, поскольку оба обслуживают один агрегат и используют один контекст транзакции. Usecase при этом зависит от отдельных узких интерфейсов для каждого сценария.

## Observability

Сервис экспортирует HTTP- и PostgreSQL-трейсы и метрики через OTLP/HTTP в Collector. Входящий запрос получает span вида `POST /api/v1/trips`; SQL-инструментирование подключено к pgx pool и не записывает SQL или параметры запросов. Логи — JSON `slog` в stdout; middleware добавляет `service`, `trace_id` и `span_id`, а также method, route template, статус и длительность.

Настройки OTel берутся из стандартных `OTEL_*` переменных в `.env`, созданном `tripgoctl`. После запуска отправь запрос на создание поездки и найди его в Jaeger; метрики `http.server.request.duration`, `db.client.operation.duration` и `tripgo.trips.created` доступны в Grafana через Prometheus datasource. Для локальной проверки exporter настраивается `OTEL_EXPORTER_OTLP_ENDPOINT` и `OTEL_EXPORTER_OTLP_PROTOCOL=http/protobuf`.

## Решения

### 1. Менеджер транзакций

Для создания поездки используется один `Do(ctx, fn)` вызов. Транзакция открывается в `internal/infra/postgres/transaction` и передаётся через контекст в репозитории. Если `fn` возвращает ошибку или происходит паника, выполняется `ROLLBACK`. Если всё прошло успешно — `COMMIT`.

Выбран уровень изоляции: `ReadCommitted`.

Почему именно он:
- достаточно для задачи;
- даёт корректную видимость данных на уровне строки/записи;
- не усложняет логику и подходит для сценария “одна активная поездка на одного водителя”.

### 2. Запрет двух активных поездок

Запрет реализован в двух слоях:

1. На уровне БД — частичный уникальный индекс по `driver_id` для `status = 'active'`:

```sql
CREATE UNIQUE INDEX trips_one_active_per_driver_idx ON trips (driver_id) WHERE status = 'active';
```

2. На уровне приложения — usecase проверяет, есть ли активная поездка у водителя; нарушение уникального индекса при гонке переводится репозиторием в доменную ошибку `trip.ErrDriverBusy`.

Так обеспечивается корректное поведение при конкурентных запросах: только одна поездка может быть активной на одного водителя.

### 3. Атомарность создания поездки

При создании поездки выполняется один блок внутри транзакции:

- `INSERT` в `trips`
- `INSERT` в `trip_status_history`

Если вторая вставка падает, транзакция откатывается целиком, и в `trips` не остаётся “висящей” записи.

## Проверка работы

### Health / Ready

```bash
curl -i http://localhost:8080/health
curl -i http://localhost:8080/ready
```

Ожидания:
- `GET /health` → `200 OK`
- `GET /ready` → `200 OK`, если БД доступна; `503`, если PostgreSQL недоступен

### Создание поездки

```bash
curl -sS -X POST http://localhost:8080/api/v1/trips \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
    "driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
    "start_point":{"latitude":59.9398,"longitude":30.3146},
    "end_point":{"latitude":59.9290,"longitude":30.3626},
    "price":1450
  }'
```

### Повторный create для того же водителя

Ожидается `409 Conflict` с кодом `driver_busy`.

### Получение и завершение поездки

```bash
curl -sS http://localhost:8080/api/v1/trips/<trip_id>
curl -sS -X POST http://localhost:8080/api/v1/trips/<trip_id>/finish -i
```

После завершения повторный `finish` должен вернуть `409 trip_completed`.

### Конкурентное создание

```bash
seq 20 | xargs -P20 -I{} curl -s -o /dev/null -w '%{http_code}\n' \
  -X POST http://localhost:8080/api/v1/trips \
  -H 'Content-Type: application/json' \
  -d '{
    "user_id":"5cb72c04-7650-45c9-a79b-bcdba0631e0c",
    "driver_id":"8860b315-ec86-42eb-a17c-7c163d721ff5",
    "start_point":{"latitude":59.9398,"longitude":30.3146},
    "end_point":{"latitude":59.9290,"longitude":30.3626},
    "price":1450
  }' | sort | uniq -c
```

Ожидаемый результат: один `201`, остальные `409`.

### Graceful shutdown

```bash
kill -TERM $(lsof -ti :8080 | head -n 1)
```

Ожидание:
- сервер перестаёт принимать новые запросы;
- активный запрос успевает завершиться;
- после `SHUTDOWN_TIMEOUT` сервис корректно завершает работу.

## Примечание

README покрывает требования лабораторной работы по запуску, конфигурации, архитектуре и принятым решениям. Он должен остаться актуальным при изменениях в контракте и миграциях.
