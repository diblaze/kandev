package github

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/kandev/kandev/internal/auth/authn"
)

const workspacePATSecretName = "GitHub workspace PAT"

// ErrWorkspaceConnectionStale prevents a validated request from replacing a
// newer workspace connection that committed while validation was in flight.
var ErrWorkspaceConnectionStale = errors.New("GitHub workspace connection changed")

// ErrGHCLIOperatorRequired prevents authenticated members from binding a
// deployment host gh CLI account to a workspace.
var ErrGHCLIOperatorRequired = errors.New("GitHub CLI operator access is required")

type SetWorkspaceConnectionRequest struct {
	Source ConnectionSource `json:"source"`
	Token  string           `json:"token,omitempty"`
	Host   string           `json:"host,omitempty"`
	Login  string           `json:"login,omitempty"`
}

// GetWorkspaceConnectionHealth exposes aggregate persisted connection state
// for system health without falling back to startup or ambient credentials.
func (s *Service) GetWorkspaceConnectionHealth(ctx context.Context) (WorkspaceConnectionHealth, error) {
	if s == nil || s.store == nil {
		return WorkspaceConnectionHealth{}, ErrGitHubNotConfigured
	}
	return s.store.GetWorkspaceConnectionHealth(ctx)
}

// WorkspaceConnectionFingerprint returns an opaque, non-secret string that
// changes whenever the workspace's automation connection is replaced,
// reconnected, or has its credential rotated/revoked (status or credential
// generation changes). Callers (e.g. workflow-sync's and the PR-watch
// poller's auth circuit breakers) use this to detect "the credential
// changed" and reset an open backoff circuit promptly instead of waiting out
// the schedule. Returns "" when no connection exists — an empty fingerprint
// must never be treated as "changed" by callers, since there was nothing to
// compare against.
func (s *Service) WorkspaceConnectionFingerprint(ctx context.Context, workspaceID string) (string, error) {
	if s == nil || s.store == nil {
		return "", nil
	}
	connections, err := s.store.ListWorkspaceConnections(ctx, workspaceID)
	if err != nil {
		return "", err
	}
	parts := make([]string, 0, len(connections))
	for _, connection := range connections {
		parts = append(parts, fmt.Sprintf("%s:%s:%d", connection.GitHubHost, connection.Status, connection.CredentialGeneration))
	}
	return strings.Join(parts, ","), nil
}

// WorkspaceGitHubHosts returns the GitHub hosts the workspace has a
// connection for, ordered by host.
func (s *Service) WorkspaceGitHubHosts(ctx context.Context, workspaceID string) ([]string, error) {
	if s == nil || s.store == nil {
		return nil, ErrGitHubNotConfigured
	}
	connections, err := s.store.ListWorkspaceConnections(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	hosts := make([]string, 0, len(connections))
	for _, connection := range connections {
		hosts = append(hosts, connection.GitHubHost)
	}
	return hosts, nil
}

func (s *Service) SetWorkspaceConnection(
	ctx context.Context,
	workspaceID string,
	req SetWorkspaceConnectionRequest,
) (*WorkspaceConnection, error) {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return nil, ErrGitHubWorkspaceRequired
	}
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return nil, err
	}
	host, err := normalizeGitHubHost(req.Host)
	if err != nil {
		return nil, err
	}
	if s.store == nil || s.connectionSecrets == nil {
		return nil, ErrGitHubNotConfigured
	}
	observed, err := s.store.GetWorkspaceConnection(ctx, workspaceID, host)
	if err != nil {
		return nil, fmt.Errorf("load workspace connection: %w", err)
	}
	connection, err := s.validateAndCommitWorkspaceConnection(
		ctx, workspaceID, host, req, workspaceConnectionGeneration(observed),
	)
	if err == nil {
		s.notifyHostConnected(ctx, workspaceID, host)
	}
	return connection, err
}

// SetHostConnectedObserver registers a callback run after a host connection
// of a workspace is committed, outside the connection mutation lock.
func (s *Service) SetHostConnectedObserver(observer func(ctx context.Context, workspaceID, host string)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hostConnectedObserver = observer
}

