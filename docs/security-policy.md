# Политика безопасности (Step 6 / S6-B1)

Этот документ — **источник истины** для поведения security в Step 6 проекта `pet-study`.
Он намеренно **декларативный** (без деталей реализации middleware), чтобы реализация и тесты
не расходились “тихо” с согласованной политикой.

## Identity (JWT claims → Principal)

Минимальный маппинг клеймов:

- `sub` → `user_id` (обязателен; положительное целое, закодированное строкой).
- `role` → `role` (`admin` | `user`).
    - Если `role` отсутствует — считаем роль `user`.

Роли в Step 6 намеренно минимальные:

- `admin`
- `user`

## Матрица доступа

Политика — **deny-by-default**.

Легенда:

- **Public**: аутентификация не требуется.
- **Auth**: требуется валидный JWT.
- **Admin**: требуется валидный JWT и роль `admin`.
- **Self**: resource-level проверка для роли `user`: `sub == {id}`.
- **Job owner**: resource-level проверка для job: `sub == job.owner_user_id`.

| Route (ServeMux pattern)     | Methods | Access | Resource-level | Примечания                                                                   |
|------------------------------|--------:|--------|----------------|------------------------------------------------------------------------------|
| `/livez`                     |     GET | Public | —              | liveness probe                                                               |
| `/readyz`                    |     GET | Public | —              | readiness probe                                                              |
| `/debug/`                    |     GET | Admin  | —              | **только когда `HTTP_DEBUG=true`** (иначе маршрута нет)                      |
| `GET /api/v1/users`          |     GET | Admin  | —              | список всех пользователей                                                    |
| `POST /api/v1/users`         |    POST | Admin  | —              | создание пользователя                                                        |
| `/api/v1/users`              |       * | Admin  | —              | fallback-обработчик коллекции (хелпер для 405); не должен стать обходом auth |
| `/api/v1/users/{id}`         |     GET | Auth   | Self           | `admin` читает любых; `user` — только себя                                   |
| `/api/v1/users/{id}/export`  |     GET | Auth   | Self           | экспорт пользователя; поддерживает Range через `http.ServeContent`           |
| `/api/v1/users/{id}/profile` |     GET | Auth   | Self           | то же правило, что и для `/api/v1/users/{id}`                                |
| `/api/v1/jobs/{id}`          |     GET | Admin  | —              | текущая RBAC-политика оставляет route admin-only                             |
| `/api/v1/jobs/{id}/events`   |     GET | Auth   | Job owner      | SSE stream job events; owner или `admin` после загрузки job                  |
| `/api/v1/jobs/{id}/grpc`     |     GET | Admin  | —              | HTTP -> gRPC bridge; успешен только когда gRPC client подключён              |
| `GET /api/v2/users`          |     GET | Admin  | —              | список всех пользователей (v2)                                               |
| `POST /api/v2/users`         |    POST | Admin  | —              | создание пользователя (v2)                                                   |
| `/api/v2/users`              |       * | Admin  | —              | fallback-обработчик коллекции v2 (хелпер для 405)                            |

### Почему jobs admin-only

Сейчас `jobs` не хранят “created by user_id”, поэтому правило
«user может читать только свои jobs» потребует менять модель/репозиторий/создание job,
что выходит за scope Step 6. Безопасный дефолт — admin-only.

## HTTP семантика (цели для реализации)

- Нет токена / токен невалиден / токен истёк → **401** с `WWW-Authenticate: Bearer ...` и Problem+JSON телом.
- Токен валиден, но прав не хватает → **403** с Problem+JSON.
- request-id обязан присутствовать и быть консистентным в response headers и в Problem+JSON.
