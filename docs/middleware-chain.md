# Canonical middleware chain (Step 6)

Этот документ фиксирует “каноничный” порядок middleware в сервисе (S6-B11).
Важно: порядок ниже описывает **порядок выполнения** (от внешнего к внутреннему).

## 1) Global chain (вокруг root router)

Execution order (outer -> inner):

1. **Proxy: SanitizeRequestIDHeader**
- Удаляет входящий `X-Request-Id`, если запрос пришёл НЕ от trusted proxy.
- Цель: не принимать request-id от произвольных клиентов.

2. **RequestIDMiddleware**
- Генерирует/валидирует request-id.
- Гарантирует echo `X-Request-Id` в response header и `request_id` в Problem+JSON.

3. **Recover (outer)**
- Ловит panic из нижележащих middleware (Logger/Metrics).

4. **TrustProxy**
- Если `RemoteAddr` ∈ trusted proxies: доверяет `X-Forwarded-For` / `X-Forwarded-Proto`.
- Пишет `RequestInfo{ClientIP, Scheme}` в context.

5. **Metrics**
- HTTP метрики (исключает `/debug/*` по текущей реализации).

6. **Logger**
- Логирует method/path/pattern/status/bytes/latency/request-id + (clientIP/scheme из RequestInfo).

7. **Recover (inner)**
- Ловит panic из Router/handlers так, чтобы Logger/Metrics увидели статус 500.

8. **RootRouter (ServeMux)**
- Монтирует sub-роутеры:
- API router (см. ниже)
- Health router (`/livez`, `/readyz`) без auth
- Debug router (`/debug/*`) только при `HTTP_DEBUG=true`

## 2) API subtree chain (вокруг /api/*)

Execution order (outer -> inner) внутри API subtree:

1. **SecurityHeaders**
   - `X-Content-Type-Options: nosniff`
   - `Referrer-Policy: no-referrer`
   - `X-Frame-Options: DENY`
   - `Strict-Transport-Security` только если “effective scheme” == https
     (использует `RequestInfo.Scheme` из TrustProxy, либо `r.TLS != nil`).

2. **CORS**
   - deny-by-default по Origin
   - Preflight `OPTIONS` short-circuit (204) без auth
   - `Vary` корректно выставлен

3. **API ServeMux Router**
   - Для каждого endpoint используется wrap (AppHandler chain):
     - Bulkhead
     - RateLimiter
     - AuthN (JWT) -> 401
     - RBAC -> 403
     - Handler

## 3) Exceptions / invariants

- `OPTIONS` preflight не должен требовать auth: CORS обязан short-circuit до AuthN.
- `/livez`, `/readyz` — без auth.
- `/debug/*`:
  - монтируется только при `HTTP_DEBUG=true`
  - требует auth + admin (RBAC)
  - не должен попадать в http metrics (как сейчас).
- Главный путь обработки запросов — `ServeMux.ServeHTTP` (не ломаем `r.Pattern`).