func (s *Service) notifyHostConnected(ctx context.Context, workspaceID, host string) {
	s.mu.Lock()
	observer := s.hostConnectedObserver
	s.mu.Unlock()
	if observer != nil {
		observer(ctx, workspaceID, host)
	}
}

func (s *Service) validateAndCommitWorkspaceConnection(
	ctx context.Context,
	workspaceID, host string,
	req SetWorkspaceConnectionRequest,
	expectedGeneration int64,
) (*WorkspaceConnection, error) {
	switch req.Source {
	case ConnectionSourcePAT:
		token := strings.TrimSpace(req.Token)
		if token == "" {
			return nil, fmt.Errorf("GitHub token is required")
		}
		login, err := s.validateWorkspaceToken(ctx, host, token)
		if err != nil {
			return nil, err
		}
		return s.commitWorkspacePAT(ctx, workspaceID, host, login, token, expectedGeneration)
	case ConnectionSourceGHCLI:
		if err := requireGHCLIOperator(ctx); err != nil {
			return nil, err
		}
		return s.validateAndCommitWorkspaceGHCLI(
			ctx, workspaceID, host, req.Login, expectedGeneration,
		)
	case ConnectionSourceLegacyShared:
		return nil, fmt.Errorf("legacy shared GitHub auth is migration-only")
	case ConnectionSourceGitHubAppInstallation:
		return nil, fmt.Errorf("GitHub App installation must use the verified installation flow")
	default:
		return nil, fmt.Errorf("unsupported GitHub connection source %q", req.Source)
	}
}

func requireGHCLIOperator(ctx context.Context) error {
	identity, ok := authn.IdentityFromContext(ctx)
	if !ok {
		// Identity-free callers are internal lifecycle operations. HTTP requests
		// always carry either a real identity or the auth-disabled synthetic admin.
		return nil
	}
	if identity.IsAdmin() {
		return nil
	}
	return ErrGHCLIOperatorRequired
}

func (s *Service) newWorkspaceConnection(
	workspaceID, host string,
	source ConnectionSource,
	existing *WorkspaceConnection,
) *WorkspaceConnection {
	now := time.Now().UTC()
	connection := &WorkspaceConnection{
		WorkspaceID: workspaceID, Source: source, GitHubHost: host,
		Status: ConnectionStatusActive, CredentialGeneration: nextCredentialGeneration(existing),
		CreatedAt: now, UpdatedAt: now,
	}
	if existing != nil && !existing.CreatedAt.IsZero() {
		connection.CreatedAt = existing.CreatedAt
	}
	return connection
}

func nextCredentialGeneration(existing *WorkspaceConnection) int64 {
	if existing == nil || existing.CredentialGeneration < 1 {
		return 1
	}
	return existing.CredentialGeneration + 1
}

func workspaceConnectionGeneration(connection *WorkspaceConnection) int64 {
	if connection == nil {
		return 0
	}
	return connection.CredentialGeneration
}

func (s *Service) commitWorkspacePAT(
	ctx context.Context,
	workspaceID, host, login, token string,
	expectedGeneration int64,
) (*WorkspaceConnection, error) {
	lock := s.workspaceConnectionMutationLock(workspaceID, host)
	lock.Lock()
	defer lock.Unlock()
	existing, err := s.store.GetWorkspaceConnection(ctx, workspaceID, host)
	if err != nil {
		return nil, fmt.Errorf("load workspace connection: %w", err)
	}
	if workspaceConnectionGeneration(existing) != expectedGeneration {
		return nil, ErrWorkspaceConnectionStale
	}
	connection := s.newWorkspaceConnection(workspaceID, host, ConnectionSourcePAT, existing)
	connection.Login = login
	secretKey := WorkspacePATSecretKey(connection.WorkspaceID, host)
	previous, hadPrevious, err := revealOptionalSecret(ctx, s.connectionSecrets, secretKey)
	if err != nil {
		return nil, fmt.Errorf("load previous workspace PAT: %w", err)
	}
	if err := s.connectionSecrets.Set(ctx, secretKey, workspacePATSecretName, token); err != nil {
		return nil, fmt.Errorf("store workspace PAT: %w", err)
	}
	if err := s.applyAutomationTransition(ctx, existing, connection, func() error {
		return s.store.UpsertWorkspaceConnection(ctx, connection)
	}); err != nil {
		s.compensatePATWrite(ctx, secretKey, previous, hadPrevious)
		return nil, fmt.Errorf("replace workspace GitHub connection: %w", err)
	}
	if existing != nil && existing.InstallationID != nil {
		s.InvalidateAppInstallationCredentials(existing.AppRegistrationID, *existing.InstallationID)
	}
	s.invalidateWorkspaceCredential(connection.WorkspaceID)
	return connection, nil
}

