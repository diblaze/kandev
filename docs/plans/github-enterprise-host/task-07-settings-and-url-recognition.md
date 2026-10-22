---
id: "07-settings-and-url-recognition"
title: "Settings and URL recognition for GitHub hosts"
status: pending
wave: 4
depends_on:
  - "03-host-routed-operations"
plan: "plan.md"
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007
acceptance_criteria:
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.4
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.6
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.1
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.2
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.1
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.2
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.3
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.4
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.5
system_design:
  - ../../specs/integrations/system-design/github-enterprise-host.md
---

# Task 07: Settings And URL Recognition For GitHub Hosts

## Summary

Render one connection card per GitHub host with an action to connect another,
add a host field to the PAT and App forms, send `host` on every GitHub API
call that names a repository, recognize URLs on connected hosts everywhere a
GitHub URL is accepted, and build links on the item host.

## ASCII UI preview

Excerpts of [plan.md](plan.md#ascii-ui-preview) UI-01 to UI-04, same labels.

```text
UI-01 desktop
| github.com      GitHub CLI . alice     Active  [Change] [Disconnect] |
| octo.ghe.com    GitHub App . octo-org  Active  [Change] [Disconnect] |
| [+ Connect another GitHub host]                                       |

UI-01 phone
| github.com            [..] |
| GitHub CLI . alice         |
| [+ Connect another host]   |

UI-02
| Host:    [ github.com                     ]  |

UI-03
| Host: [ octo.ghe.com v ]   Search: [ is:open ...     ] |

UI-04
| ghe.example.com is not connected. [Open GitHub settings] |
```

## In scope

- Status types, API clients, and a connected-hosts hook.
- Settings cards, connection dialog, PAT host field, CLI picker, App create
  and import host fields, onboarding links.
- Host-aware URL parsers and allowlists and their consumers.
- Host selector on the GitHub dashboard and search when two or more hosts are
  connected.
- Profile, quick-task, remote-repository, and file links.
- Locale strings in all seven catalogs.

## Out of scope

- Example placeholders that show `github.com` URLs (`issue-watch-placeholders.ts:13`, `review-watch-placeholders.ts:17`, `action-presets-section.tsx:40`, `task-github-issue-dialog.tsx:23`, `task-github-pr-dialog.tsx:24`, Office placeholders): they stay valid examples.
- Kandev's own links (`lsp-language-options.ts`, `notifications-settings-external-providers.tsx`, browser demo fixtures, license generation).

## Acceptance

- With `github.com` and `octo.ghe.com` connected, settings show two cards on desktop and phone; disconnecting one leaves the other.
- The PAT form sends `host`; the App create and import forms send `github_host`.
- Pasting `https://octo.ghe.com/org/repo/pull/7` into task create, the PR link dialog, and a workspace source is recognized; pasting a `ghe.example.com` URL shows UI-04.
- Author links on an `octo.ghe.com` PR open `https://octo.ghe.com/<login>`.

## Changes

### `apps/web/lib/types/github.ts`
**What:** modified.
**Justified by:** E35
**Governed by:** `apps/web/AGENTS.md` TypeScript limits.
**Specifics:** add `GitHubHostStatus` (`github_host`, `source`, `login`, `installation_account_login`, `status`, `last_error`, `personal_login`) and `hosts: GitHubHostStatus[]` on `GitHubStatusResponse`; `ReviewWatch`, `IssueWatch`, `TaskPR`, and `RepoFilter` gain `github_host` or `host`; `GitHubPersonalConnection` gains `github_host`.

### `apps/web/lib/api/domains/github-auth-api.ts` and `github-api.ts`
**What:** modified.
**Justified by:** E29, E35
**Governed by:** `apps/web/AGENTS.md`.
**Specifics:** `fetchGitHubStatus(workspaceId, options, refreshRateLimit, host?)` (github-auth-api.ts:15); delete connection takes `host`; every helper in `github-api.ts` that names an owner and repository, plus `fetchAccessibleRepos`, `listUserOrgs`, `searchOrgRepos`, `searchUserPRs`, and `searchUserIssues`, takes an optional `host` and sends it only when it is not `github.com`. `AccessibleRepo` (line 311) gains `host`.

### `apps/web/hooks/domains/github/use-github-hosts.ts`
**What:** added.
**Justified by:** E33, E35
**Governed by:** `apps/web/AGENTS.md` hook conventions.
**Specifics:** `useGitHubHosts(workspaceId): string[]` returns `["github.com", ...connected hosts]` from the status query used by `use-github-status.ts`, without a second request. A sibling `use-github-hosts.test.ts` covers empty, single, and multiple hosts.

### `apps/web/lib/github/host.ts`
**What:** added.
**Justified by:** E33, E34
**Governed by:** `apps/web/AGENTS.md`.
**Specifics:** `normalizeGitHubHost(raw)` mirroring the backend rule; `githubWebOrigin(host)`; `hostFromProviderHost(providerHost)`; `hostFromUrl(raw)`; `isKnownGitHubHost(host, hosts)`. `host.test.ts` covers the same cases as the backend host tests.

### `apps/web/lib/github/parse-url.ts`, `lib/utils/github-repo-url.ts`, `hooks/domains/github/use-pr-info-by-url.ts`, `components/task/task-github-pr-url.ts`
**What:** modified.
**Justified by:** E33
**Governed by:** `apps/web/AGENTS.md`; existing parser tests.
**Specifics:** each parser takes `hosts: readonly string[]` (default `["github.com"]`) and returns `host` in its result. `parseGitHubRepoUrl` (parse-url.ts:16; github-repo-url.ts:15) accepts `https://<host>/...` and `<user>@<host>:...` for listed hosts; `buildGitHubRepoUrl` builds on `parts.host`. `parseGitHubPrUrl`, `parseGitHubIssueUrl`, `parseGitHubAnyUrl` (use-pr-info-by-url.ts:63-100) and `parseGitHubPullRequestURL`, `pullRequestPayload` (task-github-pr-url.ts:40-90) do the same; the inferred-number path builds on the repository host. Add `matchGitHubShapedUrl(raw)` returning a host for GitHub-shaped paths so callers can show UI-04 for an unlisted host. Extend the sibling `*.test.ts` files.

### `apps/web/components/workspace-source-picker/remote-url.ts`
**What:** modified.
**Justified by:** E33
**Governed by:** `apps/web/AGENTS.md`.
**Specifics:** `looksLikeSupportedRemoteURL(value, githubHosts)` (line 17) accepts listed GitHub hosts in both URL and SCP forms with any SCP user; callers in `workspace-source-state.ts` pass `useGitHubHosts`.

### Parser consumers
**What:** modified.
**Justified by:** E33
**Governed by:** `apps/web/AGENTS.md`.
**Specifics:** pass `useGitHubHosts(workspaceId)` and forward the parsed `host` to API calls in `components/task-create-dialog-effects.ts`, `task-create-dialog-helpers.ts`, `task-create-dialog-remote-repo-chip.tsx`, `task-create-dialog-remote-repo-identity.ts` (lines 93-94 accept listed hosts), `components/workspace-source-picker/workspace-source-state.ts`, `hooks/domains/github/use-branches-by-url.ts` (lines 311-323), `hooks/domains/integrations/use-remote-repositories.ts` (line 167 builds on `repo.host`), `hooks/domains/settings/use-workflow-sync.ts` (lines 20, 75-80 store `github_host`), `components/task/task-github-pr-dialog.tsx`, `task-github-issue-dialog.tsx`, the review and issue watch dialogs (store `github_host`), and `components/github/my-github/quick-task-launcher.tsx` (lines 84-87 build on the PR host; line 162 builds `githubUrl` on the host). Each shows UI-04 for a GitHub-shaped URL on an unlisted host, linking to the GitHub settings route.

### `apps/web/components/github/pr-shared.tsx` and `pr-detail-panel.tsx`
**What:** modified.
**Justified by:** E34
**Governed by:** `apps/web/AGENTS.md`.
**Specifics:** `AuthorAvatar` and `AuthorLink` (pr-shared.tsx:52-80) take `host`; `githubPerson(login, host, ...)` (pr-detail-panel.tsx:248) builds `githubWebOrigin(host)/<login>`; callers pass `hostFromProviderHost` of the PR's repository or `hostFromUrl(pr_url)`.

### `apps/web/lib/utils/external-vcs-file-url.ts` and `hooks/domains/workspace/use-external-vcs-file-link.ts`
**What:** modified.
**Justified by:** E34
**Governed by:** `apps/web/AGENTS.md`.
**Specifics:** `sshRemoteForRepository` (lines 207-218) and `githubRemoteMatches` (lines 254-264) accept the repository's `provider_host` for GitHub instead of only `https://github.com`, mirroring the GitLab branch; `githubPRMatches` (use-external-vcs-file-link.ts:182) compares the PR host when the PR has no repository ID.

### `apps/web/lib/sidebar/repository-rule-identity.ts`
**What:** unchanged.
**Justified by:** E34
**Governed by:** n/a.
**Specifics:** `defaultProviderHost` (line 188) applies only when `provider_host` is empty, which never happens for enterprise rows.

### Settings components
**What:** modified.
**Justified by:** E2, E25, E35
**Governed by:** `/mobile-parity`; `CLAUDE.md` Internationalization; shadcn via `@kandev/ui`.
**Specifics:**
- `components/github/github-status.tsx` (lines 58-211): render a list of `GitHubHostConnectionCard` from `status.hosts`, each with the existing automation and personal content for its host (fetch per-host status lazily on expand), and a "Connect another GitHub host" button. Phone layout stacks cards and moves actions into a row menu.
- `components/github/github-connection-dialog.tsx` and `github-connection-settings-form.tsx` (line 96): take a `host` prop for Change; for a new host, the PAT method shows a host input; the CLI method lists all accounts and disables hosts that already have a connection; the App method shows the registration's host.
- `components/github/github-pat-form.tsx`: add a `Host` input (default `github.com`) validated with `normalizeGitHubHost`.
- `components/github/github-cli-form.tsx` (lines 121-122): keep listing every host; mark already-connected hosts.
- `components/github/github-app-create-form.tsx` and `github-app-import-form.tsx` (line 78): a host field sent as `github_host`; `github-app-registration-list.tsx` shows each registration's host.
- `components/github/github-app-onboarding-model.ts` (lines 144-147): settings links on the registration host.
- `components/github/my-github/*`: a host `Select` when `useGitHubHosts` has two or more entries; results show the host.
- Locale keys in `src/locales/<locale>/github.json` for all seven locales; no em dash.

### Tests
**What:** added and modified.
**Justified by:** E33, E34, E35
**Governed by:** `CLAUDE.md` Testing; `/e2e` and `/mobile-parity`.
**Specifics:**
- Unit tests next to each changed parser, hook, and helper.
- `components/github/github-status.test.tsx` and `github-connection-dialog.test.tsx`: two hosts render two cards; PAT host field is sent.
- `e2e/tests/integrations/github-multi-host-settings.spec.ts`: connect a second host with the mock GitHub backend, see two cards, disconnect one; paste an enterprise PR URL into the PR link dialog; paste an unconnected host URL and see UI-04.
- `e2e/tests/integrations/mobile-github-workspace-settings.spec.ts`: two-card phone layout and the row menu.
- The mock GitHub controller (`apps/backend/internal/github/mock_controller.go`) accepts `host` so the E2E can create a second connection.

## Verification

```bash
# From apps/web:
pnpm run typecheck
pnpm test -- lib/github components/github hooks/domains/github components/workspace-source-picker components/task
pnpm run lint
pnpm run i18n:check
pnpm run i18n:ratchet
pnpm e2e:run -- e2e/tests/integrations/github-multi-host-settings.spec.ts e2e/tests/integrations/mobile-github-workspace-settings.spec.ts
```

## Dependencies

Task 03 (`host` on routes, `hosts` in status).

## Risks

- Parsers are used in many components; a consumer that does not pass
  `useGitHubHosts` keeps `github.com`-only recognition. `rg -n 'parseGitHub\w*Url\(|looksLikeSupportedRemoteURL\(' apps/web --glob '!*.test.*'` lists them.

## Parallelism

`parallel` with tasks 04 and 05.

## Results

Pending.
