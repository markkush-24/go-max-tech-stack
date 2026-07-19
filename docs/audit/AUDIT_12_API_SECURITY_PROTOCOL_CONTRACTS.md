# Audit 12 — API, Security and Protocol Contracts

## Scope

This pass evaluates whether the uploaded `pet-study` working tree exposes a coherent, secure and testable contract across:

- HTTP v1/v2 routing and method semantics;
- Problem Details and error-status mapping;
- JSON/Protobuf content negotiation;
- ETag, conditional requests and Range;
- JWT authentication, RBAC and resource authorization;
- CORS behavior for browser clients;
- trusted-proxy and request-ID boundaries;
- HTTP TLS/HTTP2 and security headers;
- direct gRPC and HTTP-to-gRPC bridge behavior;
- SSE connection and event-delivery semantics;
- health/debug exposure;
- documentation and machine-readable API contracts.

The application code was not changed.

Two audit-only stdlib diagnostics were executed outside the source tree:

1. exact reproduction of the existing `AcceptHeader` selection algorithm with quality values;
2. exact reproduction of the mixed ServeMux pattern strategy for `HEAD` requests.

The full module still could not be executed in this sandbox because the external module archives required by the project were unavailable and network access was disabled. Final protocol verification remains required on Go 1.25.8.

## Reviewed evidence

Primary files:

- `cmd/api/main.go`
- `internal/api/server.go`
- `internal/config/config.go`
- `internal/router/root.go`
- `internal/router/router.go`
- `internal/router/problem_not_found.go`
- `internal/router/health_router.go`
- `internal/routes/users_handler.go`
- `internal/routes/users_handler_v2.go`
- `internal/routes/user_profile_handler.go`
- `internal/routes/job_handler.go`
- `internal/httputils/problem.go`
- `internal/httputils/errmap.go`
- `internal/httputils/apphandler.go`
- `internal/httputils/accept.go`
- `internal/httputils/utils.go`
- `internal/middleware/authorization.go`
- `internal/middleware/rbac.go`
- `internal/middleware/cors.go`
- `internal/middleware/trustproxy.go`
- `internal/middleware/trustproxy_requestid.go`
- `internal/middleware/security_header.go`
- `internal/security/policy.go`
- `internal/security/jwt_verifier_hs256.go`
- `internal/requestid/requestid.go`
- `internal/interceptors/interceptors.go`
- `internal/transport/grpcserver/runtime.go`
- `internal/transport/grpcserver/grpc_job_service.go`
- `internal/transport/grpcclient/grpcclient.go`
- `internal/transport/pb/*.proto`
- `internal/stream/stream.go`
- `internal/health/health.go`
- README and protocol/security tests.

Primary standards used for comparison:

- RFC 9457 — Problem Details for HTTP APIs (obsoletes RFC 7807)
- RFC 9110 — HTTP Semantics
- RFC 9111 — HTTP Caching
- RFC 6797 — HTTP Strict Transport Security
- RFC 6750 — Bearer Token Usage
- RFC 7519 and RFC 8725 — JWT and JWT Best Current Practices
- WHATWG Fetch Standard — CORS protocol
- Go `net/http` ServeMux and ServeContent documentation
- official gRPC authentication, metadata and status-code documentation.

## Executive result

API/security/protocol contracts are **PARTIAL/BROKEN**.

The project already has several strong contract foundations:

- routing remains inside `http.ServeMux`, preserving `r.Pattern` and `PathValue`;
- every registered API pattern has a matching RBAC policy entry;
- API errors are centralized into a consistent Problem-shaped JSON document;
- request ID is propagated into API error headers and bodies;
- JWT algorithm selection is pinned and registered claims are validated;
- owner/admin checks exist for user/profile/SSE resources;
- CORS validates exact origins and rejects wildcard plus credentials;
- forwarded headers are ignored unless the immediate peer is trusted;
- weak ETag comparison for `If-None-Match` follows the required comparison mode;
- Range behavior is delegated to `http.ServeContent`;
- direct HTTP and HTTPS share one handler tree.

