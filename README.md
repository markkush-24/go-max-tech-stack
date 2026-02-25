# pet-study

HTTP-сервис на Go (net/http, Go 1.25). Ресурс: **User**. Есть версии API **v1** и **v2** (breaking change по полю
имени).

## Запуск

### Требования

- Go **1.25**

### Локально

**Linux/macOS (bash/zsh):**

```bash
go run ./cmd/api
```

**Windows PowerShell:**

```powershell
go run .\cmd\api
```

По умолчанию сервер слушает `:8080`.

## Конфигурация (ENV)

Все значения читаются из переменных окружения. Если переменная **не задана** — используется дефолт.
Если переменная **задана, но пустая/некорректная** — приложение завершится с ошибкой.

| Переменная                                   |      Тип |                       Дефолт | Описание                                                                               |
|----------------------------------------------|---------:|-----------------------------:|----------------------------------------------------------------------------------------|
| `HTTP_ADDR`                                  |   string |                      `:8080` | Адрес для `http.Server.Addr`                                                           |
| `HTTP_READ_HEADER_TIMEOUT`                   | duration |                         `5s` | Таймаут чтения заголовков (slowloris mitigation)                                       |
| `HTTP_READ_TIMEOUT`                          | duration |                        `10s` | Таймаут чтения запроса целиком                                                         |
| `HTTP_WRITE_TIMEOUT`                         | duration |                        `15s` | Таймаут записи ответа                                                                  |
| `HTTP_IDLE_TIMEOUT`                          | duration |                       `120s` | Keep-alive idle timeout                                                                |
| `HTTP_SHUTDOWN_TIMEOUT`                      | duration |                        `10s` | Дедлайн на graceful shutdown                                                           |
| `HTTP_MAX_HEADER_BYTES`                      |      int | `http.DefaultMaxHeaderBytes` | Лимит размера заголовков                                                               |
| `HTTP_DEBUG`                                 |     bool |                      `false` | Включить debug endpoints (`/debug/...`)                                                |
| `DB_DSN`                                     |   string |                    *(пусто)* | Опционально (в Step 3 используется in-memory repo). Если задана — должна быть непустой |
| `WORKERS_COUNT`                              |      int |                           10 | Кол-во воркеров worker pool для обработки async jobs (должно быть > 0)                 |
| `QUEUE_SIZE`                                 |      int |                           10 | Размер bounded очереди для async jobs                                                  |
| `RATE_LIMIT_RPS`                             |      int |                            5 | Глобальный rate limit (requests per second) для API (token bucket)                     |
| `RATE_LIMIT_BURST`                           |      int |                           10 | Burst (ёмкость “ведра”), сколько запросов можно пропустить сразу                       |
| `BULKHEAD_MAX_PARALLEL`                      |      int |                            1 | Bulkhead (concurrency limit): максимум параллельных запросов (должно быть > 0)         |
| `OUTBOUND_PROFILE_BASE_URL`                  |   string |      `http://localhost:8090` | Base URL профайл-сервиса (upstream)                                                    |
| `OUTBOUND_PROFILE_TIMEOUT`                   | duration |                         `1s` | Таймаут (budget) на один вызов профайл-сервиса                                         |
| `OUTBOUND_TRANSPORT_IDLE_CONN_TIMEOUT`       | duration |                        `90s` | `http.Transport.IdleConnTimeout` для outbound                                          |
| `OUTBOUND_TRANSPORT_MAX_IDLE_CONNS`          |      int |                       `1000` | `http.Transport.MaxIdleConns`                                                          |
| `OUTBOUND_TRANSPORT_MAX_IDLE_CONNS_PER_HOST` |      int |                       `1000` | `http.Transport.MaxIdleConnsPerHost`                                                   |
| `OUTBOUND_TRANSPORT_MAX_CONNS_PER_HOST`      |      int |                          `0` | `http.Transport.MaxConnsPerHost` (0 = без лимита)                                      |
| `OUTBOUND_TRANSPORT_TLS_HANDSHAKE_TIMEOUT`   | duration |                         `5s` | `http.Transport.TLSHandshakeTimeout`                                                   |
| `OUTBOUND_TRANSPORT_RESPONSE_HEADER_TIMEOUT` | duration |                         `1s` | `http.Transport.ResponseHeaderTimeout`                                                 |
| `OUTBOUND_RETRY_MAX_ATTEMPTS`                |      int |                          `1` | Кол-во попыток outbound (включая первую)                                               |
| `OUTBOUND_RETRY_BASE_DELAY`                  | duration |                       `50ms` | Base delay для backoff (перед 2-й попыткой)                                            |
| `OUTBOUND_RETRY_MAX_DELAY`                   | duration |                      `500ms` | Max delay (cap) для backoff                                                            |

