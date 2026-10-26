---
id: "07-settings-and-url-recognition"
title: "Settings and URL recognition for GitHub hosts"
status: done
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

Implemented test-first.

What was done:

- Host helpers (`lib/github/host.ts`): `normalizeGitHubHost` (backend rule,
  null for an invalid host), `githubWebOrigin`, `hostFromUrl`,
  `hostFromProviderHost` (empty means github.com, malformed is null),
  `isKnownGitHubHost`, `githubSSHUser` (tenant name on `.ghe.com`, `git`
  elsewhere), and `setGitHubHostParam`, which sends `host` only for a host
  other than github.com and forwards a malformed host unchanged so the backend
  rejects it.
- Types: `GitHubHostStatus`, `GitHubStatus.hosts`, `github_host` on `TaskPR`,
  watches, watch create requests, `GitHubPersonalConnection`, workflow sync
  config and request; `RepoFilter.host`; `StartGitHubAppManifestRequest.github_host`.
- API clients: status, workspace disconnect, personal connect and disconnect,
  PR info/feedback/status/batch statuses/review/merge/reviewers, issue info,
  branches, merge methods and access, orgs, org repo search, accessible repos
  (tagged with `host`), user PR and issue search, and task issue links take a
  host.
- `useGitHubHosts(workspaceId)`: github.com plus every connected host, read
  from the loaded workspace status in the store (no request of its own; outside
  a store it returns github.com only).
- URL recognition: every GitHub parser takes the connected hosts and returns
  the host (`parse-url.ts`, `github-repo-url.ts`, `use-pr-info-by-url.ts`,
  `task-github-pr-url.ts`); `matchGitHubShapedUrl` and `unconnectedGitHubHost`
  name the host of a pull-request or issue URL on a host without a connection.
  `looksLikeSupportedRemoteURL` accepts listed hosts in URL and SCP form with
  any SCP user. Branch listing, PR and issue metadata, duplicate-repository
  checks, the first-row URL error, workspace source validation, remote
  repository identity (SCP and HTTPS of one host now dedupe), the PR and issue
  link dialogs, the add-remote-repository dialog, and workflow sync (stores
  `github_host`) use them. Remote repository browsing lists every active
  connected host with URLs and `providerHost` on that host.
- UI-04: `GitHubHostNotConnectedNotice` ("<host> is not connected." plus an
  "Open GitHub settings" link to the workspace GitHub settings) in the task
  create and workspace source remote picker, the PR link dialog (through a new
  optional `inputNotice` on `TaskChangeRequestLinkForm`, which blocks submit
  while it shows), and the issue link dialog.
- Links: author, reviewer, and comment profile links and the PR detail mapping
  use the PR host (`taskPRWebHost`); file links accept the repository
  `provider_host` for GitHub over HTTPS and SSH (tenant SSH user required on
  `.ghe.com`); `githubPRMatches` compares hosts when the PR has no repository
  ID; the quick-task launcher matches repositories and builds the fallback URL
  on the item host. PR feedback, merge methods, merge access, merge, review, and
  re-request calls send the task PR `github_host`, and the feedback and merge
  method caches key by host.
- Settings (UI-01, UI-02): one card per host (`github-host-card`): the existing
  github.com card plus `GitHubHostConnectionCard` per enterprise host (source,
  login or installation account, status, last error, personal login, Change,
  Disconnect for that host), and "Connect another GitHub host". On phones the
  enterprise card actions move into a row menu (44px trigger) that opens the
  connection drawer. The connection dialog takes `host` (Change) or `addHost`;
  the token form has a "GitHub host" field defaulting to github.com, validated
  like the backend; the CLI picker marks and disables accounts on hosts that
  already have a connection, except the host being changed.
- App flows: create and import forms have a host field sent as `github_host`;
  App settings links use the host; the registration list shows each host.
- GitHub page (UI-03): a host selector when two or more hosts are connected;
  search and PR statuses use the selected host; enterprise results show their
  host in the row metadata. Review and issue watch dialogs have a host field
  (shown with two or more hosts) and store `github_host`; the repository filter
  lists organizations and repositories on that host.
- Locale keys in all seven catalogs (zh-hk and zh-tw through
  `pnpm run i18n:zh-hant`) and the pseudo catalog.
- E2E: `github-multi-host-settings.spec.ts` (two cards and host-scoped
  disconnect; connect a second host with a token; PR link dialog recognizes an
  `octo.ghe.com` URL and shows UI-04 for `ghe.example.com`) and a phone case in
  `mobile-github-workspace-settings.spec.ts` (stacked cards, row menu, 44px
  target, no horizontal overflow). The API client mock connection helpers take
  an optional host.

