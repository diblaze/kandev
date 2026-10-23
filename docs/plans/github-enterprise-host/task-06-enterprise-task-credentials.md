---
id: "06-enterprise-task-credentials"
title: "Task Git and CLI credentials for enterprise hosts"
status: done
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
Confirmed during implementation: `LaunchAgentRequest.Env` reaches `ExecutorCreateRequest.Env` (`manager_launch.go`), Docker passes it whole as `ContainerConfig.Credentials`, and Sprites passes it to the agent. SSH forwards an allowlist, so the forwarding change lives in `executor_ssh_operations.go` (`sshRemoteAgentCredentialEnvKeys`) and `executor_ssh_remote_contribution.go`, not `remote_github_env.go` (see Results).

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

Implemented test-first.

What was done:

- `githubauth`: `TokenEnvForHost(host)` (`GH_TOKEN`/`GITHUB_TOKEN` for
  `github.com` and `*.ghe.com`, `GH_ENTERPRISE_TOKEN`/`GITHUB_ENTERPRISE_TOKEN`
  otherwise), `GitHubHostsEnv = "KANDEV_GITHUB_HOSTS"`, and
  `EnterpriseHosts(raw)`, which returns the distinct lowercase listed hosts
  without `github.com` and drops any entry that is not a bare hostname, so a
  listed host is safe in a Git config key, a URL, and a shell word.
- Broker authorization (`backendapp`): every scope with provider `github`
  goes through `authorizeGitHubRepository`, which now takes a
  `gitHubRepositoryScope` with the scope host. The repository's
  `provider_host` (via `github.HostFromProviderHost`) must equal the scope
  host; contribution destinations need source and target on the scope host;
  remote-contribution forks need their source on it. The exported
  `AuthorizeGitHubRepository`/`...WithIdentity` (no host in their interface)
  keep `github.com`. `isPublicGitHubProviderHost` stays only for the Kandev
  repository check.
- Executor: contribution and destination scopes are issued on the attached
  repository's clone-identity host and rejected when the binding names
  another host. `setGitHubHostsEnv` publishes the sorted enterprise hosts of
  the attached GitHub repositories after `removeHostGitHubCredentialHelpers`,
  and deletes the variable when there are none (stale values included).
- Host bridge: `hasExplicitGitHubToken` and `hasExplicitGitHubTokenInProfile`
  use `TokenEnvForHost`.
- Managed `gh` shim: the child gets `GH_HOST=<lease host>` and only the lease
  host's token variable; the other three token variables are removed. A
  `--hostname` other than the lease host is refused before the credential is
  resolved.
- agentctl: `gitHubHostsEnv`, `detectPRProviderWithHosts`, and
  `isGitHubHost(host, configured)` treat a listed host as GitHub; the
  variable joins the `TestMain` scrub list. `createManagedContributionPR`
  already used `g.detectPRProvider`.
- Remote executors with profile tokens: `containerGitConfig` (Docker) adds,
  for each listed host whose token variable is set, the SSH-to-HTTPS
  rewrites (`git@<host>:`, the tenant user form for `.ghe.com`,
  `ssh://git@<host>/`) and a helper reading that host's variables;
  `injectGitHubTokenIntoCloneURL` (Docker, Sprites, Kubernetes) injects only
  the URL host's variable for `github.com` and listed hosts; SSH forwards
  `KANDEV_GITHUB_HOSTS`, `GH_ENTERPRISE_TOKEN`, and
  `GITHUB_ENTERPRISE_TOKEN`, and adds a helper per host with a token;
  `github.auth_setup` emits the helper and `gh config set git_protocol` per
  host. `scriptengine.GitHubSSHPrefixes(host)` is the one owner of the
  rewrite prefixes (it uses `repoclone.SSHUser`).

Verification (from `apps/backend`, `GOMAXPROCS=4 GOMEMLIMIT=6GiB`, `-p 2`,
one run at a time):

- `go test -trimpath -tags fts5 ./internal/backendapp -count=1`: `ok 130.791s`.
- `go test -trimpath -tags fts5 ./internal/orchestrator/executor -count=1`:
  29 failing tests, the same 29 names as the untouched baseline run, all in
  the known environmental baseline list; no new failure.
- `go test -trimpath -tags fts5 ./internal/agent/runtime/lifecycle -count=1`: `ok 158.625s`.
- `go test -trimpath -tags fts5 ./internal/agentctl/server/process -count=1`: `ok 222.015s`.
- `go test -trimpath -tags fts5 ./cmd/agentctl -count=1`: `ok 4.015s`;
  `./internal/scriptengine`: `ok 0.728s`; `./internal/githubauth`: `ok 0.005s`.
- With `-race`, one package at a time: `internal/githubauth` `ok 1.027s`,
  `internal/scriptengine` `ok 1.855s`, `cmd/agentctl` `ok 8.078s`,
  `internal/agentctl/server/process` `ok 431.722s`,
  `internal/agent/runtime/lifecycle` `ok 251.289s`. `internal/backendapp` and
  `internal/orchestrator/executor` not run with `-race` (memory limit).
