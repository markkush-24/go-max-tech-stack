# Step 7 — Спека по протоколам и стримингу

## Статус

Черновик для backlog-пункта **S7-B1**.

## Цель

Расширить существующий сервис `pet-study` следующими возможностями:

- опциональный HTTPS listener для существующего HTTP API;
- отдельный gRPC server как внутренний транспорт;
- SSE для live-событий по job;
- Range support для частичной загрузки ресурса;

при этом **не ломая инварианты Step 4–6**:

- нельзя обходить `http.ServeMux` так, чтобы ломались `r.Pattern` / `PathValue`;
- семантика `request-id` должна оставаться консистентной;
- CORS остаётся только на API subtree, preflight short-circuit должен происходить до auth;
- `/debug/*` остаётся debug-only и исключённым из обычных HTTP metrics;
- обычные API timeout’ы и поведение Step 4–6 не должны быть случайно сломаны.

## Транспортная схема

Сервис будет поддерживать три типа listener’ов.

### 1. HTTP

Существующий REST API продолжает работать в обычном HTTP-режиме для local/dev сценариев.

Пример:
- `HTTP_ADDR=:8080`

### 2. HTTPS

Опциональный второй listener с **тем же handler chain** и **теми же маршрутами**, что и у HTTP, но поверх TLS.

Пример:
- `HTTPS_ADDR=:8443`

Примечания:
- HTTPS **не** вводит отдельные endpoint’ы; это тот же API, но поверх TLS.
- В обычном TLS server предъявляет свой сертификат клиенту.
- На HTTPS listener ожидается нормальная работа HTTP/2 через стандартную поддержку Go `net/http` для TLS/HTTP2.

### 3. gRPC

Опциональный отдельный gRPC server на собственном адресе.

Пример:
- `GRPC_ADDR=:9090`

Примечания:
- gRPC — это отдельный server, а не замена HTTP API.
- Он использует тот же domain/service слой, что и HTTP handlers.
- В рамках Step 7 он рассматривается как внутренний транспортный интерфейс.

## Зафиксированный scope Step 7

### Обязательный streaming-механизм

**SSE** на endpoint’е:

`GET /api/v1/jobs/{id}/events`

Назначение:
- стримить клиенту события жизненного цикла job:
    - `queued`
    - `running`
    - `progress` (опционально)
    - `succeeded`
    - `failed`

SSE выбран потому, что проекту нужен **one-way поток от сервера к клиенту** для async jobs.

### gRPC сервис

Выбран сервис:

**JobsService**

Начальный scope:
- `GetJob(id)` — обязательно
- `WatchJob(id)` — опционально, если останется время

Причина выбора:
- jobs уже существуют в проекте;
- уже есть async processing и статусы job;
- SSE и возможный gRPC streaming естественно ложатся на job domain.

### Вторичная протокольная часть

Выбран механизм:

**Range support**

Планируемый endpoint:
- `GET /api/v1/users/{id}/export`

Назначение:
- поддержка частичной загрузки большого экспортируемого ресурса.

Направление реализации:
- предпочтительно использовать `http.ServeContent`, так как он корректно обрабатывает Range requests и связанные conditional headers.

## Зафиксированные политики

## Политика для SSE

### Auth

Доступ к `GET /api/v1/jobs/{id}/events`:
- владелец job;
- либо admin.

Должна использоваться существующая authn/authz модель из Step 6.

### Формат стрима

