---
id: "06-enterprise-task-credentials"
title: "Task Git and CLI credentials for enterprise hosts"
status: pending
wave: 5
depends_on:
  - "04-enterprise-repository-identity"
  - "05-enterprise-pr-issue-hosts"
plan: "plan.md"
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005
acceptance_criteria:
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.1
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.2
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.3
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.4
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.5
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.6
system_design:
  - ../../specs/integrations/system-design/github-enterprise-host.md
---

# Task 06: Task Git And CLI Credentials For Enterprise Hosts

## Summary

Make every task credential path work for attached repositories on connected
enterprise hosts: managed broker leases and authorization, the managed `gh`
shim, the executor host bridge token rule, remote executor prepare scripts,
and pull-request creation in agentctl.

## In scope

- Broker scope authorization in `backendapp`.
- Contribution scopes in the executor.
- The managed `gh` shim.
- The executor host bridge explicit-token rule.
- Docker, Sprites, and SSH prepare helpers.
- agentctl pull-request provider detection.

## Out of scope

- The broker provider binding itself, done in task 02.

## Acceptance

- A managed task with an `octo.ghe.com` repository gets a lease whose scope host is `octo.ghe.com`, Git fetch and push succeed through the helper, and a scope for `ghe.example.com` without a connection is denied.
- `gh pr view` through the shim for an `octo.ghe.com` repository runs with `GH_HOST=octo.ghe.com` and `GH_TOKEN`; for `ghe.example.com` with `GH_HOST=ghe.example.com` and `GH_ENTERPRISE_TOKEN`; for `github.com` exactly as today.
- In executor mode, an explicit `GH_TOKEN` suppresses the bridge for `github.com` and `octo.ghe.com`; `GH_ENTERPRISE_TOKEN` suppresses it for `ghe.example.com`.
- A Docker task with a profile token and an attached `octo.ghe.com` repository has a helper and SSH rewrites for `octo.ghe.com`.
- agentctl creates a PR for an origin on a host listed in `KANDEV_GITHUB_HOSTS`.

## Changes

### `apps/backend/internal/backendapp/git_credentials.go` and `services.go`
**What:** modified.
**Justified by:** E13, E32
**Governed by:** Go limits.
**Specifics:**
- `AuthorizeGitCredential` (git_credentials.go:232-250) treats any scope with provider `github` as GitHub; remove `gitCredentialGitHubHost` as a gate.
- `authorizeRepositoryIdentity` path at services.go:1080-1083 replaces `isPublicGitHubProviderHost(repository.ProviderHost)` with an equality between the repository provider host and the scope host. `isPublicGitHubProviderHost` stays only for `isCanonicalKandevRepositoryInput` (services.go:1002-1014), which concerns Kandev's own repository.
- `authorizeContributionDestination` (services.go:1200) and `authorizeRemoteContribution` (remote_contribution.go in backendapp, around line 37) compare the binding host with the scope host instead of `github.com`.

### `apps/backend/internal/orchestrator/executor/executor_credentials.go`
**What:** modified.
**Justified by:** E32
**Governed by:** Go limits.
**Specifics:** `issueGitHubRemoteContributionCredentialScope` and `issueGitHubContributionDestinationCredentialScope` (lines 680-753) accept any host equal to the attached repository's provider hostname and pass that host instead of `defaultGitHubHost` (lines 708, 750). Add `envKandevGitHubHosts = "KANDEV_GITHUB_HOSTS"`; `configureGitCredentialBrokerForRepositoriesWithProfileEnvAndBridge` (line 200) sets it after `removeHostGitHubCredentialHelpers` (line 213) to the sorted, comma-joined enterprise hosts of the attached GitHub repositories, or deletes it when there are none.

### `apps/backend/internal/orchestrator/executor/executor_host_gh_bridge.go`
**What:** modified.
**Justified by:** E7, E12
**Governed by:** Go limits.
**Specifics:** `hasExplicitGitHubTokenInProfile` (line 306) and `hasExplicitGitHubToken` (line 386) pick the token variables with `githubauth.TokenEnvForHost(host)` instead of comparing with `defaultGitHubHost`.

### `apps/backend/internal/githubauth/environment.go`
**What:** modified.
**Justified by:** E7
**Governed by:** Go limits.
**Specifics:** add exported `TokenEnvForHost(host string) (primary, fallback string)`: `GH_TOKEN`, `GITHUB_TOKEN` for `github.com` and `*.ghe.com`; `GH_ENTERPRISE_TOKEN`, `GITHUB_ENTERPRISE_TOKEN` otherwise. The executor bridge, shim, lifecycle, and script engine call it.

