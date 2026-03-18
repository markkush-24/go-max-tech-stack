# Step 7 — Protocol and Streaming Spec

## Status

Draft for backlog item **S7-B1**.

## Goal

Extend the existing `pet-study` service with the following capabilities:

- an optional HTTPS listener for the existing HTTP API;
- a separate gRPC server as an internal transport;
- SSE for live job events;
- Range support for partial resource downloads;

while **not breaking the Step 4–6 invariants**:

- `http.ServeMux` must not be bypassed in a way that breaks `r.Pattern` / `PathValue`;
- `request-id` semantics must remain consistent;
- CORS stays limited to the API subtree, and preflight short-circuit must happen before auth;
- `/debug/*` remains debug-only and excluded from normal HTTP metrics;
- normal API timeouts and Step 4–6 behavior must not be accidentally broken.

## Transport layout

The service will support three listener types.

### 1. HTTP

The existing REST API continues to run in normal HTTP mode for local/dev scenarios.

Example:
- `HTTP_ADDR=:8080`

### 2. HTTPS

An optional second listener with the **same handler chain** and **the same routes** as HTTP, but running over TLS.

Example:
- `HTTP_TLS_ADDR=:8443`

Notes:
- HTTPS does **not** introduce separate endpoints; it serves the same API over TLS.
- In normal TLS mode, the server presents its certificate to the client.
- On the HTTPS listener, HTTP/2 is expected to work through Go `net/http` standard TLS/ALPN support.

### 3. gRPC

An optional separate gRPC server on its own address.

Example:
- `GRPC_ADDR=:9090`

Notes:
- gRPC is a separate server, not a replacement for the HTTP API.
- It uses the same domain/service layer as the HTTP handlers.
- In Step 7, it is treated as an internal transport interface.

## Locked scope for Step 7

### Mandatory streaming mechanism

**SSE** on the endpoint:

`GET /api/v1/jobs/{id}/events`

Purpose:
- stream job lifecycle events to the client:
    - `queued`
    - `running`
    - `progress` (optional)
    - `succeeded`
    - `failed`

SSE is chosen because the project needs a **one-way server-to-client stream** for async jobs.

### gRPC service

Chosen service:

**JobsService**

Initial scope:
- `GetJob(id)` — required
- `WatchJob(id)` — optional, if time remains

Reason for this choice:
- jobs already exist in the project;
- async processing and job status transitions already exist;
- SSE and possible gRPC streaming both fit the job domain naturally.

### Secondary protocol feature

Chosen mechanism:

**Range support**

Planned endpoint:
- `GET /api/v1/users/{id}/export`

Purpose:
- support partial download of a large exported resource.

Implementation direction:
- prefer `http.ServeContent`, because it correctly handles Range requests and related conditional headers.

## Job ownership model

Step 7 introduces ownership for async jobs.

When a job is created, the service stores the owner identity
(for example, `OwnerUserID`) taken from the authenticated principal.

Owner/admin authorization for job-based endpoints relies on this field.

At minimum, this applies to:
- `GET /api/v1/jobs/{id}/events`

For consistency, Step 7 also moves access to:
- `GET /api/v1/jobs/{id}`

to the same rule:
- the job owner;
- or an admin.

## Locked policies

## SSE policy

### Auth

Access to `GET /api/v1/jobs/{id}/events`:
- the job owner;
- or an admin.

The existing Step 6 authn/authz model must be reused.

### Stream format

Headers:
- `Content-Type: text/event-stream`
- `Cache-Control: no-cache`

Optional:
- `Connection: keep-alive` for HTTP/1.1 compatibility

### Flush

The handler must verify `http.Flusher` support, because correct streaming depends on explicit `Flush()` after each event and heartbeat.

### Heartbeat

Fixed heartbeat interval:
- `15s`

Heartbeat format:
- SSE comment frame, for example `: ping`

Reason:
- keep the long-lived connection active across proxies/intermediaries.

### Cancellation

The SSE loop must terminate on:
- `r.Context().Done()`
- subscription cancellation
- event hub shutdown

### Backpressure

Subscriber policy:
- bounded subscriber buffer;
- initial size: `16`

Publish policy:
- publish is non-blocking;
- if a subscriber buffer is full, the event is dropped;
- `sse_drops_total` is incremented;
- worker execution must not block because of a slow client.

Initial connection-closing policy:
- in the first implementation, do not close the connection automatically;
- a slow subscriber may remain connected but miss events under load.

This is an intentional tradeoff: protecting job execution is more important than guaranteeing delivery of every intermediate event.

