package service

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/events"
	"github.com/kandev/kandev/internal/task/models"
)

// GitLabRepositoryOriginVerifier checks a locator origin against the owning
// workspace connection without trusting browser-supplied host metadata.
type GitLabRepositoryOriginVerifier interface {
	VerifyGitLabRepositoryOrigin(context.Context, string, string) error
}

// SetGitLabRepositoryOriginVerifier wires the workspace-owned origin boundary.
func (s *Service) SetGitLabRepositoryOriginVerifier(verifier GitLabRepositoryOriginVerifier) {
	s.gitLabRepositoryOriginVerifier = verifier
}

// WorkspaceGitHubHostsReader lists the GitHub hosts a workspace has a
// connection for.
type WorkspaceGitHubHostsReader interface {
	WorkspaceGitHubHosts(context.Context, string) ([]string, error)
}

// SetWorkspaceGitHubHostsReader wires the workspace GitHub connection list.
func (s *Service) SetWorkspaceGitHubHostsReader(reader WorkspaceGitHubHostsReader) {
	s.workspaceGitHubHostsReader = reader
}

// connectedGitHubEnterpriseHosts lists the workspace's connected GitHub
// hosts other than github.com. A read failure yields no host, so nothing on
// an enterprise host is trusted.
func (s *Service) connectedGitHubEnterpriseHosts(ctx context.Context, workspaceID string) []string {
	if s.workspaceGitHubHostsReader == nil {
		return nil
	}
	hosts, err := s.workspaceGitHubHostsReader.WorkspaceGitHubHosts(ctx, workspaceID)
	if err != nil {
		s.logger.Debug("list workspace GitHub hosts", zap.String("workspace_id", workspaceID), zap.Error(err))
		return nil
	}
	enterprise := make([]string, 0, len(hosts))
	for _, host := range hosts {
		if !strings.EqualFold(host, githubHostname) {
			enterprise = append(enterprise, strings.ToLower(host))
		}
	}
	return enterprise
}

// trustConnectedGitHubRemote marks a remote URL on a connected GitHub
// enterprise host as a trusted GitHub locator.
func (s *Service) trustConnectedGitHubRemote(ctx context.Context, workspaceID string, input *TaskRepositoryInput) {
	if input.RepositoryID != "" || input.TrustedProviderDescriptor || effectiveRemoteURL(*input) == "" {
		return
	}
	if provider := strings.TrimSpace(input.Provider); provider != "" && !strings.EqualFold(provider, providerGitHub) {
		return
	}
	parsed, _, err := normalizeRemoteRepositoryURL(effectiveRemoteURL(*input))
	if err != nil || !slices.Contains(s.connectedGitHubEnterpriseHosts(ctx, workspaceID), strings.ToLower(parsed.Hostname())) {
		return
	}
	input.Provider = providerGitHub
	input.TrustedRemote = true
}

func (s *Service) verifyRemoteRepositoryOrigin(ctx context.Context, workspaceID string, input *TaskRepositoryInput) error {
	if input.TrustedProviderDescriptor {
		return nil
	}
	provider, _, _, canonicalURL, err := parseRemoteRepositoryURL(effectiveRemoteURL(*input), input.Provider)
	if err != nil {
		return NewRepositorySelectionError(RepositorySelectionErrorInvalid, err)
	}
	origin := remoteProviderHost(provider, canonicalURL)
	if provider != providerGitLab || origin == "https://gitlab.com" {
		return nil
	}
	if s.gitLabRepositoryOriginVerifier == nil {
		return NewRepositorySelectionError(RepositorySelectionErrorInvalid, nil)
	}
	if err := s.gitLabRepositoryOriginVerifier.VerifyGitLabRepositoryOrigin(ctx, workspaceID, origin); err != nil {
		return normalizeRepositorySelectionError(err)
	}
	input.TrustedRemote = true
	return nil
}

// untrustedRemoteOrigin rejects a parsed remote on a self-managed host that no
// workspace connection vouched for.
func untrustedRemoteOrigin(provider, providerHost string, trusted bool) error {
	switch {
	case trusted:
		return nil
	case provider == providerGitLab && providerHost != "https://gitlab.com":
		return fmt.Errorf("untrusted GitLab origin %q", providerHost)
	case provider == providerGitHub && providerHost != githubProviderHost:
		return fmt.Errorf("untrusted GitHub origin %q", providerHost)
	}
	return nil
}

// ReclassifyWorkspaceGitHubRepositories records GitHub identity on the
// workspace's repositories that have no provider and whose origin is on host.
// It returns the number of repositories updated.
func (s *Service) ReclassifyWorkspaceGitHubRepositories(ctx context.Context, workspaceID, host string) (int, error) {
	host = strings.ToLower(strings.TrimSpace(host))
	repositories, err := s.repoEntities.ListRepositories(ctx, workspaceID)
	if err != nil {
		return 0, err
	}
	updated := 0
	var errs []error
	for _, repository := range repositories {
		if repository.Provider != "" || !reclassifyGitHubIdentity(repository, host) {
			continue
		}
		if err := s.repoEntities.UpdateRepository(ctx, repository); err != nil {
			errs = append(errs, fmt.Errorf("reclassify repository %s: %w", repository.ID, err))
			continue
		}
		updated++
		s.publishRepositoryEvent(ctx, events.RepositoryUpdated, repository)
	}
	return updated, errors.Join(errs...)
}

func reclassifyGitHubIdentity(repository *models.Repository, host string) bool {
	origin := repository.RemoteURL
	if repository.LocalPath != "" {
		if local, err := readGitRemoteOriginURL(repository.LocalPath); err == nil && local != "" {
			origin = local
		}
	}
	parsed, _, err := normalizeRemoteRepositoryURL(origin)
	if err != nil || strings.ToLower(parsed.Hostname()) != host {
		return false
	}
	_, owner, name, canonical, err := parseRemoteRepositoryURL(origin, providerGitHub)
	if err != nil {
		return false
	}
	repository.Provider = providerGitHub
	repository.ProviderHost = protocolHTTPS + "://" + host
	repository.ProviderOwner = owner
	repository.ProviderName = name
	repository.RemoteURL = canonical
	return true
}
