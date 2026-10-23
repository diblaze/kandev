package github

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
)

var errWorkspaceDefaultsUnavailable = errors.New("workspace defaults unavailable")

const (
	workspaceDefaultsCLIValidationTimeout = 5 * time.Second
	freshWorkspaceDefaultsTimeout         = 30 * time.Second
)

// InitializeWorkspaceDefaults persists the task Git policy for a newly
// created workspace and, when the caller is allowed to use the deployment
// host identity, snapshots the active gh CLI account of every host that has
// no connection yet as a named connection.
// Account discovery and validation are deliberately best-effort: a fresh
// install without an authenticated gh CLI still gets executor credential
// inheritance and can be configured later from workspace settings.
func (s *Service) InitializeWorkspaceDefaults(ctx context.Context, workspaceID string) error {
	if s == nil || s.store == nil {
		return errWorkspaceDefaultsUnavailable
	}
	workspaceID = strings.TrimSpace(workspaceID)
	if workspaceID == "" {
		return fmt.Errorf("workspace_id is required")
	}
	if err := s.authorizeWorkspaceAccess(ctx, workspaceID); err != nil {
		return err
	}
	if err := s.store.EnsureWorkspaceExecutorDefaults(ctx, workspaceID); err != nil {
		return fmt.Errorf("persist workspace GitHub defaults: %w", err)
	}
	connections, err := s.store.ListWorkspaceConnections(ctx, workspaceID)
	if err != nil {
		return fmt.Errorf("load existing workspace GitHub connections: %w", err)
	}

	// An authenticated member may create a workspace, but must not receive the
	// operator's host gh identity. Keep the authorization boundary before any
	// account listing so member requests cannot observe operator accounts.
	if err := requireGHCLIOperator(ctx); err != nil {
		if errors.Is(err, ErrGHCLIOperatorRequired) {
			return nil
		}
		return err
	}

	accounts, err := s.ListGHAccounts(ctx)
	if err != nil {
		s.logWorkspaceDefaultsFallback("unable to discover host gh account", workspaceID, err)
		return nil
	}
	connected := make(map[string]bool, len(connections))
	for _, connection := range connections {
		connected[connection.GitHubHost] = true
	}
	for _, account := range activeGHAccountsByHost(accounts) {
		if connected[account.Host] {
			continue
		}
		if s.connectionSecrets == nil {
			s.logWorkspaceDefaultsFallback("connection secret store is unavailable", workspaceID, errWorkspaceDefaultsUnavailable)
			return nil
		}
		s.bindWorkspaceDefaultCLIAccount(ctx, workspaceID, account)
	}
	return nil
}

func (s *Service) bindWorkspaceDefaultCLIAccount(ctx context.Context, workspaceID string, account GHAccount) {
	bindCtx, cancel := context.WithTimeout(ctx, workspaceDefaultsCLIValidationTimeout)
	defer cancel()
	if _, err := s.SetWorkspaceConnection(bindCtx, workspaceID, SetWorkspaceConnectionRequest{
		Source: ConnectionSourceGHCLI,
		Host:   account.Host,
		Login:  account.Login,
	}); err != nil {
		s.logWorkspaceDefaultsFallback("unable to bind host gh account", workspaceID, err)
	}
}

// InitializeFreshWorkspaceDefaults handles the repository-seeded initial
// workspace. The store records whether its GitHub schema was absent before
// startup; existing installations are intentionally left untouched.
func (s *Service) InitializeFreshWorkspaceDefaults(ctx context.Context) error {
	if s == nil || s.store == nil || !s.store.freshInstall {
		return nil
	}
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, freshWorkspaceDefaultsTimeout)
		defer cancel()
	}
	s.freshDefaultsMu.Lock()
	defer s.freshDefaultsMu.Unlock()
	if s.freshDefaultsDone {
		return nil
	}

	workspaceIDs, err := s.store.listWorkspaceIDs(ctx)
	if err != nil {
		return fmt.Errorf("list initial workspaces: %w", err)
	}
	for _, workspaceID := range workspaceIDs {
		if err := s.InitializeWorkspaceDefaults(ctx, workspaceID); err != nil {
			return err
		}
	}

	s.freshDefaultsDone = true
	return nil
}

// activeGHAccountsByHost returns the valid active gh account of each host, in
// the order gh lists the hosts.
func activeGHAccountsByHost(accounts []GHAccount) []GHAccount {
	seen := make(map[string]bool, len(accounts))
	active := make([]GHAccount, 0, len(accounts))
	for _, account := range accounts {
		if !validActiveGHAccount(account) {
			continue
		}
		host, err := normalizeGitHubHost(account.Host)
		if err != nil || seen[host] {
			continue
		}
		seen[host] = true
		account.Host = host
		account.Login = strings.TrimSpace(account.Login)
		active = append(active, account)
	}
	return active
}

func validActiveGHAccount(account GHAccount) bool {
	if !account.Active || strings.TrimSpace(account.Host) == "" || strings.TrimSpace(account.Login) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(account.State)) {
	case "active", checkStatusSuccess:
		return true
	default:
		return false
	}
}

func (s *Service) logWorkspaceDefaultsFallback(message, workspaceID string, err error) {
	if s == nil || s.logger == nil {
		return
	}
	s.logger.Warn(message, zap.String("workspace_id", workspaceID), zap.Error(err))
}