func (s *Service) validateAndCommitWorkspaceGHCLI(
	ctx context.Context,
	workspaceID, host, login string,
	expectedGeneration int64,
) (*WorkspaceConnection, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return nil, fmt.Errorf("GitHub login is required")
	}
	tokenResolver := ResolveGHAccountToken
	if s.resolver != nil && s.resolver.ghToken != nil {
		tokenResolver = s.resolver.ghToken
	}
	token, err := tokenResolver(ctx, host, login)
	if err != nil {
		return nil, err
	}
	validatedLogin, err := s.validateWorkspaceToken(ctx, host, token)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(login, validatedLogin) {
		return nil, fmt.Errorf("selected gh login %q resolved as %q", login, validatedLogin)
	}
	return s.commitWorkspaceGHCLI(ctx, workspaceID, host, validatedLogin, expectedGeneration)
}

func (s *Service) commitWorkspaceGHCLI(
	ctx context.Context,
	workspaceID, host, login string,
	expectedGeneration int64,
) (*WorkspaceConnection, error) {
	lock := s.workspaceConnectionMutationLock(workspaceID, host)
	lock.Lock()
	defer lock.Unlock()
	existing, err := s.store.GetWorkspaceConnection(ctx, workspaceID, host)
	if err != nil {
		return nil, fmt.Errorf("load workspace connection: %w", err)
	}
	if workspaceConnectionGeneration(existing) != expectedGeneration {
		return nil, ErrWorkspaceConnectionStale
	}
	connection := s.newWorkspaceConnection(workspaceID, host, ConnectionSourceGHCLI, existing)
	connection.Login = login
	if err := s.applyAutomationTransition(ctx, existing, connection, func() error {
		return s.store.UpsertWorkspaceConnection(ctx, connection)
	}); err != nil {
		return nil, fmt.Errorf("replace workspace GitHub connection: %w", err)
	}
	if existing != nil && existing.Source == ConnectionSourcePAT {
		if err := deleteOptionalSecret(ctx, s.connectionSecrets, WorkspacePATSecretKey(workspaceID, host)); err != nil {
			return nil, errors.Join(err, restoreWorkspaceConnection(ctx, s.store, existing, workspaceID, host))
		}
	}
	if existing != nil && existing.InstallationID != nil {
		s.InvalidateAppInstallationCredentials(existing.AppRegistrationID, *existing.InstallationID)
	}
	s.invalidateWorkspaceCredential(connection.WorkspaceID)
	return connection, nil
}

func (s *Service) revokePersonalForAutomationTransition(
	ctx context.Context,
	existing, replacement *WorkspaceConnection,
) error {
	if existing == nil || existing.Source != ConnectionSourceGitHubAppInstallation {
		return nil
	}
	if replacement != nil && replacement.Source == ConnectionSourceGitHubAppInstallation &&
		existing.AppRegistrationID == replacement.AppRegistrationID &&
		equalInstallationID(existing.InstallationID, replacement.InstallationID) {
		return nil
	}
	if s.personalConnections == nil {
		return errors.New("personal GitHub connection repository is not configured")
	}
	return s.personalConnections.RevokeWorkspacePersonalConnections(ctx, existing.WorkspaceID, existing.GitHubHost)
}

func (s *Service) applyAutomationTransition(
	ctx context.Context,
	existing, replacement *WorkspaceConnection,
	mutation func() error,
) error {
	if existing == nil || existing.Source != ConnectionSourceGitHubAppInstallation ||
		(replacement != nil && replacement.Source == ConnectionSourceGitHubAppInstallation &&
			existing.AppRegistrationID == replacement.AppRegistrationID &&
			equalInstallationID(existing.InstallationID, replacement.InstallationID)) {
		return mutation()
	}
	if s.personalConnections == nil {
		return errors.New("personal GitHub connection repository is not configured")
	}
	return s.personalConnections.TransitionWorkspacePersonalConnections(
		ctx, existing.WorkspaceID, existing.GitHubHost, mutation,
	)
}