However, the direct gRPC listener is a security bypass: when enabled it listens on a TCP address, uses plaintext transport, has no authentication or authorization interceptor, exposes reflection and lets any direct client retrieve any job by ID. HTTP RBAC protects only the bridge endpoint, not the gRPC service itself.

The HTTP representation contract also contains verified semantic defects. The `Accept` parser ignores quality values and can choose a representation explicitly assigned `q=0`. Mixed route registration makes `HEAD` work for collections but return `405` for item/export/profile/job routes. CORS does not expose the response headers browser clients need for async jobs, ETag, request correlation, retries and Range. SSE does not commit and flush its successful response until the first event or heartbeat, so connection establishment can appear stalled.

No OpenAPI document or generated HTTP contract exists, so drift between code, README, historical step summaries and tests is not mechanically detectable.

## Dynamic audit evidence

### Accept quality values are ignored

The existing algorithm was reproduced exactly with these inputs:

```text
Accept: application/json;q=1, application/protobuf;q=0
=> application/protobuf

Accept: application/json;q=0
=> application/json

Accept: application/protobuf;q=0, */*;q=1
=> application/protobuf

Accept: application/json;q=0.9, application/protobuf;q=0.1
=> application/protobuf
```

A representation with `q=0` is not acceptable, and higher quality preferences must win. The current implementation instead returns Protobuf immediately whenever its media type appears anywhere in the list.

### HEAD behavior differs by route shape

The project uses method-specific collection patterns and method-agnostic item wrappers:

```text
GET /api/v1/users
/api/v1/users/{id} + manual r.Method == GET
```

Go ServeMux treats a `GET` pattern as matching both GET and HEAD. The audit reproduction produced:

```text
HEAD /collection => 200
HEAD /item/1     => 405, Allow: GET
```

The same inconsistency applies to jobs, Profile, export, gRPC bridge and SSE item routes. Collection `405` responses also advertise `Allow: GET, POST` even though the registered GET pattern accepts HEAD.

## Positive findings

### P1 — API route and policy tables currently align

Every pattern registered by `internal/router/router.go` has a matching entry in `security.DefaultPolicy`.

This prevents a current registered API endpoint from accidentally falling into `AuthZNoPolicyRule` solely because the policy table was forgotten.

Health and debug are mounted separately and have their corresponding public/admin rules in their own handler paths.

### P2 — ServeMux routing invariants are preserved

The final dispatch still calls `mux.ServeHTTP`, so:

- `r.Pattern` is populated;
- `PathValue` works;
- RBAC and low-cardinality HTTP metrics can use matched patterns;
- route matching is not duplicated in middleware.

This is an important invariant to preserve through OpenTelemetry and OpenAPI integration.

### P3 — Problem response shape is centralized

`WriteProblem` consistently provides:

- `type` with `about:blank` fallback;
- title/status/detail;
- request path as instance;
- request ID extension;
- `application/problem+json` content type.

Validation errors use a bounded structured `invalid_params` extension rather than embedding arbitrary Go errors.

### P4 — JWT verifier has several correct defensive controls

The verifier:

- accepts only HS256;
- configures the parser with an explicit valid-method list;
- requires `exp`;
- validates `nbf` when present;
- supports configured issuer and audience validation;
- applies bounded clock skew;
- validates positive numeric subject and bounded role values;
- supports key selection by `kid` and key rotation without `kid`.

This is a good base, despite unsafe development defaults discussed below.

### P5 — HTTP resource authorization is explicit

User, Profile, export and SSE handlers independently perform owner/admin checks after authentication. This avoids placing resource-specific ownership logic inside generic routing middleware.

### P6 — CORS has a sound base policy

The implementation:

- is scoped to the API subtree;
- is deny-by-default for requests carrying `Origin`;
- exact-matches normalized origins;
- validates preflight method and headers;
- returns `204` for accepted preflight;
- prevents wildcard origin with credentials;
- short-circuits before authentication.

### P7 — Trusted proxy checks are fail-closed at the immediate-peer boundary

The application ignores `X-Forwarded-For`, `X-Forwarded-Proto` and incoming request IDs unless `RemoteAddr` belongs to a configured trusted network.

