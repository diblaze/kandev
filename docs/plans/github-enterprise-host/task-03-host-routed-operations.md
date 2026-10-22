---
id: "03-host-routed-operations"
title: "Host routing for GitHub operations"
status: pending
wave: 3
depends_on:
  - "02-per-host-connections"
plan: "plan.md"
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002
acceptance_criteria:
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.1
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.2
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.3
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.4
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.5
system_design:
  - ../../specs/integrations/system-design/github-enterprise-host.md
---

# Task 03: Host Routing For GitHub Operations

## Summary

Carry the item host through every GitHub service operation and every caller.
Persist the host on task pull requests, pull-request watches, review and issue
watches, workflow sync, and Office config sync. Accept `host` on HTTP routes
and WebSocket actions. Requests without a host keep resolving `github.com`.

## In scope

- One internal resolution helper that takes a host.
- A `host` argument on exported `...ForWorkspace`, `...ForAutomation`, and
  `...ForWorkspaceUser` service methods that name an owner and repository, and
  on search and listing methods.
- Additive `github_host` columns and model fields.
- HTTP, WebSocket, MCP, automation, workflow sync, Office config sync, and
  Office skill import callers.

## Out of scope

- Repository classification (task 04) and URL-derived hosts for links and
  contributions (task 05).
- Deployment-client methods without `ForWorkspace` (for example
  `GetPR(ctx, owner, repo, number)` at `service_pr.go:253`), which serve the
  `github.com` `legacy_shared` source.

## Acceptance

- For each exported entry-point group (PR reads, PR writes, reviews, checks,
  files and commits, issues, branches and merge methods, contents, searches,
  accessible repositories, watches, cleanup, mentions), a test with a
  `github.com` and an `octo.ghe.com` connection proves an `octo.ghe.com` item
  uses only the `octo.ghe.com` client.
- The same groups with a `ghe.example.com` item and no connection return
  `HostNotConnectedError` and make no client call.
- Existing rows and calls without a host behave as before.

## Changes

### `apps/backend/internal/github/service_routing.go`
**What:** modified.
**Justified by:** E27
**Governed by:** Go limits.
**Specifics:** `type repoRef struct{ host, owner, name string }`; `resolveServiceClient(ctx, workspaceID, userID string, purpose CredentialPurpose, ref repoRef)` sets `ResolveCredentialRequest.Host`. `resolveAutomationClient`, `resolvePersonalReadClient`, `resolvePersonalWriteClient`, and `ensurePersonalRepositoryBoundary` take `repoRef`. `credentialCacheScope` includes `Principal.GitHubHost`.

### `apps/backend/internal/github/service_pr.go`, `service_automation_auth.go`, `service_reviews.go`, `service_issues.go`, `service_ci_automation.go`, `service_merge_access.go`, `branch_protection.go`, `service_repo_contents.go`, `service_accessible_repos.go`, `service_fork.go`, `service_mentions.go`, `service_task_issue.go`, `workspace_settings_service.go`
**What:** modified.
**Justified by:** E27, E29
**Governed by:** Go limits.
**Specifics:**
- Each exported method listed by `rg -n '^func \(s \*Service\) [A-Z]\w*(ForWorkspace|ForAutomation|ForWorkspaceUser)\(' apps/backend/internal/github` that names an owner and repository takes `host string` immediately before `owner` and passes `repoRef{host, owner, repo}`.
- Search and listing methods (`SearchUserPRsPagedForWorkspace`, `SearchUserIssuesPagedForWorkspace` and their `...User` variants, `ListAccessibleReposForWorkspace`, `SearchMentionIssuesForWorkspace`, `SearchMentionPullRequestsForWorkspace`, `ListAutomationPRs`) take `host string`.
- Service caches keyed by owner and repository use the credential cache scope, which now includes the host, so no cache key changes beyond that.
- `listRepoBranchesAnonymous` (`service_reviews.go:664`) runs only for `github.com`; enterprise hosts use the authenticated client.