Пример (PowerShell):

```powershell
$env:HTTP_ADDR=":8080"
$env:HTTP_DEBUG="true"
go run .\cmd\api
```

Пример (bash):

```bash
HTTP_ADDR=":8080" HTTP_DEBUG=true go run ./cmd/api
```

## Endpoints

### API v1

- `GET /api/v1/users`
- `POST /api/v1/users`
- `GET /api/v1/users/{id}`

**v1 contract:** поле **`name`**. Поле **`full_name`** в v1 отклоняется.

- `GET  /api/v1/users/{id}/profile`

`GET /api/v1/users/{id}/profile` возвращает пользователя вместе с профилем.
Внутри делает outbound-запрос к профайл-сервису и поддерживает content negotiation (JSON/Protobuf).

### API v2

- `GET /api/v2/users`
- `POST /api/v2/users`

**v2 contract:** поле **`full_name`**. Поле **`name`** в v2 отклоняется.

### Jobs (async)

- `GET /api/v1/jobs/{id}`

Async режим включается query-параметром `async=1`:

- `POST /api/v1/users?async=1` → `202 Accepted` + `Location: /api/v1/jobs/{id}`
- `POST /api/v2/users?async=1` → `202 Accepted` + `Location: /api/v1/jobs/{id}`

Статусы job: `queued`, `running`, `succeeded`, `failed`.

Очередь bounded; политика переполнения: **fast-fail** → `503 Service Unavailable` (Problem+JSON).

### Health

- `GET /livez` — liveness (процесс жив) → `200`
- `GET /readyz` — readiness → `200` или `503`

Ответы health — простой JSON:

```json
{
  "status": "ok",
  "ts": "2026-01-18T19:03:00.123Z"
}
```

Readiness:

- fail-fast `503`, если сервис помечен как not-ready (lifecycle)
- далее выполняются checks (сейчас: `repo.Ping`, `workerpool`) с дедлайном **200ms**

### Debug (только при `HTTP_DEBUG=true`)

- `GET /debug/vars` — expvar (в т.ч. метрики HTTP)
- `GET /debug/pprof/` и подпути — pprof

Если `HTTP_DEBUG=false`, `/debug/*` вернёт `404`.

## Request ID

- Входящий `X-Request-Id` принимается **только** если валидный (ASCII 0x21..0x7e, без пробелов/control chars, длина ≤
  128).
- Иначе генерируется новый.
- `X-Request-Id` всегда добавляется в **response header**.
- Для ошибок (Problem+JSON) request-id дублируется в теле как `request_id`.

## Метрики

Метрики экспортируются через **expvar** на `/debug/vars` (если включён debug).

Ключи (HTTP):

- `http_in_flight`
- `http_requests_total` (ключ: `METHOD|/pattern|status`)
- `http_errors_total` (ключ: `METHOD|/pattern`, только `>=500`)
- `http_latency_ns_sum` (ключ: `METHOD|/pattern`)
- `http_latency_ns_count` (ключ: `METHOD|/pattern`)

Ключи (Step 4):

- `queue_depth` (текущая глубина очереди)
- `queue_rejections_total` (сколько раз enqueue был отклонён — очередь переполнена)
- `jobs_total` (счётчики по статусам jobs: `queued`, `running`, `succeeded`, `failed`)
- `job_processing_latency_ns_sum` (сумма длительности обработки jobs в наносекундах)
- `job_processing_latency_ns_count` (кол-во обработанных jobs)
- `rate_limited_total` (сколько запросов отклонено rate limiter’ом — `429`)
- `bulkhead_in_flight` (текущие in-flight под bulkhead)
- `bulkhead_rejections_total` (сколько запросов отклонено bulkhead’ом — `503`)