Direct untrusted clients therefore cannot set the effective scheme/client IP or preserve their own request ID.

### P8 — Conditional GET implementation has correct core comparison behavior

`IfNoneMatchMatches`:

- supports `*`;
- supports lists;
- performs weak comparison;
- accepts weak/strong forms of the same opaque tag.

The v1 user response sets ETag before evaluating `If-None-Match`, and a match returns `304` without a body while retaining `Cache-Control` and `Vary: Accept`.

### P9 — Range parsing is delegated to the standard library

The export handler uses `http.ServeContent` instead of hand-parsing byte ranges. This provides standard `206`, `Content-Range` and unsatisfiable-range behavior for the generated byte representation.

## Findings

## F1 — Direct gRPC is an authentication and authorization bypass

Severity: **CRITICAL**

When `GRPC_ENABLE=true`, `grpcserver.NewRuntime`:

- listens on `GRPC_ADDR`, default `:9090`;
- creates a plaintext `grpc.Server`;
- installs only request-ID/logging interception;
- registers reflection unconditionally;
- exposes `JobsService.GetJob` without principal or owner checks.

The client similarly uses `insecure.NewCredentials()`.

The HTTP bridge endpoint is admin-only, but a direct network client can bypass the entire HTTP JWT/RBAC stack and call:

```text
pb.JobsService/GetJob
```

for any positive job ID.

Required direction:

- bind internal gRPC to an explicitly private interface by default;
- add TLS or mTLS;
- add authentication and authorization interceptors;
- propagate a trusted principal, not only request ID;
- enforce owner/admin policy in the gRPC service;
- gate reflection behind a development/debug flag;
- test direct gRPC denial independently of the HTTP bridge.

## F2 — gRPC status and metadata contracts are incomplete

Severity: **HIGH**

The service maps only invalid ID and not-found specifically. Other failures become `Internal`.

The HTTP bridge maps only:

- InvalidArgument;
- NotFound;
- PermissionDenied;
- Unauthenticated.

It does not preserve:

- Canceled;
- DeadlineExceeded;
- Unavailable;
- ResourceExhausted.

The server accepts unsanitized incoming `request-id`, and a generated request ID is not returned to the direct gRPC client in response metadata.

Required direction:

- define a complete service error/status mapping;
- sanitize and bound request metadata;
- return generated correlation metadata;
- add a per-call deadline budget for the HTTP bridge;
- add unary and stream auth/tracing interceptors or stats handlers.

## F3 — HTTP content negotiation violates Accept quality semantics

Severity: **HIGH**

`AcceptHeader` parses media types but discards all parameters, including `q`.

It also returns Protobuf immediately, before comparing preference weights or specificity.

Consequences:

- `q=0` is ignored;
- a lower-priority Protobuf option overrides a higher-priority JSON option;
- media-range specificity is ignored;
- invalid list members are silently skipped;
- representation parameters are silently ignored.

Required direction:

- implement a bounded standards-aware negotiator;
- reject or explicitly define unsupported representation parameters;
- sort by quality, specificity and deterministic server preference;
- add table tests for `q=0`, ties, wildcards, invalid members and aliases.

## F4 — HEAD and Allow contracts are inconsistent

Severity: **MEDIUM/HIGH**

Collection patterns use method-specific `GET`, which Go also matches for HEAD. Item routes use method-agnostic patterns with manual GET checks and reject HEAD.

This creates different method semantics for similar resources:

```text
HEAD /api/v1/users          -> GET handler semantics
HEAD /api/v1/users/{id}     -> 405
HEAD /api/v1/users/{id}/export -> 405, although ServeContent can serve HEAD
```

`Allow` for collection fallback is `GET, POST` even though HEAD is accepted by the GET pattern.

Required direction:

- choose and document one HEAD policy;
- preferably use method-specific ServeMux patterns consistently;
- explicitly register HEAD only when its representation/header contract differs;
- ensure every `405 Allow` lists every actually supported method.

## F5 — Browser clients cannot read important response headers

