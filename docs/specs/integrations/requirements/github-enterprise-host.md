---
status: draft
system: integrations
created: 2026-10-22
owners:
  - kandev
---

# GitHub Enterprise Host Requirements

## Overview

Teams keep code on `github.com`, on GitHub Enterprise Cloud with data
residency (a `<subdomain>.ghe.com` tenant), and on GitHub Enterprise Server (a
company hostname), often at the same time. Without this capability a
workspace reaches only `github.com`. It lets one workspace connect `github.com`
and any number of enterprise hosts, and makes every GitHub feature work on each
of them. Each repository, pull request, issue, and task credential routes to
the connection for its own host. `github.com` behavior is unchanged.

The integration system owns this contract because it owns workspace GitHub
connections, provider identity, and GitHub synchronization.

## Terminology

- **GitHub host:** The lowercase hostname of a GitHub deployment, such as
  `github.com`, `octo.ghe.com`, or `ghe.example.com`.
- **Enterprise host:** A GitHub host other than `github.com`.
- **Host connection:** The automation connection a workspace holds for one
  GitHub host. A workspace holds at most one per host.
- **Item host:** The GitHub host of a repository, pull request, issue, or
  credential request, taken from its stored provider host or URL.

## Requirements

### REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001: Host connections

**Intent:** An operator connects a workspace to several GitHub hosts with any
connection source that `github.com` supports.

**User story:** As an operator, I want to connect my workspace to
`github.com` and to my company GitHub host, so that Kandev works with both
sets of repositories.

#### Acceptance criteria

- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.1:** When an operator saves a
  GitHub CLI, personal access token, or GitHub App installation connection for
  a host, the system shall validate the credential against that host's API and
  store the connection for that host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.2:** When a workspace already
  has a connection for one host and an operator saves a connection for another
  host, the system shall keep both connections.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.3:** When an operator replaces
  or removes the connection for one host, the system shall leave the
  connections for other hosts and their cached credentials unchanged.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.4:** When the workspace GitHub
  status is read, the system shall list every host connection with its source,
  login or installation account, and health.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.5:** When a connection request
  names a host that is not a bare hostname (it contains a scheme, path, port,
  user information, or whitespace, or starts with a hyphen), the system shall
  reject the request before contacting any host or running the GitHub CLI.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.6:** When credential validation
  fails for a host, the system shall keep that host's previous connection and
  return an error that names the host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.7:** When an App-backed host
  connection exists, a user can connect a personal identity for that host, and
  the personal identity shall apply only to items on that host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.8:** When an upgraded
  installation starts, every existing workspace connection shall keep working
  as its `github.com` host connection.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-001.9:** When an operator creates a
  workspace and the backend host GitHub CLI has an active account on several
  hosts, the new workspace shall receive a GitHub CLI connection for each of
  those hosts.

### REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002: Host routing

**Intent:** Every GitHub operation uses the connection for the item host, so
the same owner and repository name on two hosts never share a credential.

#### Acceptance criteria

- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.1:** When the system reads or
  changes a repository, pull request, issue, check, review, branch, or file,
  it shall use the connection for the item host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.2:** When the item host has no
  connection in the workspace, the system shall fail the operation without
  contacting any GitHub host and report an error that names the host as not
  connected.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.3:** When a request names a
  repository without a host, the system shall use `github.com`.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.4:** When a review watch, issue
  watch, automation trigger, workflow sync source, or Office config sync
  source is saved, it shall store one host, and every poll or read for it
  shall use that host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.5:** When a user searches pull
  requests, issues, organizations, or accessible repositories, the user can
  choose the host, and the results shall come from that host only.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-002.6:** When an App webhook
  arrives, the system shall apply it only to connections on the host of the
  App registration that received it.

### REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003: Repository identity

**Intent:** Repositories on a connected enterprise host are GitHub
repositories with the right provider host, however they enter Kandev.

#### Acceptance criteria

- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.1:** When a user adds a local
  repository whose origin host is a connected host, the system shall record
  provider `github`, provider host `https://<host>`, and the owner and name.
  This includes SSH origins with any user name, such as
  `org@octo.ghe.com:org/repo.git`.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.2:** When a user adds a
  repository by a remote URL on a connected host, the system shall accept it
  as a GitHub repository on that host and clone it with that host's
  credentials.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.3:** When a remote URL or local
  origin is on a host that is neither a connected host nor a host Kandev
  already recognizes, the system shall not record a GitHub provider for it.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.4:** When a host connection is
  saved, the system shall classify the workspace's existing repositories that
  have no provider and whose origin is on that host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.5:** When a review watch
  creates a task for a pull request in a repository Kandev has not cloned, the
  system shall clone it from the pull request's host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-003.6:** When a repository is on
  `github.com`, its recorded identity shall be the same as before this
  capability.

### REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004: Pull requests, issues, and links

**Intent:** Pull-request, issue, and link features work on every connected
host, and every link points at the item host.

#### Acceptance criteria

- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.1:** When a task branch in a
  repository on a connected host has an open pull request, the system shall
  discover it, keep its status current, and expose its URL on that host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.2:** When a user or agent links
  a task to a pull-request or issue URL on a connected host, the system shall
  save the link, and later listings shall include it.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.3:** When a user or agent links
  a pull-request or issue URL on a host with no connection, the system shall
  reject it with the host-not-connected error.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.4:** When a user starts a task
  from a pull request on a connected host, including a pull request from a
  fork, the system shall check out its head and compare against its base on
  that host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.5:** When pull-request
  automation sends a prompt, the prompt shall contain the pull-request URL on
  its host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.6:** When the UI shows a
  pull-request author, reviewer, commenter, mention, or a link to a file in a
  repository, the link shall point at the item host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-004.7:** When an item is on
  `github.com`, its URLs, prompts, and links shall be the same as before this
  capability.

### REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005: Task Git and CLI credentials

**Intent:** Agents can fetch, push, and use `gh` against the host of each
attached repository in both task credential modes.

#### Acceptance criteria

- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.1:** When a task in managed
  credential mode attaches a repository on a connected host, Git over HTTPS
  shall receive a short-lived credential from that host's connection.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.2:** When an agent in a managed
  task runs `gh` for a repository on a connected host, the command shall run
  against that host with that host's credential.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.3:** When a managed-mode
  credential request names a host with no connection, the system shall deny
  it.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.4:** When a Local or Worktree
  task inherits executor credentials, the host GitHub CLI bridge shall cover
  every attached GitHub host, and an explicit `GH_TOKEN` or `GITHUB_TOKEN`
  shall take precedence for `github.com` and `ghe.com` tenants while
  `GH_ENTERPRISE_TOKEN` or `GITHUB_ENTERPRISE_TOKEN` takes precedence for
  GitHub Enterprise Server hosts.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.5:** When a Docker, SSH, or
  cloud task receives profile GitHub tokens, the system shall configure HTTPS
  credentials for each attached GitHub host with the token variable that the
  GitHub CLI uses for that host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-005.6:** When a user creates a pull
  request from a task whose origin is on a connected enterprise host, the
  system shall create it on that host instead of reporting an unsupported
  remote.

### REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006: GitHub Apps on enterprise hosts

**Intent:** GitHub App automation works on enterprise hosts with the same
flows as on `github.com`.

#### Acceptance criteria

- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.1:** When an operator creates a
  GitHub App from a manifest for a host, the registration page, manifest
  conversion, and stored registration shall use that host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.2:** When an operator imports
  an existing App and names its host, the system shall verify the App against
  that host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.3:** When an operator installs
  an App registration, the installation page shall open on the registration's
  host, and the completed installation shall become the connection for that
  host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.4:** When a user connects a
  personal identity through an enterprise App, OAuth authorization and token
  exchange shall use the registration's host.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-006.5:** When an operator starts App
  creation or import for any host, the existing requirement for a stable
  public HTTPS Kandev origin shall apply unchanged.

### REQ-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007: Host-aware settings and URL input

**Intent:** Users manage host connections in settings, and every GitHub URL
input accepts URLs on connected hosts.

#### Acceptance criteria

- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.1:** When a user opens
  workspace GitHub settings, the page shall show one card per host connection
  and an action to connect another host, on desktop and phone layouts.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.2:** When a user adds a
  personal access token connection, the form shall ask for the host and
  default it to `github.com`.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.3:** When a user pastes a
  repository, pull-request, or issue URL on a connected host into a task,
  link, watch, workspace source, or workflow sync input, the UI shall
  recognize it the same way it recognizes a `github.com` URL.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.4:** When a user pastes such a
  URL on a host with no connection, the UI shall show that the host is not
  connected and offer the GitHub settings page.
- **AC-INTEGRATIONS-GITHUB-ENTERPRISE-HOST-007.5:** When the workspace has more
  than one host connection, the GitHub dashboard and search shall offer a host
  selector, and results shall show their host.

## Out of scope

- GitHub hosts on a non-default port.
- The migration-only `legacy_shared` source, which stays the deployment's
  `github.com` credential.
- Kandev release, update, installer, and plugin downloads; gist sharing;
  Kandev's own repository links; documentation links; and User-Agent strings.
  These name Kandev's own `github.com` resources, not a user's GitHub host.
