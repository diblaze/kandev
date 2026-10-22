---
id: "01-host-aware-workspace-client"
title: "Host-aware GitHub clients"
status: done
wave: 1
depends_on: []
plan: "plan.md"
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006
acceptance_criteria:
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.5
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.1
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.4
system_design:
  - ../../specs/integrations/system-design/github-enterprise-host.md
---

# Task 01: Host-Aware GitHub Clients

## Summary

Add host validation and the host endpoint record, and make every GitHub HTTP
client take a host: token (PAT, CLI, App user), App, OAuth, and manifest
conversion clients. `gh auth token` accepts any valid host. No routing or
persistence changes yet; every caller still passes `github.com`.

## In scope

- `normalizeGitHubHost`, `githubEndpoints`, and `WebHostFromURL`.
- Client constructors and methods that build REST, GraphQL, web, OAuth, or App
  URLs.
- `gh auth token --hostname <host>` for any valid host.

## Out of scope

- Which host a caller passes (tasks 02 and 03).
- The deployment-level `GHClient` health check (`gh_client.go:127,144`), which
  serves only the `github.com` `legacy_shared` source.

## Acceptance

- A client bound to `octo.ghe.com` calls `https://api.octo.ghe.com/user` and `https://api.octo.ghe.com/graphql`; bound to `ghe.example.com` it calls `https://ghe.example.com/api/v3/user` and `https://ghe.example.com/api/graphql`; bound to `github.com` or empty it calls today's URLs.
- OAuth, App, manifest registration, conversion, and install URLs follow the system design table for all three host kinds.
- `resolveGHAccountToken` runs `gh auth token --hostname octo.ghe.com --user <login>` and rejects `https://x`, `x/y`, `x:443`, `a@b`, `-x`, and whitespace without running `gh`.

## Changes

### `apps/backend/internal/github/host.go`
**What:** added.
**Justified by:** E1, E6, E26
**Governed by:** Go limits (`apps/backend/AGENTS.md:292`).
**Specifics:**
- `func normalizeGitHubHost(raw string) (string, error)`: trim, lowercase, `www.github.com` to `github.com`, empty to `defaultGitHubHost`; reject a leading `-`, any of `/:@`, whitespace, or `url.Parse("https://"+h).Hostname() != h`; error `fmt.Errorf("invalid GitHub host %q", raw)`.
- `type githubEndpoints struct { Web, REST, GraphQL, AppInstallPrefix string }` and `func endpointsForHost(host string) githubEndpoints` with the three rows of the system design.
- Exported `func WebHostFromURL(raw string) (string, error)`: hostname of an `https` URL with no user info or port that passes `normalizeGitHubHost`; any other URL is an error, so a malformed URL fails closed instead of routing to the `github.com` connection (REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003).
- Exported `func HostFromProviderHost(providerHost string) (string, error)`: hostname of `https://<host>` or a bare host, normalized. An empty or malformed value is an unknown identity and an error, matching the fail-closed rule for legacy rows in `apps/backend/AGENTS.md`.

### `apps/backend/internal/github/host_test.go`
**What:** added.
**Justified by:** E6, E26
**Governed by:** `CLAUDE.md` Testing.
**Specifics:** table tests for the four helpers, including each rejected form, uppercase input, `www.github.com`, and the three endpoint rows.

### `apps/backend/internal/github/gh_accounts.go`
**What:** modified.
**Justified by:** E1
**Governed by:** Go limits.
**Specifics:** in `resolveGHAccountToken` (line 185) replace lines 188-193 with `host, err := normalizeGitHubHost(host)` and return the error before any runner call.

### `apps/backend/internal/github/gh_accounts_test.go`
**What:** modified.
**Justified by:** E1
**Governed by:** `CLAUDE.md` Testing.
**Specifics:** rename `TestResolveGHAccountTokenRejectsUnsupportedHost` (line 294) to `TestResolveGHAccountTokenRejectsInvalidHost` with input `https://git.example.com`; add `TestResolveGHAccountTokenUsesEnterpriseHost` asserting args `{"auth","token","--hostname","octo.ghe.com","--user","alice"}`.

### `apps/backend/internal/github/token_client.go`
**What:** modified.
**Justified by:** E4, E5
**Governed by:** host-carrying constructor shape from GitLab (E41).
**Specifics:** add `endpoints githubEndpoints` to `TokenClient`; `NewTokenClient` sets `endpointsForHost(defaultGitHubHost)`; add `func (c *TokenClient) WithHost(host string) *TokenClient`. `NewAppUserTokenClient` gains a `host` argument.

### `apps/backend/internal/github/pat_client.go`
**What:** modified.
**Justified by:** E4
**Governed by:** Go limits.
**Specifics:** use `c.endpoints.REST` at lines 934, 970, 999, 1030, 1054; `parseNextLink(header, apiBase string)` strips `apiBase` (line 1096), called with `c.endpoints.REST` at line 1076.

### `apps/backend/internal/github/graphql.go`
**What:** modified.
**Justified by:** E4
**Governed by:** Go limits.
**Specifics:** line 705 posts to `c.endpoints.GraphQL`.

