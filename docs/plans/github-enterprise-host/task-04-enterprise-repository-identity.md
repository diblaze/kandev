---
id: "04-enterprise-repository-identity"
title: "Enterprise repository identity"
status: pending
wave: 4
depends_on:
  - "03-host-routed-operations"
plan: "plan.md"
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003
acceptance_criteria:
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.1
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.2
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.3
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.4
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.5
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.6
system_design:
  - ../../specs/integrations/system-design/github-enterprise-host.md
---

# Task 04: Enterprise Repository Identity

## Summary

Classify local origins and remote URLs on any connected host as GitHub
repositories, reclassify existing rows when a host connects, clone with the
right URL and SSH user, and let the review-watch resolver clone from the pull
request's host. Connected hosts come from a setter-injected reader, as the
GitLab origin verifier does.

## In scope

- Local repository creation and remote URL registration and selection.
- Reclassification on connection save.
- Clone URL construction, including the GHE.com SSH user.
- Review-watch repository resolution.

## Out of scope

- The orchestrator provider backfill for rows without an owner
  (`event_handlers_github.go:1027-1033` and peers, E38). Enterprise rows get
  their owner from classification or reclassification, so the backfill only
  ever sees legacy `github.com` rows; it stays as it is.

## Acceptance

- A local checkout with origin `org@octo.ghe.com:org/repo.git` in a workspace connected to `octo.ghe.com` is stored with provider `github`, provider host `https://octo.ghe.com`, owner `org`, and name `repo`.
- Registering `https://ghe.example.com/org/repo` in a workspace connected to `ghe.example.com` creates a GitHub repository on that host; in a workspace without that connection it fails with the existing unsupported-host error.
- Saving an `octo.ghe.com` connection reclassifies an existing row with an empty provider and an `octo.ghe.com` remote; rows on other hosts are untouched.
- A clone URL for `octo.ghe.com` over SSH is `octo@octo.ghe.com:org/repo.git`; for `ghe.example.com` it is `git@ghe.example.com:org/repo.git`.
- `github.com` rows are recorded exactly as before.

## Changes

### `apps/backend/internal/task/service/repository_origin.go`
**What:** modified.
**Justified by:** E11
**Governed by:** setter-injected workspace reader, inferred from surrounding code (E11).
**Specifics:**
- `type WorkspaceGitHubHostsReader interface { WorkspaceGitHubHosts(context.Context, string) ([]string, error) }` and `SetWorkspaceGitHubHostsReader`.
- `func (s *Service) connectedGitHubEnterpriseHost(ctx, workspaceID, host string) bool`: true when `host` is not `github.com` and is in the reader's list; a reader error logs at debug and yields false.
- `verifyRemoteRepositoryOrigin` (line 16) sets `input.TrustedRemote = true` and `input.Provider = "github"` when the parsed URL is on a connected enterprise host.

### `apps/backend/internal/task/service/service.go`
**What:** modified.
**Justified by:** E11
**Governed by:** Go limits.
**Specifics:** add field `workspaceGitHubHostsReader` next to `gitLabRepositoryOriginVerifier` (line 587).

### `apps/backend/internal/task/service/service_tasks.go`
**What:** modified.
**Justified by:** E10, E31
**Governed by:** Go limits.
**Specifics:**
- `normalizeRemoteRepositoryURL` (around line 1826) treats any `user@host:path` without `://` as SCP style, not only `git@`.
- `parseRemoteRepositoryURL(raw, providerHint string)`: when `providerHint == "github"` and the host is not a known provider host, parse with `parseGitHubRemote(host, parts, sshStyle)` instead of failing.
- `parseGitHubRemote` (line 1841) builds the canonical HTTPS URL on its `host` argument instead of `github.com`; `www.github.com` still maps to `github.com`.
- `remoteProviderHost` (line 1877) returns `https://<host>` from the canonical URL for GitHub.
- `resolveRepoInputRemote` (line 1662) and `validateBuiltInRemoteHints` reject an enterprise GitHub origin unless `TrustedRemote` is set, mirroring the GitLab rule at line 1686.

### `apps/backend/internal/task/service/repository_selection.go` and `workspace_source_placement_resolver.go`
**What:** modified.
**Justified by:** E31
**Governed by:** Go limits.
**Specifics:** `repositorySelectionNeedsInspection` (repository_selection.go:85) and the placement check (workspace_source_placement_resolver.go:100-105) accept a GitHub origin on a connected enterprise host through the same trust flag.

