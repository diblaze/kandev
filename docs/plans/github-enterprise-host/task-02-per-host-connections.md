---
id: "02-per-host-connections"
title: "Per-host workspace connections"
status: pending
wave: 2
depends_on:
  - "01-host-aware-workspace-client"
plan: "plan.md"
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006
acceptance_criteria:
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.1
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.2
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.3
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.4
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.6
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.7
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.8
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.9
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.2
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.6
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.1
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.2
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.3
  - AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.5
system_design:
  - ../../specs/integrations/system-design/github-enterprise-host.md
---

# Task 02: Per-Host Workspace Connections

## Summary

Key workspace and personal connections by host, resolve credentials by
`(workspace, host)`, fail closed for a host with no connection, and expose
per-host connection management over HTTP. App registration creation and
import accept a host. Existing rows become `github.com` connections.

## In scope

- Schema rebuild of `github_workspace_connections`,
  `github_user_connections`, and `github_user_connection_versions`, with
  triggers, on SQLite and PostgreSQL.
- Store, personal repository, resolver, connection service, status, and
  controller changes.
- PAT, CLI, and App installation connections for any host.
- New-workspace defaults for every active `gh` host account.
- Webhook dispatch filtered by registration host.

## Out of scope

- Threading the host through GitHub operations (task 03). Until then every
  operation resolves `github.com`, so behavior for existing users is
  unchanged.
- The settings UI (task 07).

## Acceptance

- Upgrading a database with one connection and one personal connection per
  workspace keeps both rows, now keyed with `github_host = 'github.com'`, and
  replaying the migration is a no-op.
- A workspace can hold `github.com` and `octo.ghe.com` connections; replacing
  or deleting one leaves the other row and generation unchanged.
- `Resolve` with host `ghe.example.com` and no row returns
  `ErrGitHubHostNotConnected` and calls no token source.
- Status for a workspace lists both hosts; status with `host=octo.ghe.com`
  projects that connection.

## Changes

### `apps/backend/internal/github/store.go`
**What:** modified.
**Justified by:** E23, E24, E39
**Governed by:** replayable migrations and table-rebuild rules (`apps/backend/AGENTS.md:266,284`).
**Specifics:**
- Final schema: `github_workspace_connections` `PRIMARY KEY (workspace_id, github_host)` (line 266); `github_user_connections` gains `github_host TEXT NOT NULL DEFAULT 'github.com'` and `PRIMARY KEY (workspace_id, user_id, github_host)` (line 293); same for `github_user_connection_versions` (line 356).
- Triggers (lines 316-354 and `postgresGitHubAuthTriggers` at line 1129) match `workspace.github_host = NEW.github_host` and `personal.github_host = OLD.github_host` in addition to workspace and registration.
- Add `migrateGitHubConnectionsToHostKey()` called from `initSchemaUpgrades` (line 647): when `tableSQLContains` finds the old single-column key, rebuild each table in one transaction with `rebuildIfHasLegacyConstraint` (line 1745), copying every column and setting `github_host = 'github.com'` on personal rows. PostgreSQL drops and re-adds the primary key by its looked-up constraint name.
- `seedLegacyWorkspaceConnections` (line 1309) uses `ON CONFLICT(workspace_id, github_host)`.

### `apps/backend/internal/github/store_connections.go`
**What:** modified.
**Justified by:** E23, E24
**Governed by:** Go limits; PostgreSQL behavior tests for changed dialect-sensitive methods (`apps/backend/AGENTS.md:270`).
**Specifics:**
- `GetWorkspaceConnection(ctx, workspaceID, host)` (line 68) adds `AND github_host = ?`; add `ListWorkspaceConnections(ctx, workspaceID)` ordered by host.
- `UpsertWorkspaceConnection` (line 103) uses `ON CONFLICT(workspace_id, github_host)`.
- `DeleteWorkspaceConnection(ctx, workspaceID, host)` (line 174).
- `ListWorkspaceConnectionsByAppInstallation` (line 89) is unchanged: a registration has one host.
- User connection methods (lines 181-330) take and match `github_host`; `ListUserConnectionsByWorkspace` returns every host.

### `apps/backend/internal/github/models.go`
**What:** modified.
**Justified by:** E24
**Governed by:** Go limits.
**Specifics:** `UserConnection` gains `GitHubHost string \`json:"github_host" db:"github_host"\``.

