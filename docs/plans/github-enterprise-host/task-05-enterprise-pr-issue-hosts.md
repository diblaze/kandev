---
id: "05-enterprise-pr-issue-hosts"
title: "Pull-request, issue, contribution, and mention hosts"
status: done
wave: 4
depends_on:
  - "03-host-routed-operations"
plan: "plan.md"
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004
acceptance_criteria:
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.1
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.2
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.3
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.4
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.5
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.6
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.7
system_design:
  - ../../specs/integrations/system-design/github-enterprise-host.md
---

# Task 05: Pull-Request, Issue, Contribution, And Mention Hosts

## Summary

Replace the `github.com` literals on pull-request, issue, comparison,
contribution, fork, base materialization, and mention paths with the item
host. A pull-request or issue URL selects a host, and the connection for that
host must exist.

## In scope

- Issue and pull-request URL parsing and linking.
- Comparison targets and pull-request base identity.
- Pull-request automation prompt URLs and task-change links.
- Remote contribution, contribution destination, and fork resolution.
- Base materialization and managed clone relocation.
- Mention URLs and scopes.

## Out of scope

- Broker authorization of contribution scopes (task 06).
- Frontend links (task 07).

## Acceptance

- An issue or PR URL on a connected enterprise host links and lists; on an unconnected host it fails with `HostNotConnectedError`; on `gitlab.com` it still fails as invalid.
- A same-repository and a fork pull request on `octo.ghe.com` resolve their comparison target and base at launch.
- Remote contribution from `https://octo.ghe.com/org/repo/pull/7` produces source and target repositories on `octo.ghe.com`.
- Automation prompts and task-change links use the item host.
- Mentions for an `octo.ghe.com` repository have URL `https://octo.ghe.com/org/repo/pull/3` and scope `octo.ghe.com/org/repo`; `github.com` scopes are unchanged.

## Changes

### `apps/backend/internal/github/service_task_issue.go`
**What:** modified.
**Justified by:** E15
**Governed by:** Go limits.
**Specifics:** `parseIssueReference(input, defaultOwner, defaultRepo string) (host, owner, repo string, number int, err error)` (line 187) accepts any host that passes `normalizeGitHubHost` with the `/<owner>/<repo>/issues/<n>` path, still rejecting `gitlab.com`, GitLab `/-/issues/` paths, and non-HTTPS URLs. `LinkTaskIssueForUser` resolves the client for that host, so an unconnected host fails closed. `taskIssueLinkFromMetadata` (line 255) keeps any host.

### `apps/backend/internal/github/service_pr_watch.go`
**What:** modified.
**Justified by:** E22
**Governed by:** Go limits.
**Specifics:** `parsePRURL` (line 821) also returns `WebHostFromURL(prURL)`; `AssociateExistingPRByURLForWorkspace` (line 701) and `AssociatePRByURLForWorkspace` (line 778) resolve that host and persist it.

### `apps/backend/internal/github/service_comparison_target.go` and `apps/backend/internal/backendapp/orchestrator.go`
**What:** modified.
**Justified by:** E18
**Governed by:** Go limits.
**Specifics:** `githubComparisonTargetCandidate(host, pr)` (line 74) and `githubPRBaseFromPR(pr, host, ...)` (orchestrator.go:422) build both repositories on the host the pull request was read from: the association host or the task PR `github_host` for the candidate, the resolved lookup host for the PR base; `githubComparisonRepository(host, owner, repo, id)` (orchestrator.go:542).

### `apps/backend/internal/task/models/comparison_target.go`, `remote_contribution.go`, `contribution_destination.go`
**What:** modified.
**Justified by:** E18, E32
**Governed by:** Go limits.
**Specifics:** drop the `github.com`-only checks at comparison_target.go:148, remote_contribution.go:225, and contribution_destination.go:96; `validateRemoteContributionRepository` still enforces a well-formed host, no port, and host and path agreement with the remote URL. Head and target must share a host (comparison_target.go:132).