### `apps/backend/internal/task/service/service_resources.go`
**What:** modified.
**Justified by:** E9, E10
**Governed by:** Go limits.
**Specifics:**
- Line 1141 passes `s.connectedGitHubHostsForWorkspace(ctx, req.WorkspaceID)`; `resolveRepositoryProviderIdentity(repository, enterpriseHosts []string)` (line 1189) tags provider `github` and provider host `https://<host>` when the detected provider is empty and the origin hostname is in the list.
- Add `func (s *Service) ReclassifyWorkspaceGitHubRepositories(ctx context.Context, workspaceID, host string) (int, error)`: list workspace repositories with an empty provider, parse each `RemoteURL` or local origin, and update provider, provider host, owner, and name when the host matches; publish `RepositoryUpdated` per row.

### `apps/backend/internal/backendapp/services.go`
**What:** modified.
**Justified by:** E11
**Governed by:** composition stays in `backendapp`.
**Specifics:** inside `if githubSvc != nil` (lines 585-593): `taskSvc.SetWorkspaceGitHubHostsReader(githubSvc)` and `githubSvc.SetHostConnectedObserver(func(ctx context.Context, workspaceID, host string) { _, _ = taskSvc.ReclassifyWorkspaceGitHubRepositories(ctx, workspaceID, host) })`, logging the error.

### `apps/backend/internal/github/service_connections.go`
**What:** modified (task 02 added the call point).
**Justified by:** E23
**Governed by:** Go limits.
**Specifics:** add `SetHostConnectedObserver(func(context.Context, string, string))`.

### `apps/backend/internal/repoclone/protocol.go` and `clone.go`
**What:** modified.
**Justified by:** E8, E31
**Governed by:** Go limits.
**Specifics:**
- `CloneURLWithHost` SSH branch (protocol.go:101-108) uses user `<subdomain>` for a `<subdomain>.ghe.com` host and `git` otherwise.
- `nonDefaultProviderHostSegment` (clone.go:325) is unchanged; enterprise hosts already get their own clone path segment.
- `githubHTTPSCloneURL` (clone.go:742) already validates against `providerHost`; add a test for an enterprise host.

### `apps/backend/internal/backendapp/orchestrator.go`
**What:** modified.
**Justified by:** E31
**Governed by:** Go limits.
**Specifics:** `ResolveForReview(ctx, workspaceID, provider, host, owner, name, defaultBranch)` (line 1356) uses `host` when provider is `github`, falling back to `defaultProviderHostname`; update the `RepositoryResolver` interface in `internal/orchestrator` and its callers, which pass the review watch's `GitHubHost`.

### Tests
**What:** added.
**Justified by:** E8, E9, E10, E31
**Governed by:** `CLAUDE.md` Testing; new test files instead of growing `service_resources_test.go` (1765 lines).
**Specifics:**
- `task/service/repository_enterprise_identity_test.go`: follow `TestService_CreateRepositoryResolvesRemoteURLForSelfHostedGitLabOrigin` (`service_resources_test.go:287`); cases for a connected tenant with an SCP origin, a connected Server host with HTTPS, an unconnected host, `github.com` unchanged, and a reader error.
- `task/service/remote_repository_enterprise_test.go`: register by URL on connected and unconnected hosts; SCP URL with a non-`git` user.
- `task/service/repository_reclassify_test.go`: reclassification updates only matching empty-provider rows and publishes events.
- `repoclone/protocol_enterprise_test.go`: SSH and HTTPS clone URLs for the three host kinds.
- `backendapp`: `ResolveForReview` clones on the given host.

## Verification

```bash
# From apps/backend:
go test -trimpath -tags fts5 -race ./internal/task/service -run 'Test(EnterpriseRepositoryIdentity|RemoteRepositoryEnterprise|ReclassifyWorkspaceGitHubRepositories|Service_CreateRepository)' -count=1
go test -trimpath -tags fts5 -race ./internal/repoclone -run 'TestCloneURLWithHost' -count=1
go test -trimpath -tags fts5 -race ./internal/task/... ./internal/repoclone ./internal/backendapp ./internal/orchestrator/... -count=1
```

Then `bash scripts/lint-go-changed <changed Go files>` from the repository root.

## Dependencies

Task 03 (`WorkspaceGitHubHosts`, review watch host).

## Risks

- Accepting any SCP user widens the remote parser. The host must still be a
  connected host or a known provider host before anything is stored.

## Parallelism

`parallel` with tasks 05 and 07.

## Results

Pending.