Пример фрагмента `/debug/vars`:

```json
{
  "http_requests_total": {
    "GET|/api/v1/users/{id}|200": 1
  },
  "http_latency_ns_sum": {
    "GET|/api/v1/users/{id}": 567500
  },
  "http_latency_ns_count": {
    "GET|/api/v1/users/{id}": 1
  }
}
```

### Outbound метрики (Step 5)

Ключи в `/debug/vars`:

- `outbound_requests_total` (ключ: `host|route|status_class`)
- `outbound_latency_ns_sum` (ключ: `host|route`)
- `outbound_latency_ns_count` (ключ: `host|route`)
- `outbound_errors_total` (ключ: `kind` — `timeout|canceled|5xx|4xx|parse|network|bad_response`)

## Graceful shutdown

- При `SIGINT` / `SIGTERM` сервис:
    1) помечает readiness как not-ready
    2) прекращает принимать новые async задания в очередь
    3) вызывает `Server.Shutdown()` с дедлайном `HTTP_SHUTDOWN_TIMEOUT`
    4) если дедлайн истёк → делает fallback `Server.Close()`
    5) после остановки HTTP-сервера останавливается worker pool (без утечек горутин)

## Примеры запросов (curl)

> На Windows лучше явно использовать `curl.exe`, чтобы не попасть на alias PowerShell.

### Health

```powershell
curl.exe -i http://localhost:8080/livez
curl.exe -i http://localhost:8080/readyz
```

### Создать пользователя (v1)

```powershell
curl.exe -i -X POST http://localhost:8080/api/v1/users -H "Content-Type: application/json" -d "{\"name\":\"Test User\",\"email\":\"test@email.com\"}"
```

### Получить пользователя по id

1) получаем ETag
```powershell
curl.exe -i -H "Accept: application/json" http://localhost:8080/api/v1/users/1
```

2) повторяем запрос с If-None-Match (подставь ETag из шага 1)
```powershell
curl.exe -i -H "Accept: application/json" -H "If-None-Match: W/\"u:1:v:1\"" http://localhost:8080/api/v1/users/1
```

3) Получить пользователя с профилем (outbound)
```powershell
curl.exe -i -H "Accept: application/json" http://localhost:8080/api/v1/users/1/profile
```

### Создать пользователя (v2)

```powershell
curl.exe -i -X POST http://localhost:8080/api/v2/users -H "Content-Type: application/json" -d "{\"full_name\":\"Test User\",\"email\":\"test@email.com\"}"
```

### Ошибка JSON / контрактов

```powershell
# 415 (нет Content-Type)
curl.exe -i -X POST http://localhost:8080/api/v1/users -d "{bad json"

# 400 (malformed JSON)
curl.exe -i -X POST http://localhost:8080/api/v1/users -H "Content-Type: application/json" -d "{bad json"
```

### Debug (после запуска с HTTP_DEBUG=true)

```powershell
curl.exe -i http://localhost:8080/debug/vars
curl.exe -i http://localhost:8080/debug/pprof/
```

### Создать пользователя (v1, async job)

```powershell
curl.exe -i -X POST "http://localhost:8080/api/v1/users?async=1" `
-H "Content-Type: application/json" `
-d "{\"name\":\"Bob\",\"email\":\"bob@example.com\",\"age\":21}"
```

### Проверить статус job # job id берём из заголовка Location: /api/v1/jobs/{id}

```powershell
curl.exe -i http://localhost:8080/api/v1/jobs/1
```

## Поддержка Protobuf (application/protobuf)

Этот проект умеет отдавать ответы в формате **Protobuf** (в дополнение к JSON) через **content negotiation**:

- JSON (по умолчанию): `Accept: application/json`
- Protobuf: `Accept: application/protobuf` (также принимается `application/x-protobuf`)

> Ошибки всегда возвращаются в формате **Problem+JSON** (`application/problem+json`) и не зависят от `Accept`.

## Генерация `*.pb.go` из `.proto` на Windows

### 1) Установить `protoc`

**Вариант A — winget (рекомендуется):**

```powershell
winget install protobuf
protoc --version
```

**Вариант B — ручная установка (если winget недоступен):**