### `apps/backend/internal/github/errors.go`
**What:** modified.
**Justified by:** system design, Errors.
**Governed by:** Go limits.
**Specifics:** `type HostNotConnectedError struct{ Host string }` with `Error()` `GitHub host "<host>" is not connected for this workspace` and `Unwrap() error { return ErrGitHubNotConfigured }`; `var ErrGitHubHostNotConnected = &HostNotConnectedError{}` for `errors.As`. `writeGitHubAuthError` adds `"host"` to the 503 body when the error is this type.

### `apps/backend/internal/github/auth_principal.go`
**What:** modified.
**Justified by:** E27
**Governed by:** Go limits.
**Specifics:** `ResolveCredentialRequest` gains `Host string` (line 41). `AuthPrincipal` gains `GitHubHost`.

### `apps/backend/internal/github/auth_resolver.go`
**What:** modified.
**Justified by:** E5, E23
**Governed by:** Go limits.
**Specifics:**
- `workspaceConnectionReader` methods take a host (lines 17-20).
- `Resolve` normalizes `req.Host` with `normalizeGitHubHost` and rejects an invalid one.
- `resolveAtEpoch` (line 165) loads `(workspace, host)` and returns `&HostNotConnectedError{Host: host}` when nil.
- `credentialCacheKey` gains `host`; `automationCacheKey` (line 282) sets it.
- `resolvePAT` and `resolveGHCLI` build `NewTokenClient(...).WithHost(connection.GitHubHost)`; set `Principal.GitHubHost`.
- `resolveConfiguredPersonal` passes the host to `GetUserConnection`.

### `apps/backend/internal/github/personal_connection_repository.go`, `personal_auth_service.go`, `service_app_auth_connection_store.go`, `app_installation_service.go`
**What:** modified.
**Justified by:** E24, E25
**Governed by:** Go limits.
**Specifics:**
- Every `GetWorkspaceConnection` and `GetUserConnection` call passes the host of the flow's App registration (`AuthFlow.AppRegistrationID` to registration `GitHubHost`), read once at flow start and completion.
- `newInstallationConnection` (app_installation_service.go:238) sets `GitHubHost` to the registration host.
- `UserConnection` writes set `GitHubHost` from the registration.
- `TransitionWorkspacePersonalConnections` takes the host and transitions only that host's personal rows.

### `apps/backend/internal/github/service_connections.go`
**What:** modified.
**Justified by:** E3, E23
**Governed by:** Go limits.
**Specifics:**
- `SetWorkspaceConnection` normalizes `req.Host` once and returns its error before any validation. All loads, generation checks, locks, and upserts use `(workspace, host)`. The mutation lock key becomes `workspaceID + "\x00" + host`.
- The PAT branch validates with `s.tokenClientFactory(host, token)` and stores the host. The PAT secret key becomes `WorkspacePATSecretKey(workspaceID, host)`; for `github.com` it returns the existing key so stored secrets keep working.
- `validateWorkspaceToken(ctx, host, token)`; `tokenClientFactory func(host, token string) Client` (`service.go:172,260`) defaults to `NewPATClient(token).WithHost(host)`.
- `DeleteWorkspaceConnection(ctx, workspaceID, host)` (line 331) deletes only that host's row and secrets.
- `WorkspaceConnectionFingerprint` (line 50) joins `host:status:generation` of every row, sorted.
- `CopyWorkspaceConnection` (line 445) copies every host row.
- After a successful commit, call the optional `hostConnectedObserver(ctx, workspaceID, host)` (set by task 04) outside the lock.
- Add `func (s *Service) WorkspaceGitHubHosts(ctx, workspaceID string) ([]string, error)` returning connected hosts; no credential.

### `apps/backend/internal/github/service_app_auth.go`
**What:** modified.
**Justified by:** E35
**Governed by:** Go limits.
**Specifics:**
- `GetWorkspaceAuthStatus(ctx, workspaceID, userID, host string)` and `RefreshWorkspaceRateLimit` (lines 463-475) project the connection and personal connection for `host`.
- `WorkspaceAuthStatus` gains `Hosts []WorkspaceHostStatus` with `{github_host, source, login, installation_account_login, status, last_error, personal_login}` for every row.
- Lines 493 and 693 load by host.

### `apps/backend/internal/github/controller.go` and `controller_auth.go`
**What:** modified.
**Justified by:** E35
**Governed by:** Go limits.
**Specifics:** `httpGetStatus` (controller.go:144), `httpGetWorkspaceConnection` (controller_auth.go:35), and `httpDeleteWorkspaceConnection` (:91) read `host` from the query; `httpListCLIAccounts` (:46) marks `Selected` per host from `Hosts`.