Severity: **HIGH for browser/API usability**

The CORS middleware never sets `Access-Control-Expose-Headers`.

Fetch exposes only safelisted response headers by default. Browser JavaScript therefore cannot reliably read project-specific contract headers such as:

- `Location` from async/sync creation;
- `ETag`;
- `X-Request-Id`;
- `Retry-After`;
- `Content-Range`;
- `WWW-Authenticate`;
- possibly `Allow` for method discovery.

This makes important API features unusable or unobservable from an allowed browser origin even though the request itself succeeds.

Required direction:

- add a validated `CORS_EXPOSED_HEADERS` configuration;
- explicitly expose the bounded list required by public clients;
- test successful and error responses through a CORS-style client contract.

## F6 — CORS Vary coverage is incomplete on denial paths

Severity: **MEDIUM**

For a non-wildcard allowlist, response selection depends on `Origin` for both accepted and denied origins.

However, `Vary: Origin` is added only after origin acceptance. A denied-origin `403` lacks it.

Likewise, denied preflight method/header responses can depend on:

- `Access-Control-Request-Method`;
- `Access-Control-Request-Headers`;

but those Vary fields are added only on successful preflight.

A cache can therefore reuse a denial generated for one origin/method/header combination for another combination.

Required direction:

- add all applicable Vary fields before any branch whose response depends on them;
- add cache tests for accepted and denied CORS responses.

## F7 — CORS policy is global rather than route-specific

Severity: **MEDIUM**

The preflight check validates requested methods and headers against one global list, not against the selected route.

A configured method can therefore receive an accepted preflight even when a specific route does not implement it. Authentication and authorization requirements are also not represented in the CORS policy.

This is not an authorization bypass—the actual request still passes routing/auth—but it makes preflight results less precise.

Additionally, both AuthN and RBAC bypass every `OPTIONS` request, not only a validated CORS preflight. The outer CORS middleware handles real preflight first, but the inner bypass should be narrowed or explicitly documented.

## F8 — Security-header configuration is partly ignored

Severity: **MEDIUM/HIGH**

`SecurityHeadersConfig.ReferrerPolicy` is parsed from configuration, but middleware always writes:

```text
Referrer-Policy: no-referrer
```

The configured value has no effect.

The middleware also hardcodes `X-Frame-Options: DENY` and exposes no explicit policy choice between X-Frame-Options and CSP `frame-ancestors`.

Required direction:

- validate supported policy values;
- use the configured value;
- document immutable versus configurable headers;
- test non-default configurations.

## F9 — HSTS is scoped to the API subtree instead of the HTTPS host

Severity: **HIGH for a production HTTPS profile**

SecurityHeaders wraps only `userRouter`.

Consequently, HTTPS responses from:

- `/livez`;
- `/readyz`;
- `/debug/*`;
- possibly other root-level paths

do not receive HSTS.

HSTS is a host policy, not a per-API-resource policy. A user agent can learn/update that policy from any secure response it receives, so applying it only to one subtree creates accidental dependence on which URL the client first visits.

Required direction:

- split host-wide transport headers from API-only content/browser headers;
- apply HSTS outside the root router when effective HTTPS is trusted;
- keep self-signed local-development behavior explicitly disabled by default.

## F10 — TLS and HTTP2 have no explicit production security profile

Severity: **MEDIUM**

HTTPS loads one certificate and relies on `net/http` defaults. There is no project-level configuration or assertion for:

- minimum TLS version;
- client authentication;
- certificate reload/rotation;
- cipher/profile policy;
- HTTP-to-HTTPS redirect or deliberate dual-listener policy;
- test proving negotiated HTTP/2 on the target Go version.

This is acceptable for a local lab default, but it is not yet a defined production-like transport contract.

The gRPC listener does not inherit HTTP TLS settings at all.

## F11 — Development JWT defaults are unsafe without an environment guard

Severity: **HIGH configuration hazard**

Default auth configuration accepts HS256 tokens signed with:

```text
kid=dev
secret=dev-secret
```

Issuer and audience checks are disabled by default.

