# Audit Traceability Model

## Why the lists have different sizes

Audit reports use three different levels of granularity:

1. **Finding** — one atomic, evidenced observation about the current code.
2. **Remediation group** — one coherent engineering outcome that may resolve several findings.
3. **Agent task** — one small implementation/review unit with bounded files and acceptance criteria.

The counts are therefore intentionally different:

- many findings can belong to one remediation group;
- one complex finding can require several agent tasks;
- a remediation group is not considered complete until every linked finding has an explicit disposition.

## Required final-roadmap fields

Audit 14 must produce a row for every finding with:

```text
Finding ID
Source audit
Status
Evidence
Risk
Remediation group
Agent task IDs
Verification command/test
Final disposition
```

Allowed final dispositions:

- `FIX`
- `ACCEPT_RISK`
- `DEFER`
- `NOT_APPLICABLE`
- `ALREADY_RESOLVED`

No finding may disappear merely because a shorter priority list groups it with other work.

## Example from Audit 12

Remediation group `API-CORS` includes at least:

- missing `Access-Control-Expose-Headers`;
- incomplete `Vary` on denial paths;
- global rather than route-specific CORS capability;
- tests for browser-visible `Location`, `ETag`, `X-Request-Id`, `Retry-After` and `Content-Range`.

The remediation group is one roadmap item, but implementation can be split into multiple agent tasks and all individual findings remain traceable.