### `apps/backend/internal/github/service_pr_watch.go`, `service_pr_watch_batched.go`, `service_pr_unwatched.go`, `service_pr_feedback_sync.go`, `service_cleanup.go`, `poller.go`, `poller_review_cleanup.go`, `service_task_events.go`
**What:** modified.
**Justified by:** E27, E28
**Governed by:** `apps/backend/internal/github/AGENTS.md` watch identity and batched lookup rules.
**Specifics:**
- `CreatePRWatchForWorkspace`, `EnsurePRWatchForWorkspace`, and `AssociatePRWithTaskForWorkspace` take `host` and persist it on the watch and the task pull request.
- Every resolution from a watch or task pull request uses its `GitHubHost`.
- Batched GraphQL lookups group watches by host as well as credential scope; one batch never mixes hosts.
- Review and issue watch polling (`poller.go`, `service_reviews.go`) uses the watch's `GitHubHost`; a `HostNotConnectedError` is stored as the watch's last error.
- Cleanup sweeps resolve per task pull request host.

### `apps/backend/internal/github/models.go` and `store.go`
**What:** modified.
**Justified by:** E28
**Governed by:** replayable migrations (`apps/backend/AGENTS.md:266`).
**Specifics:**
- `PRWatch`, `TaskPR`, `ReviewWatch`, and `IssueWatch` gain `GitHubHost string \`json:"github_host" db:"github_host"\``; `RepoFilter` gains `Host string \`json:"host,omitempty"\``.
- `applyIdempotentSchemaColumns` (line 592) adds `github_host TEXT NOT NULL DEFAULT 'github.com'` to `github_pr_watches`, `github_task_prs`, `github_review_watches`, and `github_issue_watches`; update the final `CREATE TABLE` statements (lines 109-233) and the multi-repo rebuild copy lists (`migratePRTablesForMultiRepo`, line 1615; `rebuildPRWatchesTableForTaskOwnership`, line 2079) to carry the column.
- Every select, insert, and update of those tables includes the column; models default an empty value to `github.com` on write.
- Workspace repository scope (`RepoScopeRepos`, models.go:1453) matches `RepoFilter.Host`, empty meaning `github.com`; scope orgs stay host-agnostic.

### `apps/backend/internal/github/controller.go`, `controller_merge_access.go`, `handlers.go`
**What:** modified.
**Justified by:** E29
**Governed by:** Go limits.
**Specifics:** every route at `controller.go:91-133` and every WebSocket action that reads `owner` and `repo` (`handlers.go:42-75`) reads an optional `host` (query string for GET, JSON field for bodies) and passes it, defaulting to `github.com`; review and issue watch create and update accept `github_host`; `GET /repos`, `/repos/search`, `/orgs`, `/user/prs`, `/user/issues` read `host`.

### `apps/backend/internal/backendapp/orchestrator.go`, `status_summary_adapter.go`, `adapters_office.go`, `helpers.go`
**What:** modified.
**Justified by:** E27, E18
**Governed by:** composition stays in `backendapp`, inferred from surrounding code.
**Specifics:** every call to a changed GitHub method passes `githubpkg.HostFromProviderHost(repo.ProviderHost)` for repository-driven calls, or the task pull request's `GitHubHost`. `PRBaseLookup` (executor.go:1479) gains `Host`, set from the repository in `resolvePRBaseForLaunch`.

### `apps/backend/internal/orchestrator/event_handlers_github.go`, `event_handlers_git.go`, `event_handlers_github_pr_automation.go`, `event_handlers_automation.go`
**What:** modified.
**Justified by:** E27
**Governed by:** Go limits.
**Specifics:** calls into the GitHub service pass the repository's provider host or the task pull request's host.

### `apps/backend/internal/mcp/handlers/handlers.go` and `task_pr_enrich.go`
**What:** modified.
**Justified by:** E27
**Governed by:** Go limits.
**Specifics:** MCP tools that take a repository or pull-request URL resolve the host from the URL (`WebHostFromURL`) or from the task repository; tools that take owner and repository accept an optional `host`. `handlers.go:2543` error matching also recognizes the host-not-connected error.