Verification (from `apps/web`, `NODE_OPTIONS=--max-old-space-size=4096`,
`--maxWorkers=2`, one heavy command at a time):

- `pnpm run typecheck`: exit 0 (tsc needed `--max-old-space-size=8192`; it
  aborts with an out-of-memory error at 4096 on this tree, also before this order).
- `pnpm vitest run --maxWorkers=2` (full suite): `Test Files 3 failed | 3067 passed (3070)`,
  `Tests 3 failed | 27974 passed | 4 skipped`. The three failures are outside
  this order and untouched by it: `lib/sidebar/sidebar-archived-update-freshness.test.ts`
  and `lib/logger/indexeddb-store.test.ts` timed out under load and pass alone;
  `e2e/helpers/seed-repository-origin.test.ts` fails alone too (the local Git
  reports `reference already exists` where the test expects `cannot lock ref`).
- The affected-area run (`components/task components/task-create components/workspace-source-picker components/settings components/integrations components/github hooks lib/github lib/utils lib/api/domains app/github`):
  `Test Files 1619 passed (1619)`, `Tests 14742 passed | 4 skipped`.
- `pnpm run lint`: exit 0, no warnings.
- `pnpm run i18n:check`: all six checks pass; ja, ko, pt-pt, zh-cn, zh-hk,
  zh-tw complete, pseudo in sync.
- `pnpm run i18n:ratchet`: `0 added + 68 modified file(s) clean`, guard
  allowlist intact.
- `pnpm e2e:run --host -- tests/integrations/github-multi-host-settings.spec.ts`: `3 passed`.
- `pnpm e2e:run --host --no-build --project mobile-chrome -- tests/integrations/mobile-github-workspace-settings.spec.ts`: `4 passed`.
- Regression: `github-workspace-settings.spec.ts` and `github-authentication.spec.ts`
  `10 passed`; `mobile-github-auth-settings.spec.ts` `3 passed`.
  The E2E runs used the harness's own backend (ports 18080+, temporary HOME);
  the Playwright browser was installed into `apps/web/node_modules/.ms-playwright`
  through `PLAYWRIGHT_BROWSERS_PATH`.

Tests that passed on their first run, then proven by breaking the code: the CLI
picker host test (preferred account no longer skipped taken hosts) and the
toolbar host selector test (selector shown with one host). The second CLI test
and "does not recognize URLs on a host that is not connected" are guards that
passed before the change by construction.

Deviations:

- `useGitHubHosts` reads the store with `useOptionalAppStore` instead of calling
  `useGitHubStatus`: many hook tests run without a `StateProvider`, and the
  status is already loaded by the navigation availability hook and every
  GitHub surface. Hosts reach the task-create submit and effects through
  `DialogFormState.githubHosts`, and the remote picker through
  `UseRemoteRepositoriesResult.githubHosts`, which derives them from its own
  status read. Five test mocks of `@/components/state-provider` gained
  `useOptionalAppStore`.
- `mergePR(workspaceId, ref, mergeMethod)` and `requestPRReviewers(ref, reviewers, workspaceId)`
  take a pull-request ref object instead of positional owner, repo, and number,
  to stay within the five-parameter limit; callers and tests are updated.
- The token form sends `host` only for a host other than github.com, so the
  github.com request is unchanged.
- `matchGitHubShapedUrl` matches pull-request and issue paths only: a bare
  `https://<host>/<owner>/<repo>` is equally a GitLab or other self-hosted
  repository, so an unlisted repository URL keeps the existing invalid-URL
  message instead of UI-04 (workflow sync included).
- Per-host detail on expand (permissions, rate limits, personal OAuth connect
  for an enterprise host) is not built: an enterprise card shows what the
  `hosts` entry carries; personal connection management stays on the
  github.com projection.
- Results show their host in the row metadata for enterprise hosts only; the
  selector names the host for the whole result list.
- `review-watch-dialog.tsx` moved its form state helpers to
  `review-watch-form.ts`, and `task-create-dialog-helpers` host tests live in a
  new file, to stay within the file length limit.
- The mock GitHub controller needed no change: task 02 already gave its
  connection routes a `host` query parameter.
- A pasted URL that `usePRInfoByURL` already recorded as "no info" before the
  workspace status loaded is not re-parsed when the hosts arrive; the
  navigation hook loads the status at startup, so this needs a paste in the
  first moments after boot.

Unverified: a live enterprise host (task 08); `issue-watch-dialog` host storage
has no own test (it shares `GitHubWatchHostField` and the payload shape with the
review watch dialog, which is tested).