- `go build -tags fts5 ./...` and `go vet` of every touched package: clean;
  `gofmt -l`: none.
- From the repository root, `bash scripts/lint-go-changed <files>` per group
  (githubauth, scriptengine, cmd/agentctl, agentctl process, lifecycle,
  backendapp, orchestrator/executor): `0 issues.` each.

Tests that passed on their first run, then proven by breaking the code: the
host-mismatch denials in `TestGitCredentialHostsDenyScopesOffTheRepositoryHost`
(repository host check made host-blind, fork host check and destination host
check removed), `TestEnterprisePRProviderLeavesUnlistedHostsUnsupported`
(any non-empty list accepted), `TestGitHubAuthHostsIgnoreUnsafeListedHosts`
(raw split instead of `EnterpriseHosts`),
`TestContainerGitHubHostsSkipHostsWithoutTheirToken` (token gate removed),
and the never-reaches, unlisted, and lookalike rows of
`TestCloneURLGitHubHostsInjectOnlyTheHostTokenVariable` (swapped token
variables, fixed host list, prefix without the trailing slash). The positive
canonical tenant and server rows of
`TestGitCredentialHostsAuthorizeEnterpriseScopesOnTheRepositoryHost` passed
before through the old record-identity path and are kept as acceptance
guards; the stale-identity row went red there and is the discriminating test.

Deviations:

- SSH forwarding changed in `executor_ssh_operations.go` and
  `executor_ssh_remote_contribution.go`; `remote_github_env.go` is the
  long-lived agentctl broker env and needed no change, because PR detection
  reads the instance agent env.
- `selected_checkout.go`: a pull-request checkout now also scrubs
  `GH_ENTERPRISE_TOKEN` and `GITHUB_ENTERPRISE_TOKEN` (agent env and the
  `unset` line), since those tokens now configure Git. Not listed.
- The shim refuses `gh ... --hostname <other host>`. Not listed: `GH_TOKEN`
  applies to every `*.ghe.com` tenant, so without it a lease token could be
  sent to another tenant.
- The shim removes all four token variables for `github.com` too (before
  only `GITHUB_TOKEN`), so `github.com` output differs from before by
  `GH_HOST` and the removed enterprise variables.
- Enterprise hosts get SSH rewrites in the Docker container only when their
  token variable is set; `github.com` rewrites stay unconditional as before.
  `github.auth_setup` and the SSH env add helpers, not rewrites.
- `rewriteGitHubSSHToHTTPS` takes the host; its existing test passes
  `github.com`.
- `DescribeTaskGitCredentialPolicy` is unchanged: the mode is per workspace,
  and its method and actor fields describe the `github.com` projection only.

Deferred:

- Enterprise SSH-to-HTTPS rewrites in `github.auth_setup` (SSH and Sprites
  default scripts rewrite only `github.com`). Reopen when an SSH or Sprites
  task with an SSH-form enterprise remote and only a profile token fails to
  fetch or push.
- The agentctl PR-provider table in `internal/agentctl/AGENTS.md` (task 08).

Notes:

- With profile tokens, `GH_TOKEN` is by the GitHub CLI rule the token for
  `github.com` and every `*.ghe.com` tenant, so an attached `octo.ghe.com`
  repository makes Docker, SSH, and prepare scripts answer that host with the
  `GH_TOKEN` value. Kandev cannot tell which host a profile token belongs to;
  managed leases are per host and never cross.

Unverified: `internal/backendapp` and `internal/orchestrator/executor` under
`-race`.

### Follow-up: profile token bound to `GH_HOST` (2026-10-26)

Task 06 offered a profile `GH_TOKEN` to `github.com` and every attached
`*.ghe.com` host, following the GitHub CLI rule. Because Kandev cannot tell
which host such a token was issued for, a `github.com` token could reach a
tenant and the reverse. The binding is now explicit:

- `githubauth.ProfileTokenEnvForHost(ghHost, host)` and `ProfileTokenHost`
  bind `GH_TOKEN`/`GITHUB_TOKEN` to the profile's `GH_HOST` (`github.com` when
  unset) and `GH_ENTERPRISE_TOKEN`/`GITHUB_ENTERPRISE_TOKEN` to Enterprise
  Server hosts (only the named one when `GH_HOST` names a server). A malformed
  or secret-backed `GH_HOST` binds no token.
- Applied in the container Git config and SSH token helpers, the SSH
  environment allowlist (now forwards `GH_HOST`), clone URL token injection,
  the `github.auth_setup` prepare script, and the executor host bridge check
  `hasExplicitGitHubToken(env, profileEnvVars, host)`.
- The real-git prepare-script test runs in its temporary HOME, so the
  repository's own credential helper cannot answer it.
- Tests: `internal/githubauth/profile_token_env_test.go` and
  `internal/orchestrator/executor/executor_host_gh_bridge_token_binding_test.go`
  cover the bound host, `github.com`, another tenant, an Enterprise Server
  host, a malformed `GH_HOST`, a profile `GH_HOST` overriding the request's,
  and a secret-backed `GH_HOST`.
- Not covered by Kandev: an agent running `gh` itself still follows the CLI's
  own rule for `GH_TOKEN`.
