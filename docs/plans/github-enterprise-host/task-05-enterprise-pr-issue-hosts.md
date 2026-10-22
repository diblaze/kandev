---
id: "05-enterprise-pr-issue-hosts"
title: "Pull-request, issue, contribution, and mention hosts"
status: pending
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
**Specifics:** `githubComparisonTargetCandidate` (line 74) and `githubPRBaseFromPR` (orchestrator.go:422) use `WebHostFromURL(pr.HTMLURL)` for both repositories; `githubComparisonRepository(host, owner, repo, id)` (orchestrator.go:542).

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
**Specifics:** the origin classification (lines 772-786) keeps returning the host; set provider `github` when the host equals the repository's `ProviderHost` hostname as well as for `github.com`.
Unverified: could not inspect the caller of this classification, because only lines 772-786 were read; the implementer confirms the repository row is in scope there.

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

Pending.