### `apps/backend/cmd/agentctl/github_cli_shim.go`
**What:** modified.
**Justified by:** E7, E36
**Governed by:** Go limits.
**Specifics:** after resolving the credential (line 69), build the child environment with `GH_HOST=<target host>` and the variable from `githubauth.TokenEnvForHost(host)` set to the password, and remove the other three token variables (lines 84-89). Keep `github.com` output identical to today except for `GH_HOST=github.com`. `parseGitHubCLIRepository` (line 214) keeps defaulting to `CredentialHostEnv`, then `github.com`.

### `apps/backend/internal/agent/runtime/lifecycle/container.go`, `executor_sprites_helpers.go`, `remote_github_env.go`
**What:** modified.
**Justified by:** E36
**Governed by:** Go limits.
**Specifics:** the container Git config (container.go:895-914) and the Sprites clone URL token injection (executor_sprites_helpers.go:35-60) iterate the hosts in `KANDEV_GITHUB_HOSTS` plus `github.com`: per host, add `url.https://<host>/.insteadOf` for `git@<host>:`, `<subdomain>@<host>:` for `.ghe.com` hosts, and `ssh://git@<host>/`, and a credential helper reading the variable from `TokenEnvForHost`. `remote_github_env.go` (line 21) forwards `KANDEV_GITHUB_HOSTS` and the enterprise token variables.
Unverified: could not inspect how the lifecycle manager receives `req.Env` for Docker and Sprites, because only the helper functions were read; the implementer confirms `KANDEV_GITHUB_HOSTS` reaches them.

### `apps/backend/internal/scriptengine/providers.go`
**What:** modified.
**Justified by:** E36
**Governed by:** Go limits.
**Specifics:** `github.auth_setup` (lines 370-400) emits the `credential.https://<host>.helper` and `gh config set git_protocol https --host <host>` lines for each host in `KANDEV_GITHUB_HOSTS` and for `github.com`, reading the token variable per host.

### `apps/backend/internal/agentctl/server/process/git_pr_providers.go` and `env_hermetic_test.go`
**What:** modified.
**Justified by:** E19
**Governed by:** `.agents/skills/tdd/references/backend-tests.md` (host-trust tests).
**Specifics:** `gitHubHostsEnv = "KANDEV_GITHUB_HOSTS"`; `detectPRProviderWithHosts(remoteURL, gitLabHost, gitHubHosts string)` replaces `detectPRProviderWithGitLabHost` (line 64); `isGitHubHost(host, configured string)` (line 90) also matches any listed host. Add `gitHubHostsEnv` to `ambientGitLabEnvVars` (env_hermetic_test.go:14) and adjust its comment. `createManagedContributionPR` (git.go:1596) uses the same detection.

### Tests
**What:** added.
**Justified by:** E7, E12, E13, E19, E36
**Governed by:** `CLAUDE.md` Testing; derived-environment contract rule in `.agents/skills/tdd/references/backend-tests.md`.
**Specifics:**
- `backendapp/git_credentials_hosts_test.go`: enterprise scope authorized when the repository host matches; denied on mismatch.
- `orchestrator/executor/executor_github_hosts_env_test.go`: drive `configureGitCredentialBrokerForRepositories` with public, enterprise, and GitLab repositories; assert `KANDEV_GITHUB_HOSTS` set or removed; contribution scopes on an enterprise host.
- `orchestrator/executor/executor_host_gh_bridge_test.go` (table at line 424): `GH_TOKEN` for an `octo.ghe.com` host now suppresses the bridge; `GH_ENTERPRISE_TOKEN` for `ghe.example.com` still does.
- `cmd/agentctl/github_cli_shim_hosts_test.go`: child environment for the three host kinds.
- `agent/runtime/lifecycle` and `scriptengine` tests: helper and rewrite entries per host.
- `agentctl/server/process/git_pr_providers_enterprise_test.go`: detection with and without the variable.

## Verification

```bash
# From apps/backend:
go test -trimpath -tags fts5 -race ./internal/backendapp ./internal/orchestrator/executor ./internal/githubauth ./internal/agent/runtime/lifecycle ./internal/scriptengine ./internal/agentctl/server/process ./cmd/agentctl -count=1
```

Then `bash scripts/lint-go-changed <changed Go files>` from the repository root.

## Dependencies

Tasks 04 and 05.

## Risks

- Remote executor helpers run in user-managed images; a missing `gh` keeps
  the existing fallback behavior.
- The `.ghe.com` token rule rests on the GitHub CLI manual (E7, single
  source); the shim test pins it.

## Parallelism

`sequential`

## Results

Pending.
