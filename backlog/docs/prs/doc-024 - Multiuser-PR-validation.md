---
id: doc-024
title: Multiuser PR validation
type: other
created_date: '2026-11-12T23:40:00Z'
updated_date: '2026-11-12T23:40:00Z'
tags:
  - pr
---
> Importado de `docs-dev/PR_VALIDATION.md`. Original removido em 12/11/2026; consulte o histórico do Git.

# Multiuser PR validation

Version: **Koito 0.3.2-SUM-RC2**. Validation date: **2026-11-01**.
Environment: macOS arm64, Go 1.26.3, Node 24.15.0.

## Documentation

Added the Listening profiles guide and its sidebar entry. Updated installation,
scrobbler setup, Navidrome setup, imports, editing and configuration references.
The guides cover public access, administrator-only management, key ownership,
profile URLs, shared metadata, SQLite 2 → 3 upgrades and backup-based rollback.
They also explain per-profile forwarding, visible administrator tokens,
non-persisting connection tests, disabling/removing saved credentials and the
one-time behavior of legacy environment variables. README links to the guides;
the Portuguese and English design documents retain the implementation details.

Frontmatter, internal documentation routes and JavaScript configuration syntax
were checked successfully.

## Checks

| Command | Result |
| --- | --- |
| `make check` | Passed: Go vet, full Go suite, frontend type checking and all six frontend tests; no warnings in the output |
| `cd client && corepack yarn run build` | Passed; no warnings in the output |
| `docker build --check .` | Passed: “Check complete, no warnings found” |
| Documentation `astro build` (temporary copies, telemetry disabled) | Passed: current documentation generated 9 pages; base `a079fa6` generated 8 pages; both emitted the same existing DEP0190 warning |
| `git diff --check` | Passed |

Removed the Node experimental API from the frontend test harness by using the
already-installed TypeScript compiler. No warnings are suppressed and no new
dependency is required. Corrected the malformed JSON struct tag in
`engine/long_test.go`, also present in base commit `a079fa6`. Added Go vet to
`make check` so the same diagnostic is checked in future CI runs.

## Existing tooling warnings and scope

Yarn Classic 1.22.22 reports missing-license metadata for the documentation
package, legacy engine/workspace and peer-dependency metadata, and Node's DEP0169
deprecation for its use of `url.parse()`. These are
installation/tooling warnings from the existing dependency setup, not new
multiuser build diagnostics. `docs/package.json` and `docs/yarn.lock` remain
unchanged. The documentation dependency download was moved to a temporary copy
using the same pinned versions and integrity values, changing only the registry
hostname in that copy.

The documentation build also emits Node DEP0190 from a dependency's shell-based
child process. The unchanged base documentation at `a079fa6`, built with the same
installed dependencies and Node version, emits exactly the same warning. Neither
build introduces a new warning category or count. This existing warning remains
visible; dependencies and warning settings were not changed to conceal it.

The warning check is based on the commands and environment above. It does not
claim that every dependency, platform or configuration is warning-free. No PR,
release, image or deployment was published as part of this verification.
