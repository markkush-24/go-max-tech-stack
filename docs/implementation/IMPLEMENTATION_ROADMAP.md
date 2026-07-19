# Implementation Roadmap

This roadmap is executable only through the task cards in `tasks/`. Codex receives one card at a time. A later phase may begin only when its blocking P0/P1 dependencies are complete or explicitly accepted/deferred.

## P0 — Repository reproducibility and execution baseline

Make a clean checkout buildable, testable and safe to hand to agents.

Tasks: **7** (`TASK-001`–`TASK-007`).

Exit gate: a clean checkout on Go 1.25.12 can run the declared quality commands and generated artifacts are reproducible.

## P1 — Critical correctness, lifecycle and security

Remove defects that can corrupt state, bypass security or make shutdown results untrustworthy.

Tasks: **15** (`TASK-008`–`TASK-022`).

Exit gate: state transitions, worker/runtime lifecycle, shutdown outcome and direct gRPC security no longer contain known P0 defects.

## P2 — Observability foundation

Introduce stable logging, tracing and metric ownership without changing business behavior.

Tasks: **17** (`TASK-023`–`TASK-039`).

Exit gate: request/trace propagation, DI-owned metrics and the local telemetry pipeline work end to end without affecting readiness.

## P3 — SLO, dashboards and telemetry hardening

Turn reliable signals into operational views, alerts and runbooks.

Tasks: **7** (`TASK-040`–`TASK-046`).

Exit gate: SLI rules, alerts, dashboards and runbooks are provisioned from the repository and backed by tested metrics.

## P4 — Async durability and resilience

Make accepted asynchronous work recoverable and dependencies safely degradable.

Tasks: **16** (`TASK-047`–`TASK-062`).

Exit gate: accepted async work is recoverable, dependency failure is bounded and retry/idempotency behavior is explicit.

## P5 — HTTP and streaming contracts

Normalize API, browser, SSE and caching behavior.

Tasks: **13** (`TASK-063`–`TASK-075`).

Exit gate: HTTP, browser, SSE and Range contracts are stable and covered by focused tests.

## P6 — OpenAPI and quality system

Make contracts and high-risk behavior reproducibly verifiable in CI.

Tasks: **12** (`TASK-076`–`TASK-087`).

Exit gate: OpenAPI, integration tests, fuzzing, benchmarks and CI gates reproduce the project contract from a clean checkout.

## P7 — Performance and failure laboratory

Build repeatable high-load, saturation and fault experiments.

Tasks: **11** (`TASK-088`–`TASK-098`).

Exit gate: repeatable load/failure experiments produce comparable evidence and a reviewed capacity baseline.