### `apps/backend/internal/orchestrator/executor/executor_resume.go`
**What:** modified.
**Justified by:** E18
**Governed by:** Go limits.
**Specifics:** `isGitHubComparisonHost` (line 540) keeps the shape rules and drops the `github.com` equality; `normalizeGitHubComparisonRepository` (line 525) keeps the hostname and builds `https://<host>/<owner>/<name>.git`.

### `apps/backend/internal/orchestrator/event_handlers_github_pr_automation.go`
**What:** modified.
**Justified by:** E17
**Governed by:** Go limits.
**Specifics:** `canonicalTaskPRURL` (line 220) formats on `pr.GitHubHost`.

### `apps/backend/internal/backendapp/task_change_link_coordinator.go`
**What:** modified.
**Justified by:** E16
**Governed by:** Go limits.
**Specifics:** `normalizedGitHubHost` (line 421) accepts any well-formed host; the link resolves through `AssociateExistingPRByURLForWorkspace`, which fails closed for an unconnected host.

### `apps/backend/internal/github/service_remote_contribution.go` and `apps/backend/internal/backendapp/remote_contribution.go`
**What:** modified.
**Justified by:** E32
**Governed by:** Go limits.
**Specifics:** `parseGitHubContributionURL` (line 34) returns the host and accepts any host; `buildGitHubRemoteContribution` (lines 76-105) builds every URL, `Host`, and `TargetHost` on it; the backendapp dispatcher (remote_contribution.go:33) routes a `/<owner>/<repo>/pull/<n>` URL on any host to GitHub, after GitLab's `/-/merge_requests/` check.

### `apps/backend/internal/github/service_fork.go`
**What:** modified.
**Justified by:** E32
**Governed by:** Go limits.
**Specifics:** fork and destination methods take the host from task 03 and build `remoteURL` and `Host` on it (lines 389-392).

### `apps/backend/internal/common/gitbase/materialize.go`
**What:** modified.
**Justified by:** E32
**Governed by:** Go limits.
**Specifics:** `githubSSHComparisonURL` and `sameGitHubComparisonRepository` (lines 204-230) compare against the target repository host; `parseGitHubSSHURL`, `parseGitHubHTTPSURL`, and `parseGitHubSCPURL` (lines 248-276) take the expected host and accept the SSH user `git` or, for a `.ghe.com` host, its subdomain.

### `apps/backend/internal/worktree/managed_clone_relocation.go`
**What:** modified.
**Justified by:** E32
**Governed by:** Go limits.
**Specifics:** the origin classification (lines 772-786) is unchanged: `verifyManagedCloneOrigin` already compares the origin host with the repository `ProviderHost` and skips the provider check when the origin host is not `github.com` or `gitlab.com`. The SCP and `ssh://` origin parsers accept the SSH user `repoclone.SSHUser(host)` instead of only `git`, so a `<sub>@<sub>.ghe.com` origin verifies.

### `apps/backend/internal/mentions/provider_github.go`
**What:** modified.
**Justified by:** E34
**Governed by:** Go limits.
**Specifics:** candidate URL and scope (lines 145-156) use the repository host; scope stays `github.com/<owner>/<repo>` for `github.com` and becomes `<host>/<owner>/<repo>` otherwise; `repositoryFromGitHubScope` (line 257) parses either; the URL check (lines 248-255) uses the scope host. Search calls pass the host to task 03's mention search methods.

### Tests
**What:** added and modified.
**Justified by:** E15 to E18, E22, E32, E34
**Governed by:** `CLAUDE.md` Testing.
**Specifics:**
- `github/service_task_issue_test.go` (line 269 table): enterprise host returns its host; `gitlab.com` still invalid. New `service_task_issue_hosts_test.go` for link and list with connected and unconnected hosts.
- `github/service_comparison_target_test.go`: enterprise `HTMLURL` case.
- `task/models/comparison_target_test.go`: rename `provider mismatch` (line 64) to `mixed provider hosts`; add an enterprise pair and a port rejection.
- `orchestrator/executor/executor_pr_base_identity_test.go`: replace `TestNormalizeGitHubComparisonRepositoryRejectsOtherHosts` (line 34) with port and path rejection plus an enterprise acceptance case; add a launch case for a same-repository enterprise PR.
- `orchestrator/event_handlers_github_pr_automation_test.go` (`TestCanonicalTaskPRURL`, line 677): enterprise host case.
- `backendapp/task_change_link_coordinator_test.go`: `TestTaskChangeURLsRequireCanonicalRepositoryIdentity` (line 192) now accepts `https://gitlab.example.test` shape-wise; move the non-GitHub rejection to the coordinator test where the provider routes; keep spoofing tests at line 209 for malformed hosts.
- `github/service_remote_contribution_test.go`, `common/gitbase` tests, `mentions` tests: enterprise cases.

