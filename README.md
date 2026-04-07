# pet-study

HTTP-сервис на Go (net/http, Go 1.25). Ресурс: **User**. Есть версии API **v1** и **v2** (breaking change по полю
имени).

## Запуск

### Требования

- Go **1.25**
- Рекомендуемый patch level: **Go 1.25.8+**

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

### Локальный HTTPS

HTTPS listener опционален и использует тот же handler chain, что и обычный HTTP.

PowerShell:

```powershell
$env:HTTP_TLS_ENABLE="true"
$env:HTTP_TLS_ADDR=":8443"
$env:HTTP_TLS_CERT_FILE="C:\path\to\server.crt"
$env:HTTP_TLS_KEY_FILE="C:\path\to\server.key"
go run .\cmd\api
```

Проверка с self-signed сертификатом:

```powershell
curl.exe -k https://localhost:8443/livez
```


## Step 7 — Protocol + Streaming

### Как включить HTTPS

HTTPS listener включается отдельно от обычного HTTP listener и использует тот же handler chain.
Это позволяет сохранить dev-mode на `HTTP_ADDR`, а HTTPS поднять параллельно.

PowerShell:

```powershell
$env:HTTP_ADDR=":8080"
$env:HTTP_TLS_ENABLE="true"
$env:HTTP_TLS_ADDR=":8443"
$env:HTTP_TLS_CERT_FILE="C:\path\to\server.crt"
$env:HTTP_TLS_KEY_FILE="C:\path\to\server.key"
go run .\cmd\api
```

### Как проверить HTTP/2

HTTP/2 в этом проекте активируется через HTTPS listener (ALPN) и не требует отдельного хэндлера.
Проверка с self-signed сертификатом:

```powershell
curl.exe -k --http2 -i https://localhost:8443/livez
curl.exe -k --http2 -i https://localhost:8443/api/v1/users/1
```

Ожидание:
- запрос идёт по HTTPS;
- `curl` не откатывается на обычный HTTP/1.1;
- контракты обычного API не меняются.

### Как использовать SSE endpoint

Endpoint:
- `GET /api/v1/jobs/{id}/events`

Назначение:
- поток событий job (`queued`, `running`, `succeeded`, `failed`);
- heartbeat отправляется сервером периодически как SSE comment;
- доступ только для владельца job или `admin`.

Пример (PowerShell, с bearer token):

```powershell
curl.exe -N -i ^
  -H "Authorization: Bearer $AdminToken" ^
  http://localhost:8080/api/v1/jobs/1/events
```

Что важно:
- ответ использует `Content-Type: text/event-stream`;
- соединение остаётся открытым;
- для медленных клиентов используется bounded subscriber buffer + drop policy;
- обычные HTTP latency metrics для `/events` не искажаются отдельным долгоживущим stream.

### Как запустить и вызвать gRPC server

Внутренний gRPC transport запускается отдельно от HTTP/HTTPS.
Сервис: `pb.JobsService`.
Минимальный метод: `GetJob`.

PowerShell:

```powershell
$env:GRPC_ENABLE="true"
$env:GRPC_ADDR=":9090"
go run .\cmd\api
```

Быстрая проверка через `grpcurl`:

```powershell
grpcurl -plaintext localhost:9090 list
grpcurl -plaintext localhost:9090 describe pb.JobsService
'{"id":1}' | grpcurl.exe -plaintext -d '@' localhost:9090 pb.JobsService/GetJob
```

HTTP → gRPC bridge demo endpoint:
- `GET /api/v1/jobs/{id}/grpc`
- доступ: `admin` only
- ответ: compact HTTP DTO c источником `"grpc"`

Пример:

```powershell
curl.exe -i ^
  -H "Authorization: Bearer $AdminToken" ^
  http://localhost:8080/api/v1/jobs/1/grpc
```

### Как сгенерировать proto / gRPC код

В проекте есть два proto-файла:
- `internal/transport/pb/user.proto`
- `internal/transport/pb/job.proto`

Нужны оба плагина:

```powershell
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

Генерация из корня репозитория:

```powershell
protoc -I . --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative internal/transport/pb/user.proto internal/transport/pb/job.proto
```

После регенерации:

```powershell
go test ./...
```

### Как использовать Range export endpoint

Endpoint:
- `GET /api/v1/users/{id}/export`

Назначение:
- экспорт пользователя как JSON-файл;
- доступ только для владельца пользователя или `admin`;
- поддерживается partial download через `Range`.

Полный ответ:

```powershell
curl.exe -i ^
  -H "Authorization: Bearer $AdminToken" ^
  http://localhost:8080/api/v1/users/1/export