### `apps/backend/internal/github/app_registration_import.go` and `app_registration_lifecycle_service.go`
**What:** modified.
**Justified by:** E25
**Governed by:** Go limits.
**Specifics:**
- `validImportScalarFields` (line 168) accepts any host that passes `normalizeGitHubHost`, replacing the `github.com` equality.
- `AppRegistrationManifestStartRequest` (line 76) gains `GitHubHost`; the flow stores it (add column `github_host TEXT NOT NULL DEFAULT 'github.com'` to `github_app_registration_flows`, `store.go:498`); line 358 stores the flow host instead of `defaultGitHubAppHost`; conversion calls `Convert(ctx, flow.GitHubHost, code)`.
- The public HTTPS origin validation is unchanged.

### `apps/backend/internal/github/webhook_service.go`
**What:** modified.
**Justified by:** E25
**Governed by:** Go limits.
**Specifics:** installation lifecycle and repository events load connections by registration and installation, then keep only rows whose `GitHubHost` equals the registration host.
Unverified: could not inspect every webhook handler body, because only the store query signatures were read; the implementer lists each connection lookup in this file.

### `apps/backend/internal/github/workspace_defaults.go`
**What:** modified.
**Justified by:** E14
**Governed by:** Go limits.
**Specifics:** replace `activeGHAccount` (line 117) with `activeGHAccountsByHost` returning one valid active account per host; `InitializeWorkspaceDefaults` commits a CLI connection for each, skipping hosts that already have a row, and logs and continues on a per-host validation failure.

### `apps/backend/internal/github/credential_broker.go`
**What:** modified.
**Justified by:** E13
**Governed by:** Go limits.
**Specifics:** `Binding` (line 250) replaces the `github.com` check with `normalizeGitHubHost(scope.Host)` and loads the connection for that host through `issueConnection(ctx, workspaceID, host)`; `Resolve` (line 312) sets `Host` on the request. `RefreshScope` uses the scope host.

### `apps/backend/internal/github/mock_auth.go` and `mock_controller.go`
**What:** modified.
**Justified by:** E40
**Governed by:** Go limits.
**Specifics:** factory signature `func(host, token string) Client`; mock connection writes and reads (mock_controller.go:106-112, 312) carry the request host.

### Tests
**What:** added and modified.
**Justified by:** E23, E24, E39
**Governed by:** `CLAUDE.md` Testing; `apps/backend/AGENTS.md:266,284,287`.
**Specifics:**
- `store_connection_host_migration_test.go`: build the old schema with `NewWithDB`, insert a workspace connection, a personal connection, and a version row; run the migration; assert values, timestamps, and triggers; run it again and assert no change; inject a failure after each rebuild step and assert rollback. Same matrix behind `KANDEV_TEST_POSTGRES_DSN`.
- `service_connections_hosts_test.go`: two hosts coexist; replacing one keeps the other's generation; deleting one keeps the other; invalid host rejected before token resolution; failed validation keeps the previous row; PAT secret key for `github.com` equals the old key.
- `auth_resolver_hosts_test.go`: host routing, cache isolation between hosts, and `HostNotConnectedError` with no token-source call.
- `workspace_defaults_hosts_test.go`: two active accounts produce two connections.
- `credential_broker_hosts_test.go`: binding and resolve for an enterprise host; denial for a host without a row.
- Update existing tests for the new signatures; use `rg -n 'GetWorkspaceConnection\(|GetUserConnection\(|DeleteWorkspaceConnection\(|tokenClientFactory' apps/backend/internal/github`.

## Verification

```bash
# From apps/backend:
go test -trimpath -tags fts5 -race ./internal/github -run 'Test(StoreConnectionHostMigration|WorkspaceConnectionHosts|ResolverHosts|WorkspaceDefaultsHosts|CredentialBrokerHosts)' -count=1
KANDEV_TEST_POSTGRES_DSN=<dsn> go test -trimpath -tags fts5 -race ./internal/github -run 'TestStoreConnectionHostMigration' -count=1
go test -trimpath -tags fts5 -race ./internal/github ./internal/backendapp -count=1
go run ./cmd/sqlguard ./internal
go test -race ./internal/persistence/storeconformance -count=1
```

Then `bash scripts/lint-go-changed <changed Go files>` from the repository root.

## Dependencies

Task 01.

## Risks

- Trigger rewrites must keep both SQLite and PostgreSQL variants in step.
- Callers outside `internal/github` that use `GetWorkspaceAuthStatus` must
  pass a host; compile errors find them.

## Parallelism

`sequential`

## Results

Pending.