Заголовки:
- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`

Опционально:
- `Connection: keep-alive` для совместимости с HTTP/1.1

### Flush

Handler обязан проверять поддержку `http.Flusher`, так как корректность стриминга зависит от явного `Flush()` после записи события и heartbeat.

### Heartbeat

Фиксированный heartbeat interval:
- `15s`

Формат heartbeat:
- SSE comment frame, например `: ping`

Причина:
- поддержание long-lived соединения живым для proxy/intermediary.

### Cancellation

Цикл SSE должен завершаться по:
- `r.Context().Done()`
- отмене подписки
- shutdown event hub

### Backpressure

Политика на подписчика:
- bounded subscriber buffer;
- начальный размер: `16`

Политика публикации:
- publish non-blocking;
- если буфер подписчика переполнен, событие дропается;
- инкрементируется `sse_drops_total`;
- worker execution не должен блокироваться из-за медленного клиента.

Начальная политика закрытия:
- в первой реализации не закрывать соединение автоматически;
- медленный подписчик может оставаться подключённым, но пропускать события под нагрузкой.

Это осознанный tradeoff: защита выполнения job важнее, чем гарантия доставки каждого промежуточного события.

### Metrics

SSE endpoint’ы исключаются из обычных HTTP latency metrics.

Вместо этого вводятся отдельные streaming metrics:
- `sse_subscribers`
- `sse_events_total`
- `sse_drops_total`

Причина:
- long-lived stream искажает обычные latency metrics.

## Политика для Range

### Auth

Доступ к `GET /api/v1/users/{id}/export`:
- сам пользователь;
- либо admin.

### Поведение ответа

Endpoint должен поддерживать:
- полный ответ целиком;
- частичный ответ с `206 Partial Content`;
- `Content-Range`, когда используется Range.

### Направление реализации

Предпочтительно:
- `http.ServeContent`

Причина:
- он корректно обрабатывает Range requests и conditional headers.

## Политика для gRPC

### Граница сервиса

gRPC handlers должны вызывать существующий service/repository слой.
Бизнес-логика не должна дублироваться.

### Маппинг ошибок

Service/domain ошибки должны маппиться в gRPC status codes:

- not found → `NotFound`
- unauthenticated → `Unauthenticated`
- forbidden → `PermissionDenied`
- invalid input → `InvalidArgument`
- transient unavailable → `Unavailable`
- unexpected internal error → `Internal`

### Deadlines и cancellation

gRPC handlers обязаны уважать входящий `context.Context`, включая deadlines и cancellation.

### Request ID

Unary interceptor должен:
- извлекать request-id из metadata, если он передан;
- иначе генерировать новый доверенный request-id;
- последовательно attach/log’ировать его для gRPC requests.

Это gRPC-аналог существующей HTTP-семантики `request-id`.

## Problem details и текущая HTTP error model

Существующая HTTP error handling модель остаётся без изменений:

- HTTP handlers продолжают использовать централизованный Problem+JSON mapping;
- `request_id` по-прежнему включается как extension member в error response.

Step 7 **не** заменяет HTTP Problem+JSON каким-либо новым форматом.

## Ожидания по shutdown и readiness

Реализация Step 7 должна расширить lifecycle management для:

- shutdown HTTP server;
- shutdown HTTPS server, если он включён;
- graceful shutdown gRPC server;
- shutdown event hub без `send on closed channel`.

`/readyz` должен учитывать состояние:
- workerpool;
- gRPC server, если включён;
- streaming hub, если включён.

## Что не входит в Step 7

Вне scope этого шага:

- замена существующего REST API на gRPC;
- глобальный REST↔gRPC gateway для всего сервиса;
- выбор WebSocket как вторичной протокольной части;
- изменения Step 4–6 auth, CORS, metrics, queue, retry или request-id semantics вне минимально нужных интеграций.

## Планируемая конфигурация

### HTTP / TLS
- `HTTP_ADDR`
- `HTTPS_ADDR`
- `HTTP_TLS_ENABLE`
- `HTTP_TLS_CERT_FILE`
- `HTTP_TLS_KEY_FILE`

### gRPC
- `GRPC_ENABLE`
- `GRPC_ADDR`

### Streaming
- `STREAMING_SSE_HEARTBEAT`
- `STREAMING_SUBSCRIBER_BUFFER`

Правила валидации:
- некорректный или неполный TLS config должен валить startup fast-fail;
- включённый gRPC без адреса — ошибка;
- невалидные heartbeat/buffer values — ошибка.

## Порядок реализации

После S7-B1 реализация идёт в таком порядке:

1. config additions для TLS, gRPC, streaming;
2. optional HTTPS listener;
3. event hub;
4. интеграция workerpool → event hub publish;
5. SSE endpoint;
6. изоляция SSE metrics;
7. proto + gRPC server;
8. Range endpoint;
9. shutdown/readiness integration;
10. tests;
11. README polish.

## Критерии готовности S7-B1

S7-B1 считается завершённым, когда в этой спецификации явно зафиксированы решения:

- HTTP остаётся;
- HTTPS опционален и обслуживает тот же handler chain;
- gRPC поднимается отдельно и использует `JobsService`;
- SSE — обязательный streaming-механизм;
- Range — вторичная протокольная часть;
- auth, heartbeat, backpressure, metrics и shutdown expectations явно зафиксированы.
