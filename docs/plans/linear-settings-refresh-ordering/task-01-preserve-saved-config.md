---
id: "01-preserve-saved-config"
title: "Preserve saved Linear connection config"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-UI-SETTINGS-MANUAL-SAVE-001
acceptance_criteria:
  - AC-UI-SETTINGS-MANUAL-SAVE-001.4
system_design:
  - ../../specs/ui/system-design/settings-manual-save.md
---

# Task 01: Preserve saved Linear connection config

## Summary

Fence obsolete mounted reads after acknowledged writes and overlapping reads.
Preserve initial form seeding, raw drafts, baseline Reset, and fresh health reads.

## In scope

Local Linear publication, real-consumer regression and desktop/phone proof.

## Out of scope

Backend/API/credential changes, watchers, shared caches, and other integrations.

## Acceptance

- Stale reads cannot undo accepted Save/Remove; current reads still publish.
- Draft edits, errors, canonical ACK, Reset, and next Save retain existing semantics.
- Lifecycle/slow-seeding controls and disposable desktop/phone checks pass.

## ASCII UI preview

See [combined preview](plan.md#ui-01-linear-connection-after-save-and-stale-read-settlement).

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
state are required; spacing is illustrative. No new responsive layout. Save and Remove disable conflicting actions while
a same-route mutation is pending.

## Verification

```bash
(cd apps/web && pnpm exec vitest run components/linear/linear-settings-refresh.test.tsx components/linear/linear-settings.test.tsx lib/api/domains/linear-api.test.ts --maxWorkers=1)
(cd apps/web && GOMAXPROCS=2 GOFLAGS=-p=2 pnpm e2e:run --host --shards 1 --project chromium tests/integrations/linear-settings-refresh.spec.ts --retries=0)
(cd apps/web && GOMAXPROCS=2 GOFLAGS=-p=2 pnpm e2e:run --host --no-build --shards 1 --project mobile-chrome tests/integrations/mobile-linear-settings-refresh.spec.ts --retries=0)
(cd apps/web && NODE_OPTIONS=--max-old-space-size=8192 pnpm run typecheck)
(cd apps/web && pnpm exec eslint components/linear/linear-settings.tsx components/linear/linear-action-bar.tsx hooks/domains/linear/use-linear-config-publication.ts components/linear/linear-settings-refresh.test.tsx e2e/tests/integrations/linear-settings-refresh-helpers.ts e2e/tests/integrations/linear-settings-refresh.spec.ts e2e/tests/integrations/mobile-linear-settings-refresh.spec.ts)
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
git diff --check
```
## Files likely touched

- `apps/web/components/linear/linear-settings.tsx`
- `apps/web/components/linear/linear-action-bar.tsx`
- `apps/web/hooks/domains/linear/use-linear-config-publication.ts`
- `apps/web/components/linear/linear-settings-refresh.test.tsx`
- `apps/web/e2e/tests/integrations/linear-settings-refresh-helpers.ts`
- `apps/web/e2e/tests/integrations/linear-settings-refresh.spec.ts`
- `apps/web/e2e/tests/integrations/mobile-linear-settings-refresh.spec.ts`
- Existing settings-manual-save requirement/design pair.

## Dependencies

None.

## Risks

See plan. Teams are transport consumers, not config reads.

## Parallelism

sequential

## Inputs

Parent source packet and owning settings-manual-save requirement/design.

## Initial checkpoint results

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

## PR review remediation

Greptile identified the required domain-hook location. CodeRabbit identified
concurrent same-route Save and Remove admission. Move the unchanged publication
hooks to `hooks/domains/linear/`; use the existing coordinator `runExclusive`
for confirmed Remove, preventing overlap with Save and other Remove submissions.
Disable Remove and its confirmation while a mutation owns the route gate.
Keep raw drafts, retry behavior, accepted-write read fences, and existing
confirmation compositions. Add real-transport RED controls and native desktop/
phone busy-action proof before the bounded correction. Reuse this one work order.
Prior checkpoint results remain historical; new-head checks, assets, reviews,
and exact-main combined verification must be refreshed before merge.

## Review remediation verification

- Two additional real-consumer mutation admission assertions failed before the
  correction; all 21 existing refresh controls passed. The initial mixed timeout
  attempt is retained separately from the corrected two-assertion RED.
- Six focused suites passed: 86 tests, including 51 Linear publication/removal/API
  cases and coordinator/exclusive-action/enablement compatibility controls.
- Desktop and native phone each failed the pending-Save Remove-button assertion
  on the prior production build, then passed on the rebuilt correction. Both
  prove Save is disabled during confirmed Remove and allow ordinary subsequent
  operations. One worker/shard, retries zero, normal resource guards.
- Changed-file ESLint, frontend typecheck with the documented 8GB heap, and
  documentation catalog/spec lint passed. Final exact-head combined-tree,
  hosted review/CI, and publication evidence is retained externally; it is
  refreshed after this commit rather than represented by historical receipts.
