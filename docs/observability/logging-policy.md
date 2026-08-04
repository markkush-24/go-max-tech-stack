# Logging Policy

This document defines the log field and privacy baseline for application logs.
It is intentionally narrower than the future tracing and metrics contracts.

## Allowed Fields

Use stable, low-cardinality operational fields by default:

- `component`
- `event`
- `operation`
- `method`
- `route`
- `status`
- `duration_ms`
- `outcome`
- `reason`
- `error_kind`
- `request_id`

`request_id` is allowed for correlation and must not be used as a metric label.
`job_id` is allowed only for job-specific operational records such as SSE
connection lifecycle logs. User identifiers are not allowed in logs unless a
future task defines a reviewed audit event that requires them.

## Forbidden Fields

Do not log:

- access tokens, refresh tokens, API keys, JWTs or Authorization headers;
- request or response bodies;
- passwords, secrets, key material or DSNs with credentials;
- raw upstream URLs containing user input or query strings;
- raw CORS origins or requested header names;
- arbitrary baggage values;
- full request paths when a route pattern is available.

Prefer route templates such as `/api/v1/jobs/{id}/events` over concrete paths.
Prefer `error_kind` over raw error strings for dependency and security events.

## Security Decisions

Authentication, authorization and CORS denial logs must be safe audit events.
They may include normalized reason fields:

- `authn_kind`
- `authz_kind`
- `cors_denial`

They must not include tokens, principals, origins, requested header values or
raw verifier errors.

## Retry Events

Profile retry logs are grouped by `request_id` and `operation=profile.fetch`.
Retry records may include:

- `attempt`
- `attempts`
- `max_attempts`
- `backoff_ms`
- `deadline_remaining_ms`
- `error_kind`
- `outcome`

They must not include the Profile user ID, upstream URL or raw error text.

## SSE Events

SSE logs are connection-level only:

- connection opened;
- connection closed;
- write failure.

Per-message SSE payloads and event data are never logged. `job_id` is allowed
for connection correlation because the stream itself is job-scoped.

## Shutdown Events

Shutdown logging must include one final summary with trigger, outcome and
`duration_ms`. Aggregate counts such as queue depth, repaired active jobs and
SSE subscriber/event/drop counts are allowed. Raw cleanup error strings should
stay on component-specific diagnostic logs rather than the final summary.

## Propagation And Privacy

`request_id` is the current cross-boundary correlation key. Trace IDs, span IDs
and baggage are not emitted until the tracing tasks define ownership and trust
rules. External trace headers must not be accepted or logged ad hoc. Async
propagation metadata must be serializable and explicitly allowlisted before it
is stored or queued.