### Streaming write timeout

`STREAMING_WRITE_TIMEOUT` defines the limit for a single write/flush attempt in a streaming handler.

It must not change the global timeout behavior of the normal HTTP API.

If writing an event or heartbeat to the stream does not complete within this timeout,
the connection is treated as stalled and the handler terminates the stream.

### Metrics

SSE endpoints are excluded from normal HTTP latency metrics.

This exclusion must be implemented **without bypassing `http.ServeMux`**, keeping `r.Pattern` as the canonical route identifier.

Instead, separate streaming metrics are introduced:
- `sse_subscribers`
- `sse_events_total`
- `sse_drops_total`

Reason:
- a long-lived stream distorts normal latency metrics.

## Range policy

### Auth

Access to `GET /api/v1/users/{id}/export`:
- the user themselves;
- or an admin.

### Response behavior

The endpoint must support:
- a full response;
- a partial response with `206 Partial Content`;
- `Content-Range` when Range is used.

### Implementation direction

Preferred:
- `http.ServeContent`

Reason:
- it correctly handles Range requests and conditional headers.

## gRPC policy

### Service boundary

gRPC handlers must call the existing service/repository layer.
Business logic must not be duplicated.

### Error mapping

Service/domain errors must map to gRPC status codes:

- not found → `NotFound`
- unauthenticated → `Unauthenticated`
- forbidden → `PermissionDenied`
- invalid input → `InvalidArgument`
- transient unavailable → `Unavailable`
- unexpected internal error → `Internal`

### Deadlines and cancellation

gRPC handlers must respect the incoming `context.Context`, including deadlines and cancellation.

### Request ID

A unary interceptor must ensure that a trusted request-id is always present in the gRPC request context and logs.

For internal trusted callers, request-id propagation through gRPC metadata is allowed.

If request-id is missing, malformed, or the caller is not treated as trusted,
the interceptor generates a new trusted request-id.

This is the gRPC analogue of the existing HTTP `request-id` semantics.

## Problem details and the current HTTP error model

The existing HTTP error handling model remains unchanged:

- HTTP handlers continue to use centralized Problem+JSON mapping;
- `request_id` remains included as an extension member in error responses.

Step 7 does **not** replace HTTP Problem+JSON with any new format.

## Shutdown and readiness expectations

Step 7 must extend lifecycle management for:

- shutdown of the HTTP server;
- shutdown of the HTTPS server, if enabled;
- graceful shutdown of the gRPC server;
- shutdown of the event hub without `send on closed channel`.

`/readyz` must account for:
- workerpool state;
- gRPC server state, if enabled;
- streaming hub state, if enabled.

## What is out of scope for Step 7

Outside the scope of this step:

- replacing the existing REST API with gRPC;
- a global REST↔gRPC gateway for the whole service;
- choosing WebSocket as the secondary protocol feature;
- changes to Step 4–6 auth, CORS, metrics, queue, retry, or request-id semantics beyond the minimum integrations required for Step 7.

## Planned configuration

### HTTP / TLS
- `HTTP_ADDR`
- `HTTP_TLS_ADDR`
- `HTTP_TLS_ENABLE`
- `HTTP_TLS_CERT_FILE`
- `HTTP_TLS_KEY_FILE`

### gRPC
- `GRPC_ENABLE`
- `GRPC_ADDR`

### Streaming
- `STREAMING_SSE_HEARTBEAT`
- `STREAMING_SUBSCRIBER_BUFFER`
- `STREAMING_WRITE_TIMEOUT`

Validation rules:
- invalid or incomplete TLS config must fail startup fast;
- enabled gRPC without an address is an error;
- invalid heartbeat/buffer/write-timeout values are an error.

## Implementation order

After S7-B1, implementation proceeds in this order:

1. config additions for TLS, gRPC, and streaming;
2. optional HTTPS listener;
3. event hub;
4. workerpool → event hub publish integration;
5. SSE endpoint;
6. SSE metrics isolation;
7. proto + gRPC server;
8. Range endpoint;
9. shutdown/readiness integration;
10. tests;
11. README polish.

## S7-B1 completion criteria

S7-B1 is complete when this specification explicitly locks the following decisions:

- HTTP remains available;
- HTTPS is optional and serves the same handler chain;
- gRPC runs separately and uses `JobsService`;
- SSE is the mandatory streaming mechanism;
- Range is the secondary protocol feature;
- job ownership is explicitly introduced for owner/admin checks;
- auth, heartbeat, backpressure, request-id, metrics, write-timeout, and shutdown expectations are explicitly locked.
