---
id: "08-docs"
title: "Documentation for GitHub hosts"
status: pending
wave: 6
depends_on:
  - "01-host-aware-workspace-client"
  - "02-per-host-connections"
  - "03-host-routed-operations"
  - "04-enterprise-repository-identity"
  - "05-enterprise-pr-issue-hosts"
  - "06-enterprise-task-credentials"
  - "07-settings-and-url-recognition"
plan: "plan.md"
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007
acceptance_criteria: []
system_design:
  - ../../specs/integrations/system-design/github-enterprise-host.md
---

# Task 08: Documentation For GitHub Hosts

## Summary

Update the public GitHub guide and the authentication specification so they
describe per-host connections, record the live end-to-end run, and promote
the new specification. No ADR is added; the rationale lives in the plan and
the system design.

## In scope

- `docs/public/integrations.md` GitHub section.
- The boundary and out-of-scope lines in `github-authentication-03.md` and the
  connection table in `github-authentication-01.md`.
- Status of the new requirement and system-design files.
- Results of the live run in the plan.

## Out of scope

- Screenshots, unless `/docs-maintainer` finds an existing GitHub settings
  screenshot that the new card layout invalidates.

## Acceptance

- The public guide explains how to connect several GitHub hosts in one workspace, with every connection source, and what stays `github.com` only.
- Spec catalog validation, spec lint, and public docs validation pass.
- Plan verification has recorded results.

## Changes

### `docs/public/integrations.md`
**What:** modified.
**Justified by:** E21
**Governed by:** `docs/public/README.md`; `/docs-maintainer`.
**Specifics:**
- Line 54: GitHub supports `github.com`, GitHub Enterprise Cloud with data residency (`<subdomain>.ghe.com`), and GitHub Enterprise Server; a workspace holds one connection per host.
- Line 84 (GitHub CLI bullet): Kandev stores the selected host/login pair; the command is `gh auth token --hostname <host> --user <login>`.
- Add a "GitHub hosts" subsection under **Authenticate**: connect each host from its card; PAT, CLI, and App work on every host; items route by their own host; a URL or repository on an unconnected host is rejected with a "not connected" message; connecting a host classifies existing repositories on it; GitHub CLI token variables per host kind (`GH_TOKEN` for `github.com` and `ghe.com`, `GH_ENTERPRISE_TOKEN` for Server).
- Line 233: GitHub Apps work on every host; each GitHub Enterprise Server instance needs its own App registration; the public HTTPS origin rule is unchanged.
- State that GitHub Enterprise Server behavior is covered by automated tests and the documented endpoint mapping when the live run had no Server instance.

### `docs/specs/integrations/system-design/github-authentication-03.md`
**What:** modified.
**Justified by:** E20
**Governed by:** `docs/specs/guide/traceability-and-lifecycle.md` (one editable source of truth).
**Specifics:** line 239 becomes "Multiple automation connections for one host in one workspace"; line 243 is removed and the section links to [GitHub Enterprise Host](github-enterprise-host.md) for host behavior.

### `docs/specs/integrations/system-design/github-authentication-01.md`
**What:** modified.
**Justified by:** E23
**Governed by:** `docs/specs/guide/system-design.md`.
**Specifics:** the connection table description (around line 240) says `github_host` is part of the connection key, and links to the new design.

### `docs/specs/integrations/requirements/github-enterprise-host.md` and `system-design/github-enterprise-host.md`
**What:** modified.
**Justified by:** E1 to E41
**Governed by:** `docs/specs/guide/traceability-and-lifecycle.md` status values.
**Specifics:** set `status: active` and `status: current` after tasks 01 to 07 are done and the live run passes; correct any section the implementation changed.

### `docs/plans/github-enterprise-host/plan.md`
**What:** modified.
**Justified by:** plan Verification.
**Governed by:** `docs/specs/guide/plans-and-work-orders.md` (results after implementation).
**Specifics:** set `status: complete`, record the five live checks and their outcome, which host kinds were exercised, and set every work order to `done` with results.

## Verification

```bash
# From the repository root:
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
node scripts/validate-public-docs.mjs
```

## Dependencies

Tasks 01 to 07.

## Risks

- The guide could promise Server behavior that was not exercised live; the
  last bullet above covers that.

## Parallelism

`sequential`

## Results

Pending.