func equalInstallationID(left, right *int64) bool {
	return left != nil && right != nil && *left == *right
}

func (s *Service) validateWorkspaceToken(ctx context.Context, host, token string) (string, error) {
	factory := s.tokenClientFactory
	if factory == nil {
		factory = newHostPATClient
	}
	login, err := factory(host, token).GetAuthenticatedUser(ctx)
	if err != nil {
		return "", fmt.Errorf("%w: GitHub host %q: %w", ErrInvalidToken, host, err)
	}
	login = strings.TrimSpace(login)
	if login == "" {
		return "", fmt.Errorf("%w: GitHub host %q returned an empty login", ErrInvalidToken, host)
	}
	return login, nil
}

func newHostPATClient(host, token string) Client {
	return NewPATClient(token).WithHost(host)
}

func revealOptionalSecret(ctx context.Context, secrets ConnectionSecretStore, key string) (string, bool, error) {
	exists, err := secrets.Exists(ctx, key)
	if err != nil || !exists {
		return "", false, err
	}
	value, err := secrets.Reveal(ctx, key)
	return value, true, err
}

func (s *Service) compensatePATWrite(ctx context.Context, key, previous string, hadPrevious bool) {
	if hadPrevious {
		_ = s.connectionSecrets.Set(ctx, key, workspacePATSecretName, previous)
		return
	}
	_ = s.connectionSecrets.Delete(ctx, key)
}

// DeleteWorkspaceConnection removes the connection of one host; connections
// on other hosts are untouched.
func (s *Service) DeleteWorkspaceConnection(ctx context.Context, workspaceID, host string) error {
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return ErrGitHubWorkspaceRequired
	}
	host, err := normalizeGitHubHost(host)
	if err != nil {
		return err
	}
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return err
	}
	if s.store == nil {
		return ErrGitHubNotConfigured
	}
	lock := s.workspaceConnectionMutationLock(workspaceID, host)
	lock.Lock()
	defer lock.Unlock()
	connection, err := s.store.GetWorkspaceConnection(ctx, workspaceID, host)
	if err != nil {
		return err
	}
	if connection != nil && connection.Source == ConnectionSourcePAT && s.connectionSecrets != nil {
		err = s.deleteWorkspacePATConnection(ctx, workspaceID, host)
	} else {
		err = s.deleteWorkspaceConnectionMetadata(ctx, workspaceID, host, connection)
	}
	if err != nil {
		return err
	}
	if connection != nil && connection.InstallationID != nil {
		s.InvalidateAppInstallationCredentials(connection.AppRegistrationID, *connection.InstallationID)
	}
	s.invalidateWorkspaceCredential(workspaceID)
	return nil
}

func (s *Service) deleteWorkspacePATConnection(ctx context.Context, workspaceID, host string) error {
	secretKey := WorkspacePATSecretKey(workspaceID, host)
	previous, hadPrevious, err := revealOptionalSecret(ctx, s.connectionSecrets, secretKey)
	if err != nil {
		return fmt.Errorf("load workspace PAT: %w", err)
	}
	if err := deleteOptionalSecret(ctx, s.connectionSecrets, secretKey); err != nil {
		return fmt.Errorf("delete workspace PAT: %w", err)
	}
	if err := s.store.DeleteWorkspaceConnection(ctx, workspaceID, host); err != nil {
		return errors.Join(err, restoreOptionalSecret(
			ctx, s.connectionSecrets, secretKey, workspacePATSecretName, previous, hadPrevious,
		))
	}
	return nil
}

func (s *Service) deleteWorkspaceConnectionMetadata(
	ctx context.Context, workspaceID, host string, connection *WorkspaceConnection,
) error {
	return s.applyAutomationTransition(ctx, connection, nil, func() error {
		return s.store.DeleteWorkspaceConnection(ctx, workspaceID, host)
	})
}

