# pet-study

HTTP-сервис на Go (net/http, Go 1.25). Ресурс: **User**. Есть версии API **v1** и **v2** (breaking change по полю имени).

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

| Переменная                 |      Тип |                       Дефолт | Описание                                                                                |
|----------------------------|---------:|-----------------------------:|-----------------------------------------------------------------------------------------|
| `HTTP_ADDR`                |   string |                      `:8080` | Адрес для `http.Server.Addr`                                                            |
| `HTTP_READ_HEADER_TIMEOUT` | duration |                         `5s` | Таймаут чтения заголовков (slowloris mitigation)                                        |
| `HTTP_READ_TIMEOUT`        | duration |                        `10s` | Таймаут чтения запроса целиком                                                          |
| `HTTP_WRITE_TIMEOUT`       | duration |                        `15s` | Таймаут записи ответа                                                                   |
| `HTTP_IDLE_TIMEOUT`        | duration |                       `120s` | Keep-alive idle timeout                                                                 |
| `HTTP_SHUTDOWN_TIMEOUT`    | duration |                        `10s` | Дедлайн на graceful shutdown                                                            |
| `HTTP_MAX_HEADER_BYTES`    |      int | `http.DefaultMaxHeaderBytes` | Лимит размера заголовков                                                                |
| `HTTP_DEBUG`               |     bool |                      `false` | Включить debug endpoints (`/debug/...`)                                                 |
| `DB_DSN`                   |   string |                    *(пусто)* | Опционально (в Step 3 используется in-memory repo). Если задана — должна быть непустой  |
| `WORKERS_COUNT`            |      int |                           10 | Кол-во воркеров worker pool для обработки async jobs (должно быть > 0)                  |
| `RATE_LIMIT_RPS`           |      int |                            5 | Глобальный rate limit (requests per second) для API (token bucket)                      |
| `RATE_LIMIT_BURST`         |      int |                           10 | Burst (ёмкость “ведра”), сколько запросов можно пропустить сразу                        |

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
- `GET  /api/v1/users`
- `POST /api/v1/users`
- `GET  /api/v1/users/{id}`

**v1 contract:** поле **`name`**. Поле **`full_name`** в v1 отклоняется.

### API v2
- `GET  /api/v2/users`
- `POST /api/v2/users`

**v2 contract:** поле **`full_name`**. Поле **`name`** в v2 отклоняется.

### Health
- `GET /livez` — liveness (процесс жив) → `200`
- `GET /readyz` — readiness → `200` или `503`

Ответы health — простой JSON:
```json
{"status":"ok","ts":"2026-01-18T19:03:00.123Z"}
```

Readiness:
- fail-fast `503`, если сервис помечен как not-ready (lifecycle)
- далее выполняются checks (сейчас: `repo.Ping`) с дедлайном **200ms**

### Debug (только при `HTTP_DEBUG=true`)
- `GET /debug/vars` — expvar (в т.ч. метрики HTTP)
- `GET /debug/pprof/` и подпути — pprof

Если `HTTP_DEBUG=false`, `/debug/*` вернёт `404`.

## Request ID

- Входящий `X-Request-Id` принимается **только** если валидный (ASCII 0x21..0x7e, без пробелов/control chars, длина ≤ 128).
- Иначе генерируется новый.
- `X-Request-Id` всегда добавляется в **response header**.
- Для ошибок (Problem+JSON) request-id дублируется в теле как `request_id`.

## Метрики

Метрики экспортируются через **expvar** на `/debug/vars` (если включён debug).

Ключи:
- `http_in_flight`
- `http_requests_total` (ключ: `METHOD|/pattern|status`)
- `http_errors_total` (ключ: `METHOD|/pattern`, только `>=500`)
- `http_latency_ns_sum` (ключ: `METHOD|/pattern`)
- `http_latency_ns_count` (ключ: `METHOD|/pattern`)

Пример фрагмента `/debug/vars`:
```json
{
  "http_requests_total": {"GET|/api/v1/users/{id}|200": 1},
  "http_latency_ns_sum": {"GET|/api/v1/users/{id}": 567500},
  "http_latency_ns_count": {"GET|/api/v1/users/{id}": 1}
}
```

## Graceful shutdown

- При `SIGINT` / `SIGTERM` сервис:
  1) помечает readiness как not-ready
  2) вызывает `Server.Shutdown()` с дедлайном `HTTP_SHUTDOWN_TIMEOUT`
  3) если дедлайн истёк → делает fallback `Server.Close()`

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
```powershell
curl.exe -i http://localhost:8080/api/v1/users/1
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
