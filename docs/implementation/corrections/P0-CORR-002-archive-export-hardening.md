# P0-CORR-002 — Make repository export fail safely and scan all key risks

## Goal

Ensure a failed export never leaves a publishable unsafe archive and close known private-key/artifact bypasses.

## Review findings

- P0-03-F1 — unsafe final ZIP remains after content-scan failure.
- P0-03-F2 — files over 1 MiB and common key containers bypass scanning.
- P0-03-F3 — documented IDE/OS scratch files are not fully blocked.

## Allowed scope

- `scripts/export-archive.ps1`
- focused script tests/fixtures
- `.gitignore`
- `docs/repository-archive-hygiene.md`
- archive-related README section

## Forbidden scope

- no changes to application/runtime code;
- no generic secret-management platform;
- no unrelated repository cleanup.

## Requirements

1. Create the ZIP at a temporary path.
2. Validate paths and content before publishing to the requested output path.
3. On any error, remove the temporary file and do not leave/overwrite the final output.
4. Do not skip all content inspection merely because a file exceeds 1 MiB.
5. Scan large text entries incrementally with a bounded-memory approach.
6. Block common private-key containers including at least:
   - `*.ppk`;
   - `*.p12`;
   - `*.pfx`.
7. Align blocked path patterns with documented local artifacts, including at least:
   - `*.iml`;
   - `.DS_Store`;
   - `Thumbs.db`;
   - `*.swp`.
8. Add regression tests for every reproduced bypass.

## Acceptance criteria

- a marker-detected failure leaves no final archive;
- a >1 MiB file containing a private-key marker is rejected;
- `.ppk`, `.p12`, and `.pfx` tracked files are rejected;
- tracked `.iml`, `.DS_Store`, `Thumbs.db`, and `.swp` files are rejected;
- a valid committed tree exports successfully;
- two exports of the same treeish remain reproducible;
- `-Force` cannot leave an older valid archive replaced by an invalid one.

## Required verification

Run in disposable repositories:

```text
pwsh -File ./scripts/export-archive.ps1 -Force
# tracked blocked-path negative tests
# small and large private-key marker negative tests
# failure leaves no output assertion
# existing-output atomic replacement assertion
# same-tree two-export checksum comparison
go test ./scripts/...
git diff --check
```

## Required completion report

Report:

- temporary/atomic publish strategy;
- exact blocked extensions and paths;
- large-entry scanning strategy;
- negative-test results;
- changed files and remaining limitations.