func (s *Service) workspaceConnectionMutationLock(workspaceID, host string) *sync.Mutex {
	key := workspaceID + "\x00" + host
	var hash uint32
	for index := range len(key) {
		hash = hash*33 + uint32(key[index])
	}
	return &s.connectionMutationLocks[hash%uint32(len(s.connectionMutationLocks))]
}

func deleteOptionalSecret(ctx context.Context, secrets ConnectionSecretStore, key string) error {
	if secrets == nil {
		return nil
	}
	exists, err := secrets.Exists(ctx, key)
	if err != nil || !exists {
		return err
	}
	return secrets.Delete(ctx, key)
}

func (s *Service) invalidateWorkspaceCredential(workspaceID string) {
	if s.resolver != nil {
		s.resolver.InvalidateWorkspace(workspaceID)
	}
	if s.prDiscoveryHealth != nil {
		s.prDiscoveryHealth.clearWorkspace(workspaceID)
		s.invalidateAllPRDiscoveryAttempts(workspaceID)
	}
	// Existing caches are still global during the compatibility phase. Clear
	// them on replacement until Task 05 prefixes every key by principal.
	s.clearAuthCaches()
}

func (s *Service) clearAuthCaches() {
	for _, cache := range []*ttlCache{
		s.searchCache, s.prStatusCache, s.prFeedbackCache, s.mergeMethodsCache,
		s.accessibleReposCache, s.repoErrorCache, s.workflowRunsCache, s.workflowJobsCache,
	} {
		if cache != nil {
			cache.clear()
		}
	}
	if s.protectionCache != nil {
		s.protectionCache.clear()
	}
}

// CopyWorkspaceConnectionToWorkspace duplicates every host connection of the
// source workspace onto the destination workspace, including the PAT secret
// for `pat` sources. No-op when the source workspace has no connection or the
// two workspaces are the same. Used by the improve-kandev bootstrap when it
// creates the dedicated workspace so improve tasks inherit the user's GitHub
// access without re-authenticating.
func (s *Service) CopyWorkspaceConnectionToWorkspace(ctx context.Context, srcWorkspaceID, dstWorkspaceID string) error {
	if srcWorkspaceID == "" || dstWorkspaceID == "" || srcWorkspaceID == dstWorkspaceID {
		return nil
	}
	sources, err := s.store.ListWorkspaceConnections(ctx, srcWorkspaceID)
	if err != nil {
		return fmt.Errorf("load source workspace connections: %w", err)
	}
	for _, src := range sources {
		if err := s.copyWorkspaceConnection(ctx, src, dstWorkspaceID); err != nil {
			return err
		}
	}
	if len(sources) > 0 {
		s.invalidateWorkspaceCredential(dstWorkspaceID)
	}
	return nil
}

func (s *Service) copyWorkspaceConnection(ctx context.Context, src *WorkspaceConnection, dstWorkspaceID string) error {
	now := time.Now().UTC()
	connection := &WorkspaceConnection{
		WorkspaceID:              dstWorkspaceID,
		Source:                   src.Source,
		GitHubHost:               src.GitHubHost,
		Login:                    src.Login,
		InstallationID:           src.InstallationID,
		InstallationAccountLogin: src.InstallationAccountLogin,
		InstallationAccountType:  src.InstallationAccountType,
		AppRegistrationID:        src.AppRegistrationID,
		Status:                   src.Status,
		CredentialGeneration:     nextCredentialGeneration(nil),
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if err := s.store.UpsertWorkspaceConnection(ctx, connection); err != nil {
		return fmt.Errorf("store destination workspace connection: %w", err)
	}
	if src.Source != ConnectionSourcePAT {
		return nil
	}
	token, err := s.connectionSecrets.Reveal(ctx, WorkspacePATSecretKey(src.WorkspaceID, src.GitHubHost))
	if err != nil {
		return fmt.Errorf("reveal source workspace PAT: %w", err)
	}
	if err := s.connectionSecrets.Set(
		ctx, WorkspacePATSecretKey(dstWorkspaceID, src.GitHubHost), workspacePATSecretName, token,
	); err != nil {
		return fmt.Errorf("store destination workspace PAT: %w", err)
	}
	return nil
}