## Verification

```bash
# From apps/backend:
go test -trimpath -tags fts5 -race ./internal/github -run 'Test(ParseIssueReference|TaskIssueHosts|GitHubComparisonTarget|RemoteContribution|ParsePRURL)' -count=1
go test -trimpath -tags fts5 -race ./internal/task/models ./internal/common/gitbase ./internal/mentions ./internal/worktree -count=1
go test -trimpath -tags fts5 -race ./internal/orchestrator/... ./internal/backendapp -run 'Test(NormalizeGitHubComparisonRepository|ResolvePRBase|CanonicalTaskPRURL|TaskChange|GitHubChangeURL|RemoteContribution)' -count=1
```

Then `bash scripts/lint-go-changed <changed Go files>` from the repository root.

## Dependencies

Task 03.

## Risks

- Relaxing host checks in models widens accepted hosts; credential selection
  stays bound to workspace connections, so an unconnected host still cannot
  be reached.

## Parallelism

`parallel` with tasks 04 and 07.

## Results

Implemented test-first.

What was done:

- Issues: `parseIssueReference` returns the URL host and accepts any GitHub host (`WebHostFromURL`: https only, no user info, no port), still rejecting `gitlab.com` and GitLab `/-/issues/` paths. `LinkTaskIssueRequest` gained an optional `host` for owner/repo/number references (default `github.com`). `LinkTaskIssueForUser` resolves the scope and client on the issue host, so an unconnected host fails with `HostNotConnectedError`; the task-repository check also matches the repository host. Listing keeps links on any host.
- Pull requests by URL: `parsePRURL` returns the host; `AssociateExistingPRByURLForWorkspace` and `AssociatePRByURLForWorkspace` resolve, watch, and persist on it.
- Comparison targets and PR base: the candidate and the PR base are built on the item host; models accept any bare GitHub host and reject a port (`validateGitHubRepositoryHost`) for comparison targets, remote contributions, and contribution destinations; head and target still share a host. The executor normalizes attached and contribution repositories on their host.
- Prompts and links: `canonicalTaskPRURL` formats on `pr.GitHubHost`; `githubChangeURL` accepts any well-formed host and the coordinator rejects a GitHub link on a repository whose provider is another one.
- Remote contribution, forks: `parseGitHubContributionURL` returns the host; every URL, `Host`, and `TargetHost` is built on it; the backendapp dispatcher checks GitLab `/-/merge_requests/` first and routes `/<owner>/<repo>/pull/<n>` on any host to GitHub. Fork resolution and destinations carry the routed host (`repoRef`) down to `repositoryBinding`.
- Base materialization: SSH transport reuse and same-repository checks compare against the target repository host with its SSH user (`repoclone.SSHUser`).
- Mentions: candidates use URL `https://<host>/...` and scope `<host>/<owner>/<repo>` (`github.com/<owner>/<repo>` unchanged); search covers `github.com` plus every connected host and fails only when all hosts fail; authorization parses the scope host and matches repository-list scope entries by host.
- Automation (deferred by task 04): `github_pr` trigger repository resolution and the PR association derive the host from the payload `html_url` (empty means `github.com`, malformed skips with a warning) and associate through `AssociatePRWithTaskForWorkspace`.

Verification (from `apps/backend`, `GOMAXPROCS=4 GOMEMLIMIT=6GiB`, `-p 2`, one run at a time):