1. Скачай Windows-архив `protoc-*-win64.zip` из официальных релизов protobuf.
2. Распакуй его, например в `C:\tools\protoc\`.
3. Добавь `C:\tools\protoc\bin` в `PATH`.
4. Проверь:

```powershell
protoc --version
```

### 2) Установить Go-плагин (`protoc-gen-go`)

```powershell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

Убедись, что директория с Go-бинарниками находится в `PATH`, иначе `protoc` не найдёт `protoc-gen-go`:

- `%GOBIN%`, если задан, иначе `%GOPATH%\bin`

Проверка:

```powershell
protoc-gen-go --version
```

### 3) Сгенерировать Go-код

Команду запускать из корня репозитория (там, где лежит `go.mod`):

```powershell
protoc -I . --go_out=. --go_opt=paths=source_relative internal/transport/pb/user.proto
```

Что означают флаги:

- `-I .` задаёт корень импорта/поиска `.proto` файлов.
- `--go_out=.` записывает сгенерированные файлы в репозиторий.
- `--go_opt=paths=source_relative` кладёт `user.pb.go` рядом с `user.proto` (в той же относительной папке).

### 4) Подтянуть зависимости модуля

Если protobuf в репозитории подключаешь впервые, выполни:

```powershell
go get google.golang.org/protobuf@latest
go mod tidy
```

### 5) Быстрая проверка

```powershell
go test ./...
```

## Быстрая проверка работы через HTTP

Скачать protobuf-ответ (PowerShell):

```powershell
curl -H "Accept: application/protobuf" http://localhost:8080/api/v1/users/1 --output user.pb
```

Если получаешь `200 OK` и `Content-Type: application/protobuf`, значит negotiation для protobuf работает.

После любых изменений в `internal/transport/pb/*.proto` повторяй команду генерации из шага (3)
и коммить обновлённые `*.pb.go` (если вы храните generated-файлы в git).

## Кэширование (ETag) для GET /api/v1/users/{id}

`GET /api/v1/users/{id}` возвращает заголовки:

- `ETag: W/"u:<id>:v:<version>"` (weak validator)
- `Cache-Control: private, max-age=60`
- `Vary: Accept` (потому что ответ зависит от `Accept`)

Если клиент отправляет `If-None-Match` и ETag совпадает — сервер вернёт `304 Not Modified` **без тела**.

## Outbound: Profile service

Сервис `/api/v1/users/{id}/profile` обращается к upstream **profile service**:

- `GET {OUTBOUND_PROFILE_BASE_URL}/profiles/{user_id}` → JSON:
    ```json
  {"user_id":1,"bio":"...","city":"..."}
    ```

### Быстрый mock profile service для локальной проверки

По умолчанию `OUTBOUND_PROFILE_BASE_URL = http://localhost:8090`. Можно поднять простой mock в отдельном терминале.

**bash/zsh:**
```bash
cat > /tmp/profilemock.go <<'EOF'
package main

import (
  "fmt"
  "log"
  "net/http"
)

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("/profiles/", func(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Path[len("/profiles/"):]
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"user_id":%s,"bio":"hello","city":"NY"}`, id)
  })
  log.Println("profile mock listening on :8090")
  log.Fatal(http.ListenAndServe(":8090", mux))
}
EOF

go run /tmp/profilemock.go
```

```powershell
$code = @'
package main

import (
  "fmt"
  "log"
  "net/http"
)

func main() {
  mux := http.NewServeMux()
  mux.HandleFunc("/profiles/", func(w http.ResponseWriter, r *http.Request) {
    id := r.URL.Path[len("/profiles/"):]
    w.Header().Set("Content-Type", "application/json")
    fmt.Fprintf(w, `{"user_id":%s,"bio":"hello","city":"NY"}`, id)
  })
  log.Println("profile mock listening on :8090")
  log.Fatal(http.ListenAndServe(":8090", mux))
}
'@
$tmp = Join-Path $env:TEMP "profilemock.go"
Set-Content -Path $tmp -Value $code -Encoding UTF8
go run $tmp
```

## Проверка 
```powershell 
curl.exe -i -H "Accept: application/json" http://localhost:8080/api/v1/users/1/profile
```