There is no environment/profile guard preventing these defaults from being used on a non-local bind address or production-like deployment.

The verifier correctly pins the algorithm, but a known low-entropy shared secret defeats that protection.

Required direction:

- introduce explicit deployment environment/profile;
- reject development secrets outside local/test mode;
- require issuer and audience in production-like mode;
- require minimum secret entropy or move to asymmetric/JWKS verification;
- keep development token generation clearly isolated.

## F12 — Trusted X-Forwarded-For semantics depend on undocumented proxy sanitization

Severity: **MEDIUM/HIGH**

Once the immediate peer is trusted, `firstIPFromXFF` accepts the leftmost syntactically valid address.

This is safe only if the trusted edge proxy overwrites or correctly sanitizes client-supplied XFF. If the proxy merely appends its own hop to an existing client header, a caller can control the leftmost value.

Required direction:

- document the required edge-proxy behavior;
- preferably walk the chain from right to left and remove trusted proxy hops;
- bound header length and number of hops;
- test malicious pre-existing XFF values through a simulated trusted proxy.

## F13 — Public readiness responses expose raw internal errors

Severity: **MEDIUM/HIGH**

`/readyz` is public and inserts `err.Error()` from each failed check directly into JSON.

A PostgreSQL ping error or future dependency check can disclose:

- internal hostnames and ports;
- database names;
- driver/network details;
- topology or component implementation information.

Required direction:

- return bounded public states such as `ok`, `failed`, `timeout`;
- log the full internal error with request/trace correlation;
- optionally expose detailed diagnostics only through an admin/debug endpoint.

## F14 — Problem Details source and machine contract are outdated/incomplete

Severity: **MEDIUM**

Comments and README still identify RFC 7807, which was obsoleted by RFC 9457.

More importantly, nearly every problem defaults to:

```text
type: about:blank
```

Clients must distinguish failures using status and English `detail` strings. There are no stable project problem type URIs or bounded machine error codes for:

- queue full/stopped;
- validation;
- authn/authz reasons;
- outbound dependency errors;
- bulkhead/rate limiting;
- gRPC bridge failures.

`JobProblem` duplicates but does not share the same type as HTTP `Problem`, creating a second drift-prone error schema.

Required direction:

- update references to RFC 9457;
- define a small stable problem-type catalog;
- keep human detail non-contractual;
- share reusable schema/types where appropriate;
- document extensions such as `request_id` and `invalid_params` in OpenAPI.

## F15 — Some HTTP status mappings are semantically misleading

Severity: **HIGH for client and SLO semantics**

Examples:

- `profile.ErrCanceled` maps to `408 Request Timeout`, although 408 specifically means the server did not receive a complete request message in time;
- generic `context.Canceled` and `context.DeadlineExceeded` are not classified;
- DB timeout/unavailable errors fall through to generic 500;
- gRPC Unavailable/DeadlineExceeded/Canceled often become generic 500/Internal;
- all Profile upstream 4xx become 502 regardless of domain meaning.

Required direction:

- define an explicit transport error taxonomy;
- distinguish client disconnect, server operation timeout and dependency timeout;
- map dependency availability consistently across HTTP and gRPC;
- align telemetry outcome classification with the public status contract.

## F16 — Problem responses containing request-specific data have no cache policy

Severity: **MEDIUM**

`WriteProblem` does not set `Cache-Control`.

Some error statuses used by the project, notably 404 and 405, are heuristically cacheable. Their bodies contain request-specific fields such as:

- `request_id`;
- `instance`.

A cache could retain and reuse a Problem response generated for another request unless authorization/cache rules happen to prevent it.

Required direction:

- default dynamic Problem responses to `Cache-Control: no-store` or another explicit policy;
- define exceptions only for intentionally cacheable stable errors;
- test cache headers on 404, 405 and authorization failures.

## F17 — SSE does not establish the stream immediately

Severity: **HIGH**

The handler sets headers, subscribes and enters its select loop without:

- writing `200`;
- flushing the headers;
- sending an initial state/snapshot event.

The client may not receive a response until:

- the first job event;
- or the first heartbeat, default 15 seconds later.

