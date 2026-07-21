# ADR-001: Direct gRPC Security Model

Date: 2026-07-21

Status: Accepted

## Context

The application can start a direct gRPC listener when `GRPC_ENABLE=true`.
The current listener is used by the HTTP-to-gRPC bridge and by local `grpcurl`
checks, but it does not yet enforce transport credentials, caller
authentication, resource authorization, or environment-controlled reflection.

The HTTP API already applies JWT authentication, RBAC and owner checks before
calling protected handlers. Direct gRPC access must not be treated as trusted
only because it is a second application listener.

## Decision

Direct gRPC is a private authenticated internal interface. It is not a public
edge API.

Protected deployments must use all of the following controls before exposing
the gRPC listener beyond loopback:

- Mutual TLS for the direct gRPC transport.
- Server certificate validation by clients using an explicit server name.
- Client certificate validation by the server against a private internal CA.
- Application identity metadata for end-user or service-principal decisions.
- RBAC and resource-owner authorization equivalent to the HTTP API.
- Reflection disabled by default and enabled only in explicit development mode.

Until these controls are implemented, protected deployments must either keep
`GRPC_ENABLE=false` or bind `GRPC_ADDR` to loopback only.

## Rejected Alternatives

- Plaintext internal gRPC based only on network placement: rejected because a
  misconfigured listener becomes an authentication and authorization bypass.
- Public direct gRPC with only bearer tokens: rejected for this project because
  the direct interface exists as an internal transport and does not need public
  edge exposure.
- Loopback-only as the final model: rejected because planned integration and
  load-test work needs a deployable internal transport model beyond one process
  boundary.

## Reflection Policy

Reflection is a development tool, not a protected-environment default.

The implementation that follows this ADR must introduce explicit reflection
configuration with these semantics:

- Default: reflection disabled.
- Development: reflection may be enabled explicitly for local tooling.
- Protected environments: reflection remains disabled unless a future reviewed
  operations exception documents the exposure and compensating controls.

## Identity and Authorization

mTLS authenticates the calling workload. It does not by itself authorize access
to user-owned resources.

Direct RPCs that read or mutate user-owned resources must carry authenticated
application identity metadata, such as a verified bearer JWT or an internal
identity envelope produced from an already-authenticated HTTP request. The gRPC
server must map that identity into a typed request identity and apply the same
RBAC and owner/admin rules as the HTTP route for the same resource.

The HTTP-to-gRPC bridge may act as an internal workload authenticated by mTLS,
but it must preserve the original verified caller identity for authorization.
Direct external callers must not gain broader access than the HTTP API.

## Certificate and Caller Trust Boundaries

- The gRPC server certificate and private key are runtime secrets and must not
  be committed to the repository.
- Client certificates must be issued by the configured private client CA.
- Server-name verification is required; clients must not use insecure skip
  verification.
- Certificate subject/SAN values identify workloads, not end users.
- Trust in proxy headers, request IDs or caller-supplied metadata does not cross
  from HTTP to gRPC unless it is explicitly sanitized and re-issued by trusted
  application code.

## Implementation Acceptance Criteria for Follow-Up Tasks

TASK-019 must verify:

- Protected-mode gRPC startup fails when TLS/mTLS configuration is incomplete.
- Plaintext direct gRPC access is impossible in protected mode.
- The gRPC client validates the configured server name.
- The gRPC server validates client certificates against the configured client CA.
- Reflection follows the policy above and is off by default.

TASK-020 must verify:

- Direct unauthenticated `GetJob` is rejected.
- mTLS without valid application identity is not enough to read user-owned jobs.
- A non-admin user cannot read another user's job.
- Admin access remains available.
- HTTP bridge calls preserve and enforce the original HTTP caller identity.

TASK-021 must verify:

- Security-relevant metadata is appended without dropping existing metadata.
- Inbound request IDs are sanitized before trust or propagation.
- Generated request IDs are returned to callers through response metadata.
- Authentication, permission, deadline, cancellation, unavailable and not-found
  outcomes map to deterministic gRPC and HTTP bridge statuses.

## Consequences

The current code is explicitly treated as a local-development/direct-loopback
state for gRPC. This ADR does not implement mTLS, authentication or RBAC by
itself; it records the required deployable model and gives follow-up tasks
concrete security criteria.
