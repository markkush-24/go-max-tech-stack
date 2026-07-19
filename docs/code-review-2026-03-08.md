# Code Review: pet-study (updated 2026-03-09)

## Формат ревью

Ниже ревью строго по 9 частям из вашего baseline:
- что сделано;
- чего не хватает;
- что исправить, чтобы соответствовать требованиям.

Проверки, выполненные в кодовой базе:
- `go test ./...` — проходит;
- `go vet ./...` — падает на unreachable code;
- `go test -race ./...` — не запускается в текущем окружении (`CGO_ENABLED=1` required).
- `gofmt -l cmd internal` — расхождений формата не найдено.
- `govulncheck ./...` — утилита не установлена в окружении.
- `staticcheck ./...` — утилита не установлена в окружении.
- `golangci-lint run` — запускается, но бинарь собран на Go 1.23 и не поддерживает target `go 1.25`.

---

## 1) Простота и границы пакетов

### Что сделано
- Используется `internal/*` и package-oriented структура без тяжёлого framework-overhead.
- Зависимости читаются по import graph, а не через DI-контейнеры/магические registry.
- Есть разделение по доменам: `service`, `store`, `router`, `middleware`, `security`, `outbound`.

Доказательства:
- `cmd/api/main.go:1`
- `internal/service/repository.go:8`
- `internal/httpapi/jobs.go:5`

### Чего не хватает
- Конфликт нейминга: `internal/routes` объявлен как `package router`, при этом есть отдельный `internal/router`.
- Часть слоёв избыточно тонкая (pass-through без бизнес-логики), например `JobService`.

Доказательства:
- `internal/routes/users_handler.go:1`
- `internal/router/router.go:1`
- `internal/service/jobService.go:8`
- `internal/service/jobService.go:16`

### Что исправить
- Переименовать `internal/routes` в `package routes` (или объединить с `internal/router`).
- Упростить тонкие passthrough-слои там, где они не добавляют policy/validation/contract.

---

## 2) Context, cancellation, timeouts, lifecycle

### Что сделано
- Входной lifecycle задан через `signal.NotifyContext`.
- Сервер поднимается через `http.Server` с таймаутами (`ReadHeaderTimeout`, `WriteTimeout`, `IdleTimeout`) и `Shutdown`.
- `ctx` прокинут через handler/service/repository сигнатуры.

Доказательства:
- `cmd/api/main.go:37`
- `internal/api/server.go:42`
- `internal/api/server.go:44`
- `internal/api/server.go:45`
- `internal/api/server.go:85`
- `internal/service/service.go:16`
- `internal/store/userrepo/memory_user_repo.go:30`

### Чего не хватает
- Worker pool живёт на `context.Background()` внутри себя (lifecycle отвязан от root-context по конструкции).
- `context.Context` хранится в `WorkerPool` struct (для request-path это anti-pattern; для lifecycle-компонентов допустимо, но контракт должен быть явно описан).
- Shutdown-цепочка не гарантирует явную стратегию для уже queued задач (drain или explicit cancel status).
- В in-memory репозиториях `ctx` почти не используется (кроме `Ping`), что не проверяет дисциплину для I/O слоя.

Доказательства:
- `internal/workerpool/workerpool.go:62`
- `internal/workerpool/workerpool.go:28`
- `internal/api/server.go:64`
- `internal/queue/queue.go:26`
- `internal/queue/queue.go:61`
- `internal/store/userrepo/memory_user_repo.go:26`

### Что исправить
- Привязать pool lifecycle к root-context приложения (или явно документировать текущий контракт).
- Зафиксировать shutdown semantics для async jobs: `drain-with-deadline` или `cancel-queued -> failed`.
- Для будущего DB-слоя добавить тесты на propagation timeout/cancel.

---

## 3) Ошибки как часть API

### Что сделано
- Есть централизованный mapping ошибок в HTTP Problem.
- Активно используются `errors.Is/errors.As` и wrapping через `%w`.
- Нет регулярного control-flow через panic в бизнес-обработке запросов.

Доказательства:
- `internal/httputils/errmap.go:69`
- `internal/httputils/errmap.go:165`
- `internal/service/service.go:37`
- `internal/service/service.go:45`

### Чего не хватает
- Есть panic на этапе wiring/конфигурации middleware (не request-flow, но повышает blast radius ошибок инициализации).
- Recover не пишет stack trace, диагностическая ценность panic-логов ограничена.

Доказательства:
- `internal/middleware/authorization.go:27`
- `internal/middleware/rbac.go:28`
- `internal/middleware/middleware.go:77`

### Что исправить
- Заменить panic-конфигурацию на явные ошибки и fail-fast в `run()`.
- Добавить stack trace в `Recover` (`runtime/debug.Stack()`).

---

## 4) Concurrency и жизненный цикл горутин

### Что сделано
- Для конкурентного доступа используются `sync.RWMutex`.
- У worker pool есть `WaitGroup`, cancel и ожидание завершения при остановке.