This can look like connection failure to clients, proxies or load balancers with shorter header/idle budgets.

Required direction:

- commit `200 OK` and flush immediately after successful authorization/subscription;
- send a versioned current-state event;
- ensure heartbeat is comfortably below every applicable proxy/server timeout.

## F18 — SSE delivery has no resumable contract

Severity: **HIGH**

As identified in Audit 11:

- job state is read before subscription;
- a terminal event can occur in that gap;
- full subscriber buffers silently drop events;
- events have no SSE `id`;
- `Last-Event-ID` is ignored;
- no replay or resync endpoint exists.

A client can therefore remain connected forever after missing the terminal event.

Required direction:

- publish monotonic transition version/sequence;
- subscribe and snapshot atomically or re-read after subscription;
- support resync/replay or make current state authoritative after reconnect;
- define the meaning of dropped intermediate versus terminal events.

## F19 — Streaming errors can corrupt an already committed response

Severity: **HIGH**

SSE write/flush failures return through `AppHandler`, which then calls `WriteProblem`.

If streaming has already started, this attempts to write a second HTTP status and Problem JSON into a stream response.

The same generic risk exists for any handler that writes part of a response and then returns an error.

Required direction:

- distinguish pre-commit and post-commit errors;
- terminate/log post-commit stream errors without another response body;
- expose response-commit state in the wrapper;
- add tests for disconnect and write-timeout paths.

## F20 — Range works, but the export representation lacks validators and consistent HEAD

Severity: **MEDIUM**

`ServeContent` correctly implements byte ranges, but the handler passes zero modification time and sets no ETag.

Consequences:

- no useful `Last-Modified`;
- `If-Range` cannot safely validate the generated representation;
- repeated full JSON materialization is required;
- HEAD is rejected by the surrounding manual GET wrapper even though `ServeContent` can handle HEAD.

For a tiny user export this is acceptable as a Range demonstration, but it is not yet a robust large-object download contract.

## F21 — v2 contains dead item-handler code but no item endpoint contract

Severity: **LOW/MEDIUM**

`UsersV2Handler.GetByID` exists, but there is no v2 item route, no matching policy entry and README lists only v2 collection endpoints.

Historical project context described v2 item authorization, while the uploaded source does not expose it.

Required direction:

- either remove dead handler code;
- or deliberately add and document `GET /api/v2/users/{id}` with v2 DTO, policy, ETag/negotiation decisions and tests.

## F22 — Unknown-route behavior contradicts the deny-by-default policy comment

Severity: **LOW/MEDIUM**

`DefaultPolicy` describes unknown patterns as closed by default. In practice, `WithProblemNotFound` checks for an unmatched pattern before AuthN/RBAC wrappers run and returns a public 404.

Public 404 may be the desired anti-enumeration policy, but code and policy documentation currently describe different behavior.

The same ambiguity exists for unsupported methods: protected route wrappers authenticate before producing 405, so an unauthenticated caller can receive 401 instead of method information.

Required direction:

- explicitly document unknown-route and unsupported-method precedence;
- encode it in contract tests and OpenAPI/gateway policy.

## F23 — Query parameter contract is permissive and undocumented

Severity: **LOW**

Create handlers interpret only:

```text
async=1
```

Any other value silently becomes synchronous creation, and unknown query parameters are ignored.

A typo such as `async=true` can therefore change operation semantics without an error.

Required direction:

- define accepted query values;
- reject malformed/duplicate/unknown behavior when safety matters;
- include exact semantics in OpenAPI.

## F24 — No machine-readable HTTP contract exists

Severity: **HIGH / MISSING**

The repository contains no OpenAPI/Swagger document, runtime request/response validation or generated typed HTTP client/server contract.

As a result, nothing mechanically checks:

- route/status/header drift;
- v1/v2 schema drift;
- Problem extensions;
- JWT requirements;
- CORS-visible response headers;
- async `202`/Location behavior;
- SSE content type;
- ETag/304;
- Protobuf alternatives;
- Range responses.

This is the expected primary subject of the final contract/quality implementation phase.

