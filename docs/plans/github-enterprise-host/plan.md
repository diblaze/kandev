---
created: 2026-10-22
status: draft
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007
system_design:
  - ../../specs/integrations/system-design/github-enterprise-host.md
legacy_specs: []
---

# Plan: GitHub Enterprise Hosts With Full Parity

## Summary

A workspace can connect `github.com` and any number of GitHub Enterprise hosts
(GHE.com tenants and GitHub Enterprise Server) at once, and every GitHub
feature works on each host: connections of every source, routing by item host,
repository identity, pull requests and issues, App flows, task credentials,
links, and URL input. The need is stated directly: "one should be able to
connect any GH provider" and "anything supported by GH should be supported by
GHE"; today the first step already fails with `unsupported GitHub host` (E1)
and the connection table allows one row per workspace (E23). The largest risk
is the connection-table key change, which touches every credential resolution
(E27) and must migrate existing rows without behavior change.

## Context

Today:

1. A workspace has one connection row, keyed by workspace (E23). Personal
   connections are keyed by workspace and user (E24). The resolver loads the
   one row and caches by workspace (E23). The spec boundary says "one
   automation connection per workspace" (E20).
2. Token resolution rejects hosts other than `github.com` (E1), and every
   client talks to `api.github.com` (E4, E5, E25).
3. No resolution request carries a host (E27), and the PR, watch, and sync
   tables store owner and repo without a host (E28, E30).
4. Repository classification, remote URL parsing, cloning, and the review
   resolver accept only `github.com` (E9, E10, E31).
5. Pull-request identity, prompts, links, contributions, forks, and mentions
   force `github.com` (E15 to E18, E32, E34).
6. Task credential paths outside the executor host bridge are `github.com`
   only (E13, E19, E36).
7. The settings UI shows one automation connection; parsers and allowlists
   accept `github.com` only (E33, E35).

Pieces already in place: the connection row stores `github_host` (E3), the CLI
picker sends the account host (E2), App registrations are keyed by host and
App ID (E25), repositories store `provider_host`, broker scopes carry `Host`,
and the executor host bridge already bridges enterprise hosts (E12).

### Decisions

- **Status keeps its top-level fields.** `GET /api/v1/github/status` keeps the
  current fields as the `github.com` projection and adds a `hosts` array, so
  existing clients keep working.
- **Requests without a host mean `github.com`.** Routes and WebSocket actions
  that name an owner and repository take an optional `host`. Anyone using an
  enterprise host has to add that host's connection first, so a missing `host`
  cannot silently reach the wrong enterprise tenant, and existing API clients,
  WebSocket payloads, stored rows, and automation configs keep working.
- **A workspace usually holds one or two hosts** (`github.com` and one
  enterprise host). The settings list and status `hosts` array need no
  pagination or search.
- **Host helpers fail closed.** `WebHostFromURL` and `HostFromProviderHost`
  return an error for a malformed URL or an empty or malformed
  `provider_host`; every caller in tasks 03 and 05 surfaces that error instead
  of falling back to the `github.com` connection.
- **Connection key becomes (workspace, host).** It is the smallest model that
  lets one workspace hold several hosts: the host column, request field, and
  App host key already exist (E2, E3, E25). A second connection table would
  duplicate the concept. The spec boundary in E20 changes to "one connection
  per workspace and host".
- **Routing uses the item host, never a lookup by owner and repo.** The same
  owner and name can exist on two hosts, so host travels with every item: from
  `provider_host`, a `github_host` column, a URL, or a request parameter
  (system design, Item host).
- **Endpoint mapping follows GitHub and `gh`.** `github.com`, `*.ghe.com`, and
  other hosts map to the records in the system design (E6, E26).
- **New workspaces snapshot one CLI connection per active `gh` host account.**
  Today they snapshot one account and prefer `github.com` (E14).

Unrelated `github.com` literals stay unchanged:
Kandev release and update downloads (`internal/agent/runtime/agentctl/agentctl_resolver.go:30`,
`internal/system/updates/github.go:21`, `tools/installer`, runtime release
sources), improve-Kandev issue filing (`internal/improvekandev/`), gist
sharing, documentation links, User-Agent strings, and the migration-only
`legacy_shared` source. They name Kandev's own `github.com` resources or a
deployment credential that predates per-host connections, not a user's GitHub
host.