### `apps/backend/internal/github/app_client.go`
**What:** modified.
**Justified by:** E25
**Governed by:** Go limits.
**Specifics:** `NewAppClient(host string, appID int64, privateKeyPEM []byte)` sets `baseURL` from `endpointsForHost(host).REST` (line 117).

### `apps/backend/internal/github/oauth_flow.go`
**What:** modified.
**Justified by:** E25, E26
**Governed by:** Go limits.
**Specifics:** `NewGitHubOAuthClient(host, clientID, clientSecret string)` sets `webBaseURL` and `apiBaseURL` from the endpoint record (line 260).

### `apps/backend/internal/github/deployment_app_conversion_client.go`
**What:** modified.
**Justified by:** E25
**Governed by:** Go limits.
**Specifics:** `Convert(ctx, host, conversionCode string)` posts to `endpointsForHost(host).REST + "/app-manifests/<code>/conversions"`; drop the fixed `apiBaseURL` (line 91) except as a test override field. Update the `DeploymentAppManifestConverter` interface (`app_registration_helpers.go:21`).

### `apps/backend/internal/github/deployment_app_manifest.go`
**What:** modified.
**Justified by:** E25, E26
**Governed by:** Go limits.
**Specifics:** `manifestRegistrationURL(host string, ownerType ManifestOwnerType, ownerLogin string)` (line 303) builds on `endpointsForHost(host).Web`; caller at line 217 passes the flow host.

### `apps/backend/internal/github/app_installation_service.go`
**What:** modified.
**Justified by:** E25, E26
**Governed by:** Go limits.
**Specifics:** `AppInstallationConfig` gains `Host`; the install URL (lines 124-128) uses `endpointsForHost(Host).Web + AppInstallPrefix + slug + "/installations/new"`.

### `apps/backend/internal/github/personal_auth_service.go`
**What:** modified.
**Justified by:** E25, E26
**Governed by:** Go limits.
**Specifics:** `PersonalAuthConfig` gains `Host`; the authorize URL (line 136) uses its web base.

### `apps/backend/internal/github/service_app_auth.go`
**What:** modified.
**Justified by:** E25
**Governed by:** Go limits.
**Specifics:** pass `resolved.Registration.GitHubHost` to `NewAppClient` (line 156), `NewGitHubOAuthClient` (line 174), `PersonalAuthConfig`, and `AppInstallationConfig` (lines 178-193).

### `apps/backend/internal/github/app_registration_import.go`
**What:** modified.
**Justified by:** E25
**Governed by:** Go limits.
**Specifics:** pass the request host to `NewAppClient` (line 215). The `github.com`-only check at line 168 is lifted in task 02, together with the request field.

### `apps/backend/internal/github/client_host_test.go`
**What:** added.
**Justified by:** E4, E6, E25, E26
**Governed by:** `CLAUDE.md` Testing; new file instead of growing `pat_client_test.go`.
**Specifics:** recording `http.RoundTripper`; for `github.com`, `octo.ghe.com`, and `ghe.example.com` assert the URLs from `GetAuthenticatedUser`, `ExecuteGraphQL`, `AppClient` installation token mint, OAuth code exchange, manifest conversion, `manifestRegistrationURL` for user and organization owners, the install URL, and the personal authorize URL. Add a `parseNextLink` case with a `ghe.example.com/api/v3` next link.

### Existing tests that construct these clients
**What:** modified.
**Justified by:** E40
**Governed by:** `CLAUDE.md` Testing.
**Specifics:** `pat_client_contents_test.go:333` passes `githubAPIBase`; every `NewAppClient`, `NewGitHubOAuthClient`, `NewAppUserTokenClient`, and converter fake in `internal/github` tests passes `defaultGitHubHost`. Use `rg -n 'NewAppClient\(|NewGitHubOAuthClient\(|NewAppUserTokenClient\(|Convert\(' apps/backend/internal/github` to find them.

## Verification

```bash
# From apps/backend:
go test -trimpath -tags fts5 -race ./internal/github -run 'Test(NormalizeGitHubHost|EndpointsForHost|WebHostFromURL|HostFromProviderHost|ResolveGHAccountToken|ClientHost|ParseNextLink)' -count=1
go test -trimpath -tags fts5 -race ./internal/github -count=1
# From the repository root:
bash scripts/lint-go-changed apps/backend/internal/github/*.go
```

## Dependencies

None.

## Risks

- A struct literal of `TokenClient` would have empty endpoints; all
  constructions go through `NewTokenClient`.

## Parallelism

`sequential`

## Results

Done. `internal/github` passes in full and `scripts/lint-go-changed` reports
0 issues. Deviations from the specifics above:

- `WebHostFromURL` and `HostFromProviderHost` return an error instead of
  `github.com` (see their entries).
- `endpointsForHost("")` returns the `github.com` endpoints, so a client built
  with an empty host never produces `https:///...` URLs.
- The install and authorize URLs are built by `appInstallationURL` and
  `oauthAuthorizeURL`, which the tests exercise directly.
- Registration flows and personal connections carry no host yet; their callers
  pass `github.com` until task 02 adds the column.