- `go test -trimpath -tags fts5 ./internal/github ./internal/task/models ./internal/common/gitbase ./internal/mentions -count=1`: `ok internal/github 31.142s`, `ok internal/task/models 0.335s`, `ok internal/common/gitbase 0.014s`, `ok internal/mentions 0.236s`.
- `go test -trimpath -tags fts5 ./internal/backendapp -count=1`: `ok 131.701s`.
- `go test -trimpath -tags fts5 ./internal/orchestrator/... -count=1`: 37 failures in `internal/orchestrator` and `internal/orchestrator/executor`, all in the known environmental baseline list; every other subpackage `ok`.
- `./internal/worktree`: the same 24 environmental failures as an untouched run before this order (`external content filters`, checkout transfer); `TestParseManagedGitOrigin` `ok`.
- `./internal/task/service`: only the known baseline failure `TestWorkspaceRecoveryProjectsRelocationErrorFromEveryEntryPoint`; `./internal/repoclone` `ok`; `./internal/agentctl/server/process` and `./internal/agentctl/server/api` comparison and materialization tests `ok`.
- With `-race`, one package at a time: `internal/github` `ok 71.363s`, `internal/task/models`, `internal/common/gitbase`, `internal/mentions`, and the touched `internal/worktree` tests `ok`. `internal/backendapp` and `internal/orchestrator` not run with `-race` (memory limit).
- `go build -tags fts5 ./...` and `go vet` of every touched package: clean; `gofmt -l`: none.
- From the repository root, `bash scripts/lint-go-changed <files>` per package group (github, backendapp, task/models, orchestrator/executor, orchestrator, common/gitbase, worktree, mentions): `0 issues.` each.

Tests that passed on their first run, then proven by breaking the code: `TestTaskIssueHostsListKeepsEnterpriseIssueLinks` (forced a `github.com`-only host check), the port rejections in `internal/task/models` (dropped the `:` check), `TestResolveTaskRepoInfo_SameRepositoryEnterprisePRResolvesBase` (executor host check back to `github.com`), `TestRemoteContributionHostsResolveOnlyThroughTheURLHost` (resolved on `github.com`), `TestContributionForkHostsBuildDestinationOnTheItemHost` (binding host `github.com`), `TestMaterializeEnterpriseBaseIgnoresOriginOnAnotherHost` (dropped the SCP host comparison), `TestGitHubPRTriggerRepositoryResolvesOnTheHTMLURLHost` (passed an empty host), `TestGitHubProviderHostsRejectReferenceURLOffTheScopeHost` (already rejected before; guard). `TestGitHubPRBaseResolverUsesLinkedPRForLegacyAttachment` (existing) went red during the order on a linked task PR with an empty `github_host`; the PR base resolver now normalizes it with `RepositoryGitHubHost`.

Deviations:

- The comparison candidate and PR base take the host the pull request was read from, not `WebHostFromURL(pr.HTMLURL)`: that host is the routing source the rest of the package uses, and many PR fixtures carry no `HTMLURL`.
- `managed_clone_relocation.go`: setting provider `github` for a matching host would reject a self-hosted GitLab row with the same host; the existing host comparison already covers enterprise rows. The real gap was the SSH user, now `repoclone.SSHUser(host)`.
- `LinkTaskIssueRequest.Host` was added (optional host per the plan decision) and the issue repository check also compares hosts; neither was listed.
- `gitbase` now imports `repoclone` for `SSHUser`, so the SSH user rule has one owner.
- The SSH user accepted for a `.ghe.com` host is exactly its subdomain, not also `git`.
- Mention search across hosts (system design updated) and the `WorkspaceGitHubHosts` method on `GitHubMentionService` were not listed.
- `associateAutomationPR` now stores the automation workspace on the task PR row for `github.com` too.
- The spoofing test in `task_change_link_coordinator_test.go` became `TestGitHubChangeURLRejectsMalformedHosts`; `github.com.evil.test` is a well-formed host and now fails closed at credential resolution instead.

Deferred: broker authorization of enterprise contribution scopes (`authorizeContributionDestination`, `authorizeRemoteContribution` in `backendapp/services.go`) stays with task 06; frontend links stay with task 07.

Unverified: `internal/backendapp` and `internal/orchestrator` under `-race`.