## Requirements

- Requirements: [`github-enterprise-host.md`](../../specs/integrations/requirements/github-enterprise-host.md)
- System design: [`github-enterprise-host.md`](../../specs/integrations/system-design/github-enterprise-host.md)

| Requirement | Justified by | Work orders |
| --- | --- | --- |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001` | E1, E2, E3, E14, E20, E23, E24, E35 | 01, 02, 07 |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002` | E27, E28, E29, E30 | 03 |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003` | E8, E9, E10, E11, E31, E38 | 04 |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004` | E15, E16, E17, E18, E22, E32, E34 | 05, 07 |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005` | E7, E12, E13, E19, E36, E37 | 06 |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006` | E25, E26 | 01, 02, 07 |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007` | E2, E33, E35 | 07 |

## Work orders

1. [Task 01: Host-aware GitHub clients](task-01-host-aware-workspace-client.md). Wave 1.
2. [Task 02: Per-host workspace connections](task-02-per-host-connections.md). Wave 2, after 01.
3. [Task 03: Host routing for GitHub operations](task-03-host-routed-operations.md). Wave 3, after 02.
4. [Task 04: Enterprise repository identity](task-04-enterprise-repository-identity.md). Wave 4, after 03.
5. [Task 05: Pull-request, issue, contribution, and mention hosts](task-05-enterprise-pr-issue-hosts.md). Wave 4, after 03.
6. [Task 06: Task Git and CLI credentials](task-06-enterprise-task-credentials.md). Wave 5, after 04.
7. [Task 07: Settings and URL recognition](task-07-settings-and-url-recognition.md). Wave 4, after 03.
8. [Task 08: Documentation](task-08-docs.md). Wave 6, after 01 to 07.

Tasks 04, 05, and 07 touch disjoint files and can run in parallel. Each work
order lists its files under `## Changes` with What, Justified by, Governed by,
and Specifics. Evidence IDs refer to the list below.

## ASCII UI preview

Structure is required; spacing, icons, and copy are illustrative. Copy goes
through `t()` in seven locales. Maps to `AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.1`
to `.5` and `001.4`.

UI-01: Workspace GitHub settings, connections section. Entry: Settings >
Workspace > Integrations > GitHub. State: two hosts connected.

Before (current, single connection):

```text
+-- Workspace automation ---------------------------------+
| GitHub CLI . alice (github.com)          [Change] [Disc] |
+---------------------------------------------------------+
```

After, desktop:

```text
+-- GitHub connections -----------------------------------+
| github.com      GitHub CLI . alice     Active  [Change] [Disconnect] |
| octo.ghe.com    GitHub App . octo-org  Active  [Change] [Disconnect] |
|                 Personal: bob          [Disconnect personal]          |
|                                                                       |
| [+ Connect another GitHub host]                                       |
+-----------------------------------------------------------------------+
```

After, phone (cards stack; actions move into a row menu):

```text
+-- GitHub connections ------+
| github.com            [..] |
| GitHub CLI . alice         |
| Active                     |
|----------------------------|
| octo.ghe.com          [..] |
| GitHub App . octo-org      |
| Active . Personal: bob     |
|----------------------------|
| [+ Connect another host]   |
+----------------------------+
```

UI-02: Connection dialog, PAT method. Entry: Connect another GitHub host, or
Change. The host field is new and defaults to `github.com`; the CLI method
lists accounts on every host; the App method shows the registration host.

```text
+-- Connect GitHub ---------------------------+
| Method:  ( ) GitHub CLI (o) Token ( ) App    |
| Host:    [ github.com                     ]  |
| Token:   [ ************************       ]  |
|                          [Cancel] [Connect]  |
+---------------------------------------------+
```

UI-03: GitHub dashboard and search with more than one host. A host selector
appears only when the workspace has two or more connections.

