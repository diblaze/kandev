---
status: draft
system: integrations
requirements:
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006
  - REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007
created: 2026-10-22
owners:
  - kandev
---

# GitHub Enterprise Host System Design

## Purpose and boundaries

A workspace GitHub connection becomes a per-host connection: the workspace
holds at most one automation connection for each GitHub host. Every credential
resolution names a host, and every item that leads to a GitHub call carries
its host. The connection table already stores `github_host`, App registrations
already store `github_host` with `UNIQUE (github_host, app_id)`, and
repositories already store `provider_host`; this design makes those values
authoritative.

This replaces the "one automation connection per workspace" boundary in
[Workspace GitHub Authentication](github-authentication-03.md) and keeps every
other rule there: connection sources, credential generations, task credential
policy, registration catalog reuse, and the public HTTPS origin rule.

It uses, but does not own, the executor host `gh` bridge
(`REQ-INTEGRATIONS-GITHUB-AUTHENTICATION-002`) and the GitLab per-workspace
host pattern.

## Requirement mapping

| Requirement | Design sections |
| --- | --- |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001` | [Host value](#host-value), [Connection model](#connection-model), [Connection API](#connection-api) |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002` | [Credential resolution](#credential-resolution), [Item host](#item-host), [Host-scoped watches and sources](#host-scoped-watches-and-sources) |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003` | [Repository identity](#repository-identity) |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004` | [Item host](#item-host), [Links](#links) |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005` | [Task credentials](#task-credentials) |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006` | [Host endpoints](#host-endpoints), [GitHub App flows](#github-app-flows) |
| `REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007` | [Frontend](#frontend) |

## Data and contracts

### Host value

A host is a lowercase bare hostname such as `octo.ghe.com` or
`ghe.example.com`. An empty value means `github.com`. A value that contains a
scheme, `/`, `:`, `@`, whitespace, or a leading `-` is rejected before any
subprocess or request, so it can neither become a `gh` flag nor alter a URL.
Ports are not supported. `www.github.com` normalizes to `github.com`.

Connections, App registrations, watches, and sources store the bare
hostname. Repository rows keep the origin form `https://<host>` in
`provider_host`.

### Host endpoints

| Host | Web | REST | GraphQL | App install path |
| --- | --- | --- | --- | --- |
| `github.com` | `https://github.com` | `https://api.github.com` | `https://api.github.com/graphql` | `/apps/<slug>/installations/new` |
| `<sub>.ghe.com` | `https://<sub>.ghe.com` | `https://api.<sub>.ghe.com` | `https://api.<sub>.ghe.com/graphql` | `/apps/<slug>/installations/new` |
| other host | `https://<host>` | `https://<host>/api/v3` | `https://<host>/api/graphql` | `/github-apps/<slug>/installations/new` |

The `.ghe.com` row is GitHub Enterprise Cloud with data residency; the last
row is GitHub Enterprise Server. OAuth authorization and token exchange use
`<web>/login/oauth/...`. App manifest registration uses
`<web>/settings/apps/new` or `<web>/organizations/<org>/settings/apps/new`.
One package-level function maps a host to this record; every client takes the
record instead of a constant.

### Connection model

`github_workspace_connections` primary key becomes
`(workspace_id, github_host)`. `github_user_connections` and
`github_user_connection_versions` gain `github_host` and key on
`(workspace_id, user_id, github_host)`. The personal-connection triggers match
on workspace, host, and App registration. Existing rows keep
`github_host = 'github.com'`, so an upgrade changes no behavior.

`github_task_prs`, `github_pr_watches`, `github_review_watches`,
`github_issue_watches`, `workflow_sync_configs`, and the Office config sync
table gain `github_host TEXT NOT NULL DEFAULT 'github.com'`. Automation
trigger repositories carry an optional `host` in their JSON config.

### Connection API

- `GET /api/v1/github/status?workspace_id=&host=` returns the status of one
  host connection (default `github.com`) and a `hosts` array with one entry
  per host connection: host, source, login or installation account, status,
  and last error. Existing top-level fields keep describing the requested
  host.
- `PUT /api/v1/github/workspace-connection` already carries `host`; the PAT
  source now honors it.
- `DELETE /api/v1/github/workspace-connection?workspace_id=&host=` removes
  one host connection.
- App manifest start and import requests carry `github_host`; installation
  and personal flows take the host from the registration.
- Every route and WebSocket action that names `owner` and `repo`, and every
  search route, accepts `host`, defaulting to `github.com`.

### Errors

`HostNotConnectedError` wraps `ErrGitHubNotConfigured` and carries the host;
callers match it with `errors.As`. A missing `github.com` connection keeps
returning `ErrGitHubNotConfigured` unchanged. HTTP responses keep the existing
`github_not_configured` status (409) and code and add `host`, so existing
clients keep their "Connect GitHub" handling and new clients can name the
host.

## Control flow

### Credential resolution

`ResolveCredentialRequest` gains `Host`. The resolver normalizes it, loads the
connection for `(workspace, host)`, and fails with `HostNotConnectedError`
when none exists. Its cache key gains the host. Workspace cache epochs stay
per workspace, so replacing one host connection invalidates cached clients for
the workspace, and the next resolution rebuilds only what it needs.

Token clients, App installation clients, OAuth clients, and the manifest
conversion client take the host endpoint record. PAT and CLI clients get the
connection host; App clients get the registration host.

### Item host

Each GitHub operation takes its host from the item it serves:

| Item | Host source |
| --- | --- |
| Repository | `provider_host` |
| Task pull request, pull-request watch | `github_host` column |
| Review or issue watch, review or issue task | the watch's `github_host` |
| Pull request or issue URL | URL hostname |
| HTTP or WebSocket request with owner and repo | `host` parameter |
| Webhook delivery | the receiving App registration's `github_host` |
| Broker credential scope | `scope.Host` |

A GitHub repository row with an empty `provider_host` is a `github.com`
repository, because the repositories migration backfills it that way; a
malformed `provider_host` fails the operation.

The internal service resolves clients through one helper that takes a
`repoRef{host, owner, name}`. Exported service methods that name an owner and
repository take the host as an extra argument. Callers outside the GitHub
package pass the repository host.

### Host-scoped watches and sources

Review watches, issue watches, automation triggers, workflow sync, and Office
config sync store one host. Polls, searches, and repository reads resolve the
connection for that host. A watch whose host loses its connection records the
host-not-connected error as its last error and keeps its configuration.
Batched pull-request lookups run one batch per host. A repository-list
workspace scope matches each entry's host (empty means `github.com`);
organization scopes name owners only and apply to every host.

### Repository identity

- Remote URL parsing accepts `github.com` and any host with a connection in
  the workspace. The task service learns connected hosts through a reader
  injected by the backend composition root, as it learns the GitLab origin
  verifier.
- Local repository creation classifies an origin on a connected host as
  provider `github` with `provider_host = https://<host>`. SCP-style origins
  with any user name are parsed.
- Saving a host connection runs a reclassification in that workspace: rows
  with an empty provider whose remote URL host equals the connected host get
  provider `github`, provider host, owner, and name.
- Clone URLs and managed clone paths use `provider_host`. SSH clone URLs use
  the user `<subdomain>` on a `<subdomain>.ghe.com` host and `git` elsewhere.
  The review-watch repository resolver looks up and clones on the pull
  request's host.

### Links

Pull-request, issue, profile, mention, and file links are built on the item
host. The web host for a pull request or issue comes from its URL; the web host
for a repository comes from `provider_host`. Mention scopes become
`<host>/<owner>/<repo>`; `github.com` scopes keep their current form.

### Task credentials

- **Managed mode:** broker scopes already carry `Host`. The GitHub provider
  accepts any connected host and resolves the credential for that host. The
  managed `gh` shim keeps its repository detection, sets `GH_HOST` to the
  target host, and passes the credential as `GH_TOKEN` for `github.com` and
  `.ghe.com` hosts or as `GH_ENTERPRISE_TOKEN` for other hosts.
- **Executor mode:** the host bridge already covers attached GitHub hosts. The
  explicit-token check follows the same variable rule as the shim.
- **Remote executors with profile tokens:** Docker, Sprites, and SSH prepare
  scripts receive the list of attached GitHub hosts and add an HTTPS helper
  and SSH-to-HTTPS rewrite for each host, reading the variable that applies to
  that host.
- **Pull-request creation:** the executor publishes
  `KANDEV_GITHUB_HOSTS`, a comma-separated list of attached enterprise hosts.
  agentctl treats an origin on a listed host as GitHub and runs `gh pr create`
  in the checkout.

### GitHub App flows

Manifest start validates the requested host and builds the registration URL
on it. Manifest conversion posts to the host's REST base. Import validates
against the named host. The stored registration keeps the host, and the
runtime builds its App, OAuth, and installation clients from it. Installation
completion upserts the connection for the registration host. Webhook delivery
routes by registration, then filters affected connections by the registration
host.

### Frontend

- A workspace query exposes the connected hosts from the status `hosts` array.
- GitHub URL parsers take the connected hosts and accept them alongside
  `github.com`. The remote URL allowlist does the same. A URL on a GitHub-like
  path but an unknown host reports "host not connected".
- API helpers that name an owner and repository send `host`.
- Settings render one connection card per host and an "Connect another GitHub
  host" action. The PAT form gains a host field. The CLI picker lists accounts
  on every host and marks hosts that are already connected. App create and
  import forms gain a host field.
- Profile, mention, quick-task, remote-repository, and file links use the item
  host.

## Failure and recovery

- An invalid host or a failed validation leaves the previous connection for
  that host unchanged.
- An item on a host with no connection fails with the host-not-connected
  error. No request reaches any host, and no other connection is used.
- A GitHub Enterprise Server instance without rate limiting answers
  `/rate_limit` with 404; rate-limit seeding already tolerates fetch failures.
- The connection table rebuild runs in one transaction and is replayable on
  SQLite and PostgreSQL.

## Persistence

- Table rebuild of `github_workspace_connections`, `github_user_connections`,
  and `github_user_connection_versions` with the new keys and triggers,
  following the existing legacy-constraint rebuild helper.
- Additive `github_host` columns with a `github.com` default on the PR, watch,
  and sync tables.
- No change to `repositories`.

## Security

- The token for a host is sent only to that host's endpoints.
- A URL host never selects a credential by itself: it selects a host, and the
  credential is the workspace connection for that host.
- Repository classification and remote URL acceptance trust only hosts with a
  connection in the same workspace.
- Host validation prevents flag injection into `gh` and path injection into
  API and web URLs.
- Operator checks for CLI connections and App registrations are unchanged.

## Observability

No new metrics. Errors and connection health name the host. Existing PR
discovery health and rate-limit snapshots are keyed by credential scope, which
now includes the host.

## Related decisions

- [Task Git credential policy](../../../decisions/2026-07-27-task-git-credential-policy.md)
- [Workspace-selectable GitHub App registrations](../../../decisions/2026-07-21-workspace-selectable-github-app-registrations.md)
- [Provider-neutral Git credential broker](../../../decisions/2026-07-31-provider-neutral-git-credential-broker.md)