```

Partial download:

```powershell
curl.exe -i ^
  -H "Authorization: Bearer $AdminToken" ^
  -H "Range: bytes=0-9" ^
  http://localhost:8080/api/v1/users/1/export
```

Ожидание:
- полный ответ -> `200 OK`;
- partial -> `206 Partial Content`;
- присутствует `Content-Range`;
- используется `http.ServeContent`, а не кастомный manual parser.

### Инварианты Step 7 не сломаны

После добавления TLS, gRPC, SSE и Range в проекте сохранены базовые ограничения:

- `ServeMux` не обходится, поэтому `r.Pattern` и `PathValue()` продолжают работать для RBAC и метрик;
- `X-Request-Id` остаётся trust-boundary controlled: невалидный входной header не принимается, а response/problem body остаются консистентными;
- CORS применяется только к API subtree, preflight short-circuit остаётся до auth;
- `/debug/*` монтируется только при `HTTP_DEBUG=true` и не попадает в обычные HTTP metrics;
- таймауты обычного API (`HTTP_*`) не были заменены стриминговой логикой; для SSE используется отдельная streaming policy;
- shutdown/readiness учитывают worker pool, gRPC runtime и stream hub;
- manual smoke и Step 13 autotests закрывают SSE, gRPC и Range сценарии.

## Tooling

Для локальных quality checks в проекте зафиксированы версии:
- `govulncheck`
- `staticcheck`

Установка на Windows PowerShell:

```powershell
.\scripts\install-tools.ps1
```

Проверка:

```powershell
.\scripts\check-format.ps1
.\scripts\run-staticcheck.ps1
.\scripts\run-race.ps1
.\scripts\run-govulncheck.ps1
```

Полный локальный прогон:

```powershell
.\scripts\run-checks.ps1
```

CI:

- GitHub Actions workflow: `.github/workflows/ci.yml`
- Проверки в CI: `gofmt`, `go test`, `go vet`, `go test -race`, `staticcheck`, `govulncheck -mode binary`
- Vulnerability scan собирает binary на patched toolchain `go1.25.8`

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
| `HTTP_TLS_ENABLE`                            |     bool |                      `false` | Включить дополнительный HTTPS listener                                                 |
| `HTTP_TLS_ADDR`                              |   string |                      `:8443` | Адрес HTTPS listener                                                                   |
| `HTTP_TLS_CERT_FILE`                         |   string |                    *(пусто)* | Путь к PEM-сертификату сервера; обязателен при `HTTP_TLS_ENABLE=true`                  |
| `HTTP_TLS_KEY_FILE`                          |   string |                    *(пусто)* | Путь к PEM-ключу сервера; обязателен при `HTTP_TLS_ENABLE=true`                        |
| `GRPC_ENABLE`                                |     bool |                      `false` | Включить внутренний gRPC server                                                         |
| `GRPC_ADDR`                                  |   string |                      `:9090` | Адрес gRPC listener; обязателен при `GRPC_ENABLE=true`                                  |
| `STREAMING_SSE_HEARTBEAT`                    | duration |                        `15s` | Интервал heartbeat для SSE (`: heartbeat`)                                              |
| `STREAMING_SUBSCRIBER_BUFFER`                |      int |                           16 | Размер bounded buffer на одного SSE subscriber                                          |
| `STREAMING_WRITE_TIMEOUT`                    | duration |                        `10s` | Дедлайн записи одного SSE flush/write                                                   |
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
Shutdown semantics для async jobs: **fail-fast**. На остановке сервис перестаёт принимать новые job, а все job в состояниях
`queued` и `running` переводятся в `failed` с причиной `job canceled: server shutting down`.
Worker pool хранит свой внутренний lifecycle-context только для времени жизни воркеров; это не request-context, а
производный context приложения, передаваемый из `main`.

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

- `ready=true` выставляется только после успешного `listen/bind`
- для этого сервер стартует через `net.Listen + Serve`: `ListenAndServe()` скрывает фазу bind внутри себя, поэтому при
  старом запуске readiness мог стать `true` ещё до подтверждения, что порт действительно занят процессом
- fail-fast `503`, если сервис помечен как not-ready (lifecycle)
- далее выполняются checks (сейчас: `repo.Ping`, `workerpool`) с дедлайном **200ms**

Примечание про in-memory хранилища:

- in-memory репозитории почти не используют `ctx`, потому что в них нет реального блокирующего I/O
- это нормально для учебной/in-memory реализации, но не считается заменой корректной cancel/timeout дисциплины для будущего DB-слоя

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
    3) закрывает stream hub, чтобы SSE subscribers корректно завершились
    4) вызывает `Server.Shutdown()` / `ServeTLS` shutdown с дедлайном `HTTP_SHUTDOWN_TIMEOUT`
    5) останавливает gRPC runtime через graceful shutdown с timeout fallback
    6) после остановки HTTP/gRPC отменяет worker pool
    7) все оставшиеся `queued` / `running` jobs переводятся в `failed` с причиной `job canceled: server shutting down`
    8) если дедлайн истёк → делает fallback `Server.Close()` / принудительный gRPC stop

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

### SSE stream по job

```powershell
curl.exe -N -i ^
  -H "Authorization: Bearer $AdminToken" ^
  http://localhost:8080/api/v1/jobs/1/events
```

### HTTP -> gRPC bridge

```powershell
curl.exe -i ^
  -H "Authorization: Bearer $AdminToken" ^
  http://localhost:8080/api/v1/jobs/1/grpc
```

### Range export

```powershell
curl.exe -i ^
  -H "Authorization: Bearer $AdminToken" ^
  http://localhost:8080/api/v1/users/1/export

curl.exe -i ^
  -H "Authorization: Bearer $AdminToken" ^
  -H "Range: bytes=0-9" ^
  http://localhost:8080/api/v1/users/1/export
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

## Step 6 — Security layer

На этом шаге в сервис добавлены:

- JWT authentication (`Authorization: Bearer <token>`)
- RBAC authorization (`admin` / `user`)
- resource-level authorization (`user` может читать только себя)
- CORS (deny-by-default, preflight support)
- security headers
- trust proxy (`X-Forwarded-For`, `X-Forwarded-Proto`, `X-Request-Id` только от trusted proxy)

### 1. JWT для локальной проверки

Для локальной разработки используются dev-ключи JWT.  
Если не переопределять конфиг, можно использовать следующие токены.

#### Admin token
```text
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImRldiJ9.eyJzdWIiOiIxIiwicm9sZSI6ImFkbWluIiwiZXhwIjoxODkzNDU2MDAwfQ.vUO8q4ilMk36iOimRFvdt_EWo5TXffr7Q3MIOMU3vIU
```

#### User token
```text
eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImRldiJ9.eyJzdWIiOiIxIiwicm9sZSI6InVzZXIiLCJleHAiOjE4OTM0NTYwMDB9.088yBOvdyuqkiHsBjQCETIUEZHH6U4O_RVV4nduojVc
```

Пример переменных для PowerShell:

```powershell
$Base = "http://localhost:8080"

$AdminToken = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImRldiJ9.eyJzdWIiOiIxIiwicm9sZSI6ImFkbWluIiwiZXhwIjoxODkzNDU2MDAwfQ.vUO8q4ilMk36iOimRFvdt_EWo5TXffr7Q3MIOMU3vIU"
$UserToken  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCIsImtpZCI6ImRldiJ9.eyJzdWIiOiIxIiwicm9sZSI6InVzZXIiLCJleHAiOjE4OTM0NTYwMDB9.088yBOvdyuqkiHsBjQCETIUEZHH6U4O_RVV4nduojVc"
```

### 2. AuthN / AuthZ поведение

Поддерживается Bearer JWT через заголовок:

```text
Authorization: Bearer <jwt>
```

Семантика ответов:

- `401 Unauthorized` — токен отсутствует / невалиден / истёк
- `403 Forbidden` — токен валиден, но прав недостаточно

Текущее поведение:

- `GET /api/v1/users/{id}` — admin может читать любого, user только себя
- `GET /api/v2/users/{id}` — admin может читать любого, user только себя
- `GET /api/v1/jobs/{id}` — admin only
- `GET /debug/*` — только при `HTTP_DEBUG=true`, и только admin
- `/livez`, `/readyz` — без auth

### 3. CORS

CORS реализован как deny-by-default для запросов с `Origin`.

Поддерживается:

- allowlist origins
- preflight `OPTIONS`
- `Access-Control-Allow-Methods`
- `Access-Control-Allow-Headers`
- `Access-Control-Max-Age`
- корректный `Vary`
- запрет комбинации `Allow-Credentials=true` и `Access-Control-Allow-Origin=*`

Пример для PowerShell перед запуском сервера:

```powershell
$env:CORS_ALLOWED_ORIGINS = "http://localhost:3000"
$env:CORS_ALLOW_CREDENTIALS = "false"
go run .\cmd\api
```

### 4. Security headers

На API-ответы выставляются:

- `X-Content-Type-Options: nosniff`
- `Referrer-Policy: no-referrer`
- `X-Frame-Options: DENY`

HSTS (`Strict-Transport-Security`) выставляется только если запрос считается HTTPS:
- напрямую через `r.TLS`
- либо через trusted proxy и `X-Forwarded-Proto=https`

### 5. Trusted proxies

Forwarded-заголовки учитываются только от trusted proxy.

Используются:

- `X-Forwarded-For` → effective client IP
- `X-Forwarded-Proto` → effective scheme (`http` / `https`)
- `X-Request-Id` → принимается только от trusted proxy, иначе входящий header санитайзится

Пример настройки для локальной проверки:

```powershell
$env:PROXY_TRUSTED_PROXIES = "127.0.0.1/32"
$env:PROXY_TRUST_XFF = "true"
$env:PROXY_TRUST_XFP = "true"
go run .\cmd\api
```

### 6. Примеры curl (PowerShell)

#### 6.1 Нет токена → 401
```powershell
curl.exe -i "$Base/api/v1/users/1"
```

#### 6.2 Admin создаёт пользователя
```powershell
curl.exe -i `
  -H "Authorization: Bearer $AdminToken" `
  -H "Content-Type: application/json" `
  -d '{"name":"Bob","email":"bob@example.com","age":21}' `
  "$Base/api/v1/users"
```

#### 6.3 User читает себя → 200
```powershell
curl.exe -i `
  -H "Authorization: Bearer $UserToken" `
  "$Base/api/v1/users/1"
```

#### 6.4 User читает чужого → 403
```powershell
curl.exe -i `
  -H "Authorization: Bearer $UserToken" `
  "$Base/api/v1/users/2"
```

#### 6.5 Debug без токена → 401
```powershell
curl.exe -i "$Base/debug/vars"
```

#### 6.6 Debug с user token → 403
```powershell
curl.exe -i `
  -H "Authorization: Bearer $UserToken" `
  "$Base/debug/vars"
```

#### 6.7 Debug с admin token → 200
```powershell
curl.exe -i `
  -H "Authorization: Bearer $AdminToken" `
  "$Base/debug/vars"
```

#### 6.8 CORS preflight → 204
```powershell
curl.exe -i -X OPTIONS "$Base/api/v1/users" `
  -H "Origin: http://localhost:3000" `
  -H "Access-Control-Request-Method: POST" `
  -H "Access-Control-Request-Headers: authorization, content-type"
```

#### 6.9 Health endpoints без auth
```powershell
curl.exe -i "$Base/livez"
curl.exe -i "$Base/readyz"
```

### 7. ETag / If-None-Match

Для `GET /api/v1/users/{id}` поддерживаются `ETag` и `If-None-Match`.

Получить `ETag`:

```powershell
$headers = curl.exe -s -D - -o NUL `
  -H "Authorization: Bearer $UserToken" `
  "$Base/api/v1/users/1"

$etag = ($headers | Select-String -Pattern '^(?i)etag:\s*' | Select-Object -First 1).Line `
  -replace '^(?i)etag:\s*',''
$etag = $etag.Trim()
$etag
```

Проверить `304 Not Modified`:

```powershell
curl.exe -i `
  -H "Authorization: Bearer $UserToken" `
  -H "If-None-Match: $etag" `
  "$Base/api/v1/users/1"
```

### 8. Канонический порядок middleware

Фактический порядок выполнения глобальной цепочки (outer -> inner):

1. `Proxy.SanitizeRequestIDHeader`
2. `RequestIDMiddleware`
3. `Recover (outer)`
4. `TrustProxy`
5. `Metrics`
6. `Logger`
7. `Recover (inner)`
8. `RootRouter`

Для API subtree (outer -> inner):

1. `SecurityHeaders`
2. `CORS`
3. `ServeMux router`
4. `Bulkhead`
5. `RateLimiter`
6. `AuthN`
7. `RBAC`
8. `Handler`

Инварианты:

- preflight `OPTIONS` не требует auth
- `/livez`, `/readyz` — без auth
- `/debug/*` — только при `HTTP_DEBUG=true` и только для admin
- `/debug/*` не учитывается в http metrics
- основной путь матчинга остаётся через `ServeMux.ServeHTTP`, `r.Pattern` не ломается

### 9. Security expvar metrics

Добавлены метрики:

- `authn_failures_total`
- `authz_forbidden_total`
- `cors_preflight_total`
- `cors_denied_total`

### 10. Проверка после изменений

Базовая команда:

```powershell
go test ./...
```

Если всё настроено корректно, тесты Step 6 должны проходить полностью.