```text
+-- My GitHub -------------------------------------------+
| Host: [ octo.ghe.com v ]   Search: [ is:open ...     ] |
| #42  Fix login   octo-org/api        octo.ghe.com      |
+--------------------------------------------------------+
```

UI-04: URL input with a GitHub-shaped URL on a host with no connection.
Applies to task create, PR and issue link dialogs, watches, workspace
sources, and workflow sync.

```text
| URL: [ https://ghe.example.com/org/repo/pull/7 ]       |
| ghe.example.com is not connected. [Open GitHub settings] |
```

## Evidence

- **E1** (codebase) `apps/backend/internal/github/gh_accounts.go:191`: `if host != defaultGitHubHost { return "", fmt.Errorf("unsupported GitHub host %q", host) }`; pinned by `gh_accounts_test.go:294`.
- **E2** (codebase) `apps/web/components/github/github-cli-form.tsx:121-122` lists `{item.login} ({item.host})`; `github-connection-settings-form.tsx:96` sends `{ source: "gh_cli", host: cliAccount.host, login: cliAccount.login }`.
- **E3** (codebase) `apps/backend/internal/github/models.go:139` `GitHubHost` with `db:"github_host"`; `service_connections.go:131,198-200,240` default and store the host.
- **E4** (codebase) `apps/backend/internal/github/pat_client.go:19` `githubAPIBase = "https://api.github.com"`, used at `:934,970,999,1030,1054,1096`; `graphql.go:705`; `service_connections.go:301`; `service_reviews.go:664` anonymous base.
- **E5** (codebase) `apps/backend/internal/github/auth_resolver.go:392-399` passes `connection.GitHubHost` to `ghToken` but builds the token client without a host; `:365-389` PAT has no host at all.
- **E6** (web) GitHub docs, "Migrating repositories between two data-resident enterprises": data-residency API URLs use `https://api.SUBDOMAIN.ghe.com`, unlike GitHub Enterprise Server's `https://HOSTNAME/api/v3`. GitHub Enterprise Server docs, "Forming calls with GraphQL" (3.20 to 3.22): the GraphQL endpoint is `http(s)://HOSTNAME/api/graphql`. Corroborated by runatlantis/atlantis PR 6339, which cites cli/cli PR 9618 for `api.SUBDOMAIN.ghe.com` and `/graphql`.
- **E7** (web, single-source) GitHub CLI manual, `gh help environment`: `GH_TOKEN`/`GITHUB_TOKEN` apply to `github.com` and `ghe.com` subdomains; `GH_ENTERPRISE_TOKEN`/`GITHUB_ENTERPRISE_TOKEN` apply to GitHub Enterprise Server hosts; `GH_HOST` selects the host when it cannot be inferred. Reopen if a `gh` release changes this rule.
- **E8** (web) GitHub docs, "Feature overview for GitHub Enterprise Cloud with data residency", URL differences: SSH clone is `git clone SUBDOMAIN@SUBDOMAIN.ghe.com:OWNER/REPOSITORY.git`, so SCP remotes on a tenant use a non-`git` user; same page: REST and GraphQL requests must go to the tenant's dedicated URL.
- **E9** (codebase) `apps/backend/internal/task/service/repository_discovery.go:1340-1357` sets a provider only for `github.com` and `gitlab.com`; `service_resources.go:1189-1211` stores provider host, owner, and name only when a provider was found; called from `createRepository` at `:1140-1141`.
- **E10** (codebase) `repository_discovery.go:1248-1262` `ParseGitRemoteIdentity` strips any `user@`; `service_tasks.go:1826` `sshStyle := strings.HasPrefix(raw, "git@")`, so `canonicalCloneOrigin` fails for other users and `service_resources.go:1204-1209` falls back to an `ssh://` remote URL.
- **E11** (codebase) `apps/backend/internal/task/service/repository_origin.go:7-41` GitLab origin verifier setter and trust flag; `apps/backend/internal/backendapp/repository_origin.go:19-27` compares the workspace GitLab host; wired at `backendapp/services.go:599`.
- **E12** (codebase) `apps/backend/internal/orchestrator/executor/executor_host_gh_bridge.go:334-385` bridges attached hosts with provider `github`; `:306-322` and `:386-391` treat every non-`github.com` host as needing `GH_ENTERPRISE_TOKEN`, which contradicts E7 for `ghe.com` tenants.
- **E13** (codebase) `apps/backend/internal/github/credential_broker.go:250-252` broker `Binding` denies hosts other than `github.com`; `:303-316` `Resolve` builds a request without the scope host.
- **E14** (codebase) `apps/backend/internal/github/workspace_defaults.go:117-131` picks one active account and prefers `github.com`; `docs/plans/new-workspace-github-access-defaults/plan.md:30-31` new workspaces default to executor credentials.
- **E15** (codebase) `apps/backend/internal/github/service_task_issue.go:202-205` rejects issue URLs off `github.com`; `:255` re-parses stored links when listing.
- **E16** (codebase) `apps/backend/internal/backendapp/task_change_link_coordinator.go:243,437` builds the PR URL from `ProviderHost` but rejects hosts other than `github.com` and `*.github.com`.
- **E17** (codebase) `apps/backend/internal/orchestrator/event_handlers_github_pr_automation.go:220-225` prompt URL is `https://github.com/%s/%s/pull/%d`.
- **E18** (codebase) `apps/backend/internal/github/service_comparison_target.go:106-117` candidate host `github.com`; `apps/backend/internal/task/models/comparison_target.go:148` rejects other GitHub hosts; `apps/backend/internal/backendapp/orchestrator.go:422-434,542-550` PR base identity on `github.com`; `apps/backend/internal/orchestrator/executor/executor_resume.go:504-555` normalizes attached repositories to `github.com` and `:340-348` fails the launch on mismatch.
- **E19** (codebase) `apps/backend/internal/agentctl/server/process/git_pr_providers.go:64-92` GitHub only for `github.com` and `*.github.com`; `git.go:1533-1536` "unsupported git remote for PR creation"; GitLab precedent `KANDEV_GITLAB_HOST` at `executor_credentials.go:933-952` and `git_pr_providers.go:86-87`; `env_hermetic_test.go:14-18,50` scrub list and coverage guard.
- **E20** (codebase) `docs/specs/integrations/system-design/github-authentication-03.md:239` excludes "Multiple automation connections or per-repository credential routing inside one workspace"; `:243` excludes hosts other than `github.com`.
- **E21** (codebase) `docs/public/integrations.md:54,84,233` state that GitHub targets `github.com` only.
- **E22** (codebase) `apps/backend/internal/github/service_pr_watch.go:821-850` `parsePRURL` ignores the host; `apps/web/components/github/my-github/quick-task-launcher.tsx:84-87` prefers `html_url`.
- **E23** (codebase) `apps/backend/internal/github/store.go:266-291` `github_workspace_connections` with `workspace_id TEXT PRIMARY KEY`; `store_connections.go:68-76` load by workspace; `:117-131` `ON CONFLICT(workspace_id)`; `auth_resolver.go:169-175` resolves the single row; `:282-289` cache key without host; `store.go:1309-1324` legacy seed `'github.com'`.
- **E24** (codebase) `store.go:293-354` `github_user_connections` `PRIMARY KEY (workspace_id, user_id)` and triggers that match the workspace connection by workspace and registration; `:356-363` versions table keyed the same; `store.go:1129` PostgreSQL trigger variant.
- **E25** (codebase) `store.go:474-495` App registrations store `github_host` with `UNIQUE (github_host, app_id)`; `app_registration_import.go:168` requires `github.com`; `app_registration_lifecycle_service.go:358` stores `github.com`; `deployment_app_manifest.go:303-308` registration URL; `deployment_app_conversion_client.go:91` conversion base; `app_installation_service.go:124-128,238` install URL and connection host; `oauth_flow.go:256-261` OAuth bases; `personal_auth_service.go:136` authorize host; `app_client.go:117` App API base; `service_app_auth.go:156-214` runtime wiring; `apps/web/components/github/github-app-import-form.tsx:78` sends `github.com`.
- **E26** (web) GitHub docs source `content/apps/using-github-apps/installing-a-github-app-from-a-third-party.md`: the install URL is `{% ifversion ghes %}github-apps{% else %}apps{% endif %}/APP-NAME/installations/new` on the instance host; GitHub docs "Making your GitHub App available for GitHub Enterprise Server": OAuth uses `https://HOSTNAME/login/oauth/authorize` and each Server instance registers its own App. Corroborated by coollabsio/coolify issue 10573, which reports that `.ghe.com` tenants need the `/apps/` path, not `/github-apps/`.
- **E27** (codebase) `apps/backend/internal/github/service_routing.go:46-82` `resolveServiceClient` is the single service resolution entry; 60 calls through `resolveAutomationClient`, `resolvePersonalReadClient`, `resolvePersonalWriteClient` in 17 files; `auth_principal.go:41-47` `ResolveCredentialRequest` has no host; `service_git_transport.go` and `service_app_auth.go` build the other requests.
- **E28** (codebase) `store.go:109-233` `github_pr_watches`, `github_task_prs`, `github_review_watches`, `github_issue_watches` store owner and repo without a host; `models.go:1008-1011` `RepoFilter{Owner, Name}` used by watches, scope, and automation config.
- **E29** (codebase) `apps/backend/internal/github/controller.go:91-133` routes `/prs/:owner/:repo/:number/*`, `/issues/:owner/:repo/:number/info`, `/repos/:owner/:repo/*`, `/repos`, `/repos/search`, `/orgs`, `/user/prs`, `/user/issues`; `handlers.go:42-75` WebSocket actions for PR feedback, files, commits, and sync.
- **E30** (codebase) `apps/backend/internal/workflowsync/service.go:34-48,329-345` reads files by owner and repo; `workflowsync/store.go:34-40` config table; `apps/backend/internal/office/configsync/walk.go:41-65,415,454` and `store.go:36` the same for Office config sync; `apps/backend/internal/automation/evaluator.go:113-130` polls `ListAutomationPRs` by owner and repo; `automation/models.go:469-476` trigger `Repos []github.RepoFilter`; `apps/backend/internal/office/skills/import.go:83,112,301-330` imports skills from `github.com` only.
- **E31** (codebase) `apps/backend/internal/task/service/service_tasks.go:1797-1817` remote URL parser accepts `github.com` only (`:1804`); `:1841-1851` canonical URL on `github.com`; `:1877-1880` `remoteProviderHost` returns `https://github.com`; `repository_selection.go:85-100` and `workspace_source_placement_resolver.go:100-105` reuse it; `apps/backend/internal/repoclone/protocol.go:130-139` provider host `github.com`; `repoclone/clone.go:29,331,745`; `apps/backend/internal/backendapp/orchestrator.go:1356-1409` review resolver on `github.com`; `improvekandev/handler.go:406` (Kandev's own repository, unchanged).
- **E32** (codebase) `apps/backend/internal/github/service_remote_contribution.go:39,76-105`, `service_fork.go:389-392`, `apps/backend/internal/task/models/remote_contribution.go:225`, `contribution_destination.go:96`, `apps/backend/internal/backendapp/remote_contribution.go:33`, `backendapp/services.go:1028,1129,1200`, `backendapp/git_credentials.go:20`, `executor_credentials.go:692,708,727-728,750`, `apps/backend/internal/common/gitbase/materialize.go:205,221,229-276`, `apps/backend/internal/worktree/managed_clone_relocation.go:781` assume `github.com` for contributions, forks, and base materialization.
- **E33** (codebase) `apps/web/components/workspace-source-picker/remote-url.ts:17-26`; `lib/github/parse-url.ts:16-24`; `lib/utils/github-repo-url.ts:8-9,52-55,65-70`; `hooks/domains/github/use-pr-info-by-url.ts:63-100`; `components/task/task-github-pr-url.ts:40-55,64-70`; `components/task-create-dialog-remote-repo-identity.ts:93-94`; `hooks/domains/github/use-branches-by-url.ts:311-323`; `hooks/domains/settings/use-workflow-sync.ts:20,75-80`.
- **E34** (codebase) `apps/web/components/github/pr-shared.tsx:62,75` and `pr-detail-panel.tsx:248-253` profile links; `lib/utils/external-vcs-file-url.ts:213-217,254-264` file links; `quick-task-launcher.tsx:84-87,162`; `hooks/domains/integrations/use-remote-repositories.ts:167`; `components/github/github-app-onboarding-model.ts:144-147`; `apps/backend/internal/mentions/provider_github.go:145-156,248-262` mention URLs and scope.
- **E35** (codebase) `apps/backend/internal/github/service_app_auth.go:463-540` status projects one connection; `controller.go:144-164`, `controller_auth.go:35-99` single-connection handlers; `apps/web/lib/types/github.ts:31-50` one `automation`; `components/github/github-status.tsx:58-91,142-211` render it; `lib/api/domains/github-auth-api.ts:15-40`.
- **E36** (codebase) `apps/backend/internal/agent/runtime/lifecycle/container.go:38,895-914` `github.com` helper and rewrites; `executor_sprites_helpers.go:35-60`; `apps/backend/internal/scriptengine/providers.go:379-400`; `apps/backend/cmd/agentctl/github_cli_shim.go:84-89` sets only `GH_TOKEN`, `:214` defaults the host to `github.com`.
- **E37** (codebase) `apps/backend/internal/gitcredentials/identity.go:142-158` uses `ProviderHost` when present and defaults to `github.com` only when it is empty.
- **E38** (codebase) `apps/backend/internal/orchestrator/event_handlers_github.go:1027-1033,1157-1163,1392,1466` backfill provider fields from `ResolveGitRemoteProvider`, which stays `github.com` only and records no host.
- **E39** (codebase) `store.go:1745` `rebuildIfHasLegacyConstraint` precedent; `apps/backend/AGENTS.md:266,284,287` replayable migrations, table-rebuild and PostgreSQL rules, `sqlguard` and store conformance commands.
- **E40** (codebase) `apps/backend/internal/github/service.go:172,260` `tokenClientFactory func(string) Client`, overridden at `mock_auth.go:156` and `service_connections_test.go:83,202,422`.
- **E41** (codebase) `apps/backend/internal/gitlab/constants.go:30-32` per-workspace `gitlab_host`; `gitlab/pat_client.go:57` `NewPATClient(host, token string)`.

## Coding standards

- Go limits: functions at most 80 lines and 50 statements, cyclomatic 15,
  cognitive 30, nesting 5, constants for strings repeated 3 or more times, new
  tests in a new file when the target test file is large. Found in
  `apps/backend/AGENTS.md:292`. Governs every Go change.
- Use `-trimpath` in direct `go test`; lint changed Go files with
  `bash scripts/lint-go-changed <files>`. Found in `apps/backend/AGENTS.md:294-295`.
  Governs verification.
- Replayable schema changes, fresh-DB and same-DB replay tests, table rebuilds
  that mirror every column with a rollback test, PostgreSQL behavior tests for
  dialect-sensitive methods, `go run ./cmd/sqlguard ./internal`, and store
  conformance. Found in `apps/backend/AGENTS.md:266,270,284,287` and
  `docs/decisions/0027-replayable-schema-migrations.md`. Governs task 02 and
  the column additions in task 03.
- Workspace-scoped side tables need explicit deletion handling and E2E reset
  coverage. Found in `apps/backend/AGENTS.md:244,253`. Governs task 02.
- Clear ambient host variables in package `TestMain` when tests read them.
  Found in `.agents/skills/tdd/references/backend-tests.md`. Governs task 06.
- Every change ships tests; production comments state the invariant without
  requirement IDs. Found in `CLAUDE.md` (Testing, Code Quality). Governs every
  work order.
- All new UI copy goes through `t()` in seven locales, with no em dash; plain
  `.ts` copy passes `check-nonjsx-copy`. Found in `CLAUDE.md`
  (Internationalization). Governs task 07.
- UI changes follow `/mobile-parity` and the ASCII preview rules in
  `docs/specs/guide/plans-and-work-orders.md`. Governs task 07.
- TypeScript limits and shadcn imports. Found in `apps/web/AGENTS.md`. Governs
  task 07.
- Requirements, designs, plans, and work orders follow `docs/specs/guide/`;
  public docs follow `docs/public/README.md`. Found in `CLAUDE.md`
  (Knowledge). Governs task 08.
- Mirror the GitLab host shape: host-carrying client constructors and a
  setter-injected workspace reader in the task service. Inferred from
  surrounding code (E11, E41). Governs tasks 01, 02, and 04.

## Verification

From `apps/backend`:

```bash
go test -trimpath -tags fts5 -race ./internal/github ./internal/task/... ./internal/backendapp ./internal/orchestrator/... ./internal/agentctl/server/process ./internal/agent/runtime/lifecycle ./internal/scriptengine ./internal/gitcredentials ./internal/repoclone ./internal/common/gitbase ./internal/mentions ./internal/workflowsync ./internal/office/configsync ./internal/office/skills ./internal/automation ./cmd/agentctl -count=1
go run ./cmd/sqlguard ./internal
go test -race ./internal/persistence/storeconformance -count=1
KANDEV_TEST_POSTGRES_DSN=<dsn> go test -trimpath -tags fts5 -race ./internal/github -run 'TestStore.*Postgres|TestWorkspaceConnection.*Migration' -count=1
```

From `apps/web`:

```bash
pnpm run typecheck
pnpm test
pnpm run lint
pnpm run i18n:check
pnpm run i18n:ratchet
pnpm e2e:run -- e2e/tests/integrations/github-multi-host-settings.spec.ts
```

From the repository root:

```bash
bash scripts/lint-go-changed $(git diff --name-only origin/main -- '*.go')
python3 scripts/list-docs.py validate
python3 scripts/lint-spec-files.py --all
node scripts/validate-public-docs.mjs
```

Live end-to-end evidence before the upstream PR, recorded in task 08 results.
Use a GHE.com tenant and, if available, a GitHub Enterprise Server instance,
in one workspace that also has a `github.com` connection:

1. Connect `github.com` and the enterprise host through GitHub CLI; both cards
   show the login. Replace the enterprise connection with a PAT; the
   `github.com` card is unchanged.
2. Add a local checkout with an `org@<tenant>.ghe.com:` origin and add an
   enterprise repository by remote URL; both show as GitHub on that host.
3. Start a Worktree task and a managed-mode task on the enterprise repository;
   push, run `gh pr view`, and create the PR from Kandev; checks and reviews
   update.
4. Link an issue URL and a PR URL from the enterprise host; both survive a
   reload. Link a URL from an unconnected host; the UI says it is not
   connected.
5. Create and install a GitHub App on the enterprise host through a public
   HTTPS origin, connect a personal identity, and receive a webhook.

## Risks

- **Connection key migration.** Every resolution path changes. The rebuild
  must preserve rows, triggers, generations, and personal connections on SQLite
  and PostgreSQL; task 02 requires replay and rollback tests.
- **Call-site coverage.** 60 service resolutions plus callers in other
  packages must pass the right host; a missed site silently uses `github.com`.
  Task 03 adds a test that a non-`github.com` item never reaches the
  `github.com` connection for each exported entry point group.
- **GitHub Enterprise Server is likely tested by unit tests only.** Older
  Server releases can lack GraphQL fields used by pull-request discovery.
- **GHE.com App install URL.** E26 gives `/apps/<slug>/installations/new`; a
  third-party report uses an owner-scoped form. The live run in step 5 settles
  it.
- **Same owner and repository on two hosts.** Correct by construction only if
  every item carries its host; requests without a host default to
  `github.com`.

## Deferred (YAGNI)

### GitHub hosts on a non-default port
**Why deferred:** Gate 1. GitHub Enterprise Server serves its web UI and API
on the standard HTTPS port, and no request named a port. Supporting ports
widens host validation, clone URLs, and credential helper keys.
**Reopen when:** a user reports a GitHub Enterprise Server instance reachable
only on a custom port.
**Source:** host validation design.
