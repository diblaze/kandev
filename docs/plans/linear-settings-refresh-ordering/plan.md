---
created: 2026-10-22
status: done
requirements:
  - REQ-UI-SETTINGS-MANUAL-SAVE-001
system_design:
  - ../../specs/ui/system-design/settings-manual-save.md
legacy_specs: []
---

# Implementation Plan: Linear settings refresh ordering

## Overview

Preserve acknowledged connection config/health against pending reads, then prove
ordinary draft and later refresh behavior through real consumers. Source-first
parent evidence at immutable base `36a12f5ff99e4095116a823d937fef2a205a671a`
proved old 200 and 204 regressions, with old 500 control passing.

## Scope

Only mounted Linear settings publication, initial seeding, and lifecycle fencing.
Exclude backend, credentials, API/schema, other integrations, watchers, teams
redesign, shared Query migration, global resources, and new revision frameworks.

## Technical approach

Adapt Jira's bounded local owner/sequence pattern into
`components/linear/use-linear-config-publication.ts`. Gate every config-read
publication and fence pending reads only at accepted mutation ACK. Keep separate
Linear baseline/form semantics. Start the existing interval after initial load.

### UI-01: Linear connection after Save and stale read settlement

Entry: `/settings/workspaces/:workspaceId/integrations/linear`.
Existing shared composition on desktop and phone:

```text
Linear connection
[Accepted auth health banner]
API key [saved mask / raw draft]
Default team [selected team]
[Test connection]  [Remove configuration]
           [Reset] [Save changes] (only while dirty)
```

Before: an older GET can clear saved mask/removal or revert health after Save.
After: accepted metadata remains until a legitimate later read. Phone entry is
its dedicated settings page; Remove opens the shipped bottom sheet. Reuse the
single settings scroll owner and fixed safe-area Save surface. Structure and
state are required; spacing is illustrative. No new responsive layout.

## Tests

`linear-settings-refresh.test.tsx` maps AC-UI-SETTINGS-MANUAL-SAVE-001.4 to
transport-controlled real-provider/coordinator/API tests: stale 200/204/500,
overlapping reads, Save/Remove during reads, failure, raw draft/canonical ACK,
Reset/next Save, slow initial load, workspace/unmount, and StrictMode.

## E2E tests

`linear-settings-refresh.spec.ts` (chromium) and
`mobile-linear-settings-refresh.spec.ts` (mobile-chrome) prove the same AC with
real disposable backend/mock Linear and captured pre-save response transport.
Own unique workspace/repo and explicit Worktree executor; guard seed and owned
source HEAD/full raw porcelain, aggregate cleanup failures, remove owned data.
Capture and inspect shipped desktop and phone screenshots.

## Work orders

- [x] [Task 01: Preserve saved config](task-01-preserve-saved-config.md)

## Verification

```bash
(cd apps/web && pnpm exec vitest run components/linear/linear-settings-refresh.test.tsx components/linear/linear-settings.test.tsx lib/api/domains/linear-api.test.ts --maxWorkers=1)
(cd apps/web && GOMAXPROCS=2 GOFLAGS=-p=2 pnpm e2e:run --host --shards 1 --project chromium tests/integrations/linear-settings-refresh.spec.ts --retries=0)
(cd apps/web && GOMAXPROCS=2 GOFLAGS=-p=2 pnpm e2e:run --host --no-build --shards 1 --project mobile-chrome tests/integrations/mobile-linear-settings-refresh.spec.ts --retries=0)
(cd apps/web && NODE_OPTIONS=--max-old-space-size=8192 pnpm run typecheck)
(cd apps/web && pnpm exec eslint components/linear/linear-settings.tsx components/linear/use-linear-config-publication.ts components/linear/linear-settings-refresh.test.tsx e2e/tests/integrations/linear-settings-refresh-helpers.ts e2e/tests/integrations/linear-settings-refresh.spec.ts e2e/tests/integrations/mobile-linear-settings-refresh.spec.ts)
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
git diff --check
```
## Verification results

- Permanent real-consumer/API suite: 49 passed across three files (21 refresh
  ordering/lifecycle cases, 4 existing removal cases, 24 API cases), one worker.
  Final command ran after all production and test corrections.
- Corrected pre-fix permanent RED: 11 expected assertion failures and 6 passing
  controls. Parent RED remains 2 expected failures and 1 passing 500 control.
  Initial mixed test-scaffold failures are retained separately, not causal RED.
- Browser RED: desktop accepted-health reversion and phone old-204 removal loss
  both failed as intended on the pre-fix production build.
- Fresh managed desktop browser: 1 passed (4.0s test, 7.0s total). Phone on the
  same unchanged build: 1 passed (4.7s test, 7.4s total). One worker/shard,
  retries 0, GOMAXPROCS=2, GOFLAGS=-p=2; both screenshots inspected.
- Unique disposable workspace/repository/Worktree executor cleaned up. No tasks
  or sessions started. Seed, source, and owned repository HEAD/full raw porcelain
  guards passed. All four backend fixture directories absent after teardown.
- ESLint with zero warnings and Prettier checks passed for all six changed TS files.
- Full frontend typecheck passed with the documented 8GB Node heap. Default-heap
  exhaustion and an intermediate test-only type-option failure are separately
  retained in the external receipts. No product incident claim.
- Documentation catalog: 397 decisions and 1712 specifications validated.
  Full specification lint and PR-documentation coverage preflight passed.
- Public-doc assessment: existing workspace credential/manual Save/blank-key
  contract is unchanged; no public guide or screenshot catalog changes needed.
- Duplicate target patches refreshed after causal proof: #1512 has 895 files,
  Query config writes without pending-read cancellation/admission fencing;
  #4375/#4584 have no target changes; #2336 changes watcher tests only.
  All four heads remained unchanged at the final metadata refresh.
- Diff whitespace and package cross-references checked. Source commit and active
  hook receipt are supplied in the external parent-validation packet. No push/PR
  is authorized at this checkpoint.


## Risks

Initial seeding must remain authoritative despite slow transport. Read sequence
must never reject a successful write ACK. Failed writes must not fence reads.
Public docs retain their existing Save/blank-key contract; no new user operation.