### `apps/backend/internal/automation/models.go` and `evaluator.go`
**What:** modified.
**Justified by:** E30
**Governed by:** Go limits.
**Specifics:** trigger `Repos []github.RepoFilter` already carry `Host` from models.go; `checkPRTrigger` (evaluator.go:113-130) passes `repo.Host` to `ListAutomationPRs` and includes the host in the dedup key only when it is not `github.com`, so existing keys stay stable. Webhook-driven triggers match the delivery's registration host.

### `apps/backend/internal/workflowsync/store.go`, `service.go`, and its handlers
**What:** modified.
**Justified by:** E30
**Governed by:** replayable migrations.
**Specifics:** `workflow_sync_configs` gains `github_host TEXT NOT NULL DEFAULT 'github.com'` (store.go:34); `Config` and `SetConfigRequest` gain `GitHubHost`; `GitHubClientProvider` methods (service.go:34-48) take `host`; `listGitHubEntries` (service.go:329-345) passes it.

### `apps/backend/internal/office/configsync/store.go`, `walk.go`, and its handlers
**What:** modified.
**Justified by:** E30
**Governed by:** replayable migrations.
**Specifics:** `office_config_sync_configs` (store.go:36) gains `github_host`; `GitHubClientProvider` (walk.go:41-65) takes `host`; calls at walk.go:415,454 pass `cfg.GitHubHost`.

### `apps/backend/internal/office/skills/import.go`
**What:** modified.
**Justified by:** E30
**Governed by:** Go limits.
**Specifics:** `ParseSource` (line 83) recognizes `https://<host>/` URLs for hosts connected in the importing workspace; `github.com` keeps the current anonymous raw and contents fetch (lines 301-330); an enterprise host fetches through `ListRepoDirectoryForWorkspace` and `GetRepoFileContentForWorkspace` with that host.
Unverified: could not inspect how the skill import service receives the workspace ID, because only `import.go` was read; the implementer threads it from the import request.

### Tests
**What:** added and modified.
**Justified by:** E27, E28, E30
**Governed by:** `CLAUDE.md` Testing; `apps/backend/internal/github/AGENTS.md` call-count assertions for batched lookups.
**Specifics:**
- `service_host_routing_test.go` in `internal/github`: a resolver fake with `github.com` and `octo.ghe.com` connections returning distinct mock clients; table over one method per entry-point group asserting the client used, plus the not-connected case asserting zero calls.
- `store_host_columns_test.go`: fresh and replayed schemas have the columns with default `github.com`; rows round-trip the host; SQLite and PostgreSQL.
- `service_pr_watch_batched_hosts_test.go`: two hosts produce two batches with the existing call-count assertions.
- Controller tests: `host` query and body fields reach the service; absence passes `github.com`.
- `automation`, `workflowsync`, and `office/configsync` tests: an enterprise host config uses the host; an existing config without a host uses `github.com`.
- Update existing fakes and callers for new signatures across `internal/github`, `backendapp`, `orchestrator`, `mcp`, `automation`, `workflowsync`, and `office`.

## Verification

```bash
# From apps/backend:
go test -trimpath -tags fts5 -race ./internal/github -run 'Test(ServiceHostRouting|StoreHostColumns|PRWatchBatchedHosts)' -count=1
go test -trimpath -tags fts5 -race ./internal/github ./internal/backendapp ./internal/orchestrator/... ./internal/mcp/... ./internal/automation ./internal/workflowsync ./internal/office/configsync ./internal/office/skills -count=1
KANDEV_TEST_POSTGRES_DSN=<dsn> go test -trimpath -tags fts5 -race ./internal/github ./internal/workflowsync ./internal/office/configsync -run 'Host' -count=1
go run ./cmd/sqlguard ./internal
```

Then `bash scripts/lint-go-changed <changed Go files>` from the repository root.

## Dependencies

Task 02.

## Risks

- This is the widest diff. Split commits by package and keep the routing test
  table complete; a missing group lets a call fall back to `github.com`.

## Parallelism

`sequential`

## Results

Pending.