Доказательства:
- `internal/store/userrepo/memory_user_repo.go:15`
- `internal/store/jobrepo/memory_job_repo.go:18`
- `internal/workerpool/workerpool.go:30`
- `internal/workerpool/workerpool.go:86`

### Чего не хватает
- TOCTOU race на уникальности email (`ExistsByEmail` отдельно от `Save`).
- В тестах встречается `time.Sleep`, что делает concurrency-сценарии хрупкими.
- Есть package-global mutable state для метрик/expvar (`once` + singleton), что ухудшает изоляцию тестов и multi-instance поведение.
- `go test -race` сейчас не встроен в регулярный pipeline.

Доказательства:
- `internal/service/service.go:35`
- `internal/service/service.go:45`
- `internal/store/userrepo/memory_user_repo.go:54`
- `internal/store/userrepo/memory_user_repo.go:87`
- `internal/middleware/async_metrics_test.go:93`
- `internal/middleware/metrics_test.go:54`
- `internal/metrics/http.go:33`
- `internal/queue/queue.go:50`
- `internal/middleware/bulkhead.go:12`

### Что исправить
- Сделать проверку уникальности атомарной на уровне репозитория/БД.
- Убрать `sleep` в тестах в пользу синхронизационных примитивов.
- Уйти от глобальных singleton-метрик к app-scoped registry.
- Запускать `go test -race` в окружении с `CGO_ENABLED=1`.

---

## 5) Тесты: уровни и покрытие

### Что сделано
- Есть хороший слой интеграционных тестов API/middleware/outbound.
- Есть unit-тесты для некоторых инфраструктурных компонентов (например, `userrepo`, `requestid`, `httputils`).

Доказательства:
- `internal/routes/security_http_test.go:1`
- `internal/routes/negotiation_test.go:1`
- `internal/routes/job_handler_test.go:1`
- `internal/outbound/profile_outbound_test.go:1`
- `internal/store/userrepo/memory_user_repo_test.go:1`
- `internal/requestid/requestid_test.go:1`

### Чего не хватает
- Нет отдельных тестов для `queue`, `workerpool`, `jobrepo`, `service`, `config`, `security` verifier.
- Нет fuzz-тестов для input-heavy участков (JSON/headers/parsers).
- Нет дисциплины integration coverage как отдельного сигнала.
- По пакетам заметный дисбаланс: 11 из 23 пакетов не имеют ни одного `*_test.go`.

Доказательства:
- `internal/queue/queue.go:1` (нет `*_test.go` в пакете)
- `internal/workerpool/workerpool.go:1` (нет `*_test.go` в пакете)
- `internal/store/jobrepo/memory_job_repo.go:1` (нет `*_test.go` в пакете)
- `internal/config/config.go:1` (нет `*_test.go` в пакете)
- `internal/security/jwt_verifier_hs256.go:1` (нет `*_test.go` в пакете)
- поиск по репозиторию `func Fuzz...` — отсутствует
- `go list ./...` + проверка `*_test.go`: без тестов, в т.ч. `internal/security`, `internal/service`, `internal/workerpool`, `internal/config`, `internal/queue`.

### Что исправить
- Добавить таргетированные unit/integration тесты на критичные пакеты.
- Добавить минимум 1-2 fuzz-suite для парсеров входов.
- Отдельно отслеживать покрытие ключевых границ (auth/config/async pipeline).

---

## 6) Tooling gates и CI

### Что сделано
- Базовая проверка `go test ./...` зелёная.
- Формат кода по `gofmt` соблюдается.

### Чего не хватает
- `go vet ./...` не зелёный (unreachable code).
- Нет видимой CI-конфигурации и обязательных quality gates.
- Нет автоматизации для `govulncheck`, `staticcheck`, `golangci-lint`.
- Часть tooling недоступна/неактуальна в окружении (что дополнительно мешает reproducible quality bar).

Доказательства:
- `internal/routes/users_handler.go:92`
- `internal/routes/users_handler_v2.go:91`
- в репозитории отсутствуют `.github/workflows`, `.gitlab-ci.yml`, `golangci` config
- `govulncheck ./...` -> command not found
- `staticcheck ./...` -> command not found
- `golangci-lint run` -> binary built with go1.23, target is go1.25

### Что исправить
- Убрать unreachable `return nil` в двух хендлерах.
- Включить обязательные пайплайн-проверки: `gofmt -w`/format check, `go vet`, `go test`, `go test -race`, `govulncheck`.
- Добавить стат-анализ (минимум `staticcheck`, лучше через `golangci-lint`).

---

## 7) Операбельность и дебаггability

### Что сделано
- Есть debug endpoints (`expvar`, `pprof`) и они включаются флагом.
- Debug-роут дополнительно завёрнут в auth/authz.
- Есть graceful shutdown и readiness/liveness контур.

Доказательства:
- `internal/router/debug.go:9`
- `internal/router/debug.go:12`
- `cmd/api/main.go:136`
- `cmd/api/main.go:137`
- `internal/api/server.go:85`