## Contract test gaps

Missing direct tests include:

- Accept quality-value and specificity matrix;
- HEAD behavior and complete `Allow` headers for every route class;
- CORS exposed response headers;
- Vary on denied origins/methods/headers;
- non-default Referrer-Policy configuration;
- HSTS on health/debug/root HTTPS responses;
- direct gRPC unauthenticated/unauthorized calls;
- gRPC TLS/mTLS and reflection policy;
- malformed/oversized gRPC request IDs;
- public readiness redaction;
- cache policy on 404/405/401/403 Problem responses;
- immediate SSE header flush;
- terminal-event race/reconnect/resync;
- post-commit SSE write failure;
- HEAD/If-Range for export;
- unknown-route versus auth precedence;
- v2 item absence/presence as an explicit contract;
- OpenAPI runtime conformance.

## Required remediation order

Before generating the final OpenAPI contract, the protocol contract should be stabilized in this order:

1. Secure direct gRPC with bind/TLS/authz policy.
2. Fix Accept quality semantics.
3. Choose one HEAD/method-registration strategy.
4. Fix CORS exposed headers and Vary behavior.
5. Split host-wide HSTS from API-only headers and honor configuration.
6. Add production-like JWT/TLS configuration guards.
7. Redact public readiness details.
8. Define RFC 9457 problem types and error taxonomy.
9. Fix SSE handshake, state snapshot and committed-error handling.
10. Decide v2 item and Range validator contracts.
11. Create OpenAPI, examples and generated typed-client pipeline.
12. Add runtime contract tests and breaking-change checks.

## Status summary

| Capability | Status |
|---|---|
| ServeMux/r.Pattern contract | PRESENT |
| API route-to-RBAC policy alignment | PRESENT |
| Central Problem-shaped errors | PARTIAL |
| RFC 9457 problem catalog | MISSING |
| JWT verification core | PRESENT/PARTIAL |
| Safe production JWT defaults | BROKEN/MISSING |
| Resource-level HTTP authorization | PRESENT |
| Direct gRPC authorization | BROKEN |
| CORS origin/preflight core | PRESENT/PARTIAL |
| Browser-visible response headers | MISSING |
| Complete CORS cache variation | BROKEN/PARTIAL |
| Trusted immediate-proxy boundary | PRESENT |
| Robust forwarded-chain interpretation | PARTIAL |
| HSTS host-wide policy | BROKEN/PARTIAL |
| Explicit production TLS profile | MISSING/PARTIAL |
| JSON/Protobuf negotiation | BROKEN/PARTIAL |
| ETag/If-None-Match | PRESENT for v1 item |
| Range support | PRESENT/PARTIAL |
| SSE establishment/replay | BROKEN/PARTIAL |
| gRPC status/metadata contract | PARTIAL/BROKEN |
| OpenAPI/codegen contract | MISSING |

## Sources

- RFC 9457, Problem Details for HTTP APIs: https://www.rfc-editor.org/rfc/rfc9457.html
- RFC 9110, HTTP Semantics: https://www.rfc-editor.org/rfc/rfc9110.html
- RFC 9111, HTTP Caching: https://www.rfc-editor.org/rfc/rfc9111.html
- RFC 6797, HTTP Strict Transport Security: https://www.rfc-editor.org/rfc/rfc6797.html
- RFC 6750, Bearer Token Usage: https://www.rfc-editor.org/rfc/rfc6750.html
- RFC 7519, JSON Web Token: https://www.rfc-editor.org/rfc/rfc7519.html
- RFC 8725, JWT Best Current Practices: https://www.rfc-editor.org/rfc/rfc8725.html
- WHATWG Fetch Standard, CORS: https://fetch.spec.whatwg.org/
- Go net/http documentation: https://pkg.go.dev/net/http
- Go 1.22 routing enhancements: https://go.dev/blog/routing-enhancements
- gRPC authentication: https://grpc.io/docs/guides/auth/
- gRPC metadata: https://grpc.io/docs/guides/metadata/
- gRPC status codes: https://grpc.io/docs/guides/status-codes/