### Чего не хватает
- Логирование остаётся неструктурированным (`log.Printf`), нет `slog`/уровней/нормализованных полей.
- Readiness ставится `ready=true` до подтверждения успешного listen-bind.
- В panic-логах нет stack trace.

Доказательства:
- `cmd/api/main.go:164`
- `internal/httputils/apphandler.go:22`
- `internal/middleware/middleware.go:57`
- `internal/api/server.go:54`
- `internal/api/server.go:60`

### Что исправить
- Перейти на structured logging (`log/slog` или эквивалент).
- Перевести запуск на `net.Listen + Serve`, выставлять ready только после успешного bind.
- Добавить stack trace в recover-лог.

---

## 8) Module/dependency hygiene

### Что сделано
- `go.mod` компактный, без `replace`-хаоса.
- Версия Go зафиксирована (`go 1.25`).

Доказательства:
- `go.mod:1`
- `go.mod:3`

### Чего не хватает
- Нет `toolchain` directive для воспроизводимой среды.
- Нет формализованного процесса dependency security scanning (`govulncheck` в CI).
- `jwt/v5` используется напрямую в production-коде, но в `go.mod` помечен как `// indirect` (нужно привести к tidy-состоянию).

Доказательства:
- `go.mod:10`
- `internal/security/jwt_verifier_hs256.go:9`

### Что исправить
- Добавить `toolchain` directive (если команда фиксирует конкретный toolchain).
- Включить регулярный `go mod tidy` + `govulncheck` в pipeline.

---

## 9) Документация и примеры

### Что сделано
- Хороший README: конфигурация, endpoints, безопасность, graceful shutdown, примеры вызовов.

Доказательства:
- `README.md:28`
- `README.md:87`
- `README.md:192`
- `README.md:255`

### Чего не хватает
- Не используются package comments (`// Package ...`) для пакетов.
- Существенная часть exported API без doc comments.
- Не документированы concurrency guarantees и zero-value semantics для ключевых типов.

Доказательства:
- `internal/api/server.go:14` (`type APIServer struct` без doc comment)
- `internal/queue/queue.go:16` (`type Queue struct` без doc comment)
- `internal/workerpool/workerpool.go:19` (`type WorkerPool struct` без doc comment)
- `internal/outbound/profile/profile.go:3` (`type Profile struct` без doc comment)
- `internal/outbound/profile/profile_client.go:7` (`type Client interface` без doc comment)
- поиск `^// Package ` по `cmd/internal` (non-test) — совпадений нет
- автоматизированная проверка exported declarations: ~328 экспортируемых объявлений без doc comment

### Что исправить
- Добавить package comments для ключевых пакетов.
- Добавить doc comments на exported types/functions (минимум для boundary API).
- Задокументировать thread-safety/ownership/lifetime контракт для `Queue`, `WorkerPool`, метрик.

---

## Отдельный блок: Критичные cross-cutting риски

Это риски, которые бьют сразу по нескольким из 9 разделов:

1. TOCTOU race на email-уникальности.
Доказательства:
- `internal/service/service.go:35`
- `internal/store/userrepo/memory_user_repo.go:54`
- `internal/store/userrepo/memory_user_repo.go:87`

2. Неполная HTTP negotiation по `Accept` (`q`/`q=0` не учитываются).
Доказательства:
- `internal/httputils/accept.go:38`
- `internal/httputils/accept.go:46`
- `internal/httputils/accept.go:49`

3. Shutdown не фиксирует судьбу queued задач явным контрактом.
Доказательства:
- `internal/api/server.go:64`
- `internal/api/server.go:85`
- `internal/queue/queue.go:26`

---

## Отдельный блок: Вне baseline 9 пунктов, но важно для production

1. Scope-gap: нет персистентного хранилища (пока только in-memory).
Следствие: нет durability/ACID/индексов/миграций.
Доказательства:
- `cmd/api/main.go:50`
- `cmd/api/main.go:51`
- `internal/config/config.go:208`

2. Rate limit глобальный на процесс, не по клиенту (`IP/principal/API-key`).
Следствие: noisy-neighbor и слабая fairness при mixed traffic.
Доказательства:
- `cmd/api/main.go:83`
- `internal/middleware/rate_limiter.go:24`

---

## Что уже соответствует baseline на хорошем уровне

- Базовый HTTP lifecycle (timeouts + shutdown) реализован.
- Error mapping в Problem+JSON и typed-errors дисциплина в целом на месте.
- Есть заметный интеграционный тестовый слой и observability-контур (expvar/pprof/HTTP metrics).

## Что нужно закрыть в первую очередь

1. Убрать high-risk конкурентность/negotiation/shutdown gaps.
2. Привести quality gates к зелёному и автоматическому состоянию (`vet/race/vuln/lint`).
3. Усилить documentation contracts и убрать неоднозначности package boundaries/нейминга.
