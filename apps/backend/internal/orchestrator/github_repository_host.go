package orchestrator

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/github"
	"github.com/kandev/kandev/internal/task/models"
)

const githubProviderName = "github"

// repositoryGitHubHost returns the GitHub host of a task repository. A
// repository of another provider, or no repository row, yields an empty host,
// which means github.com; a malformed GitHub provider_host is an error.
func (s *Service) repositoryGitHubHost(ctx context.Context, repositoryID string) (string, error) {
	return githubHostOfRepository(s.getPushRepository(ctx, repositoryID))
}

func githubHostOfRepository(repository *models.Repository) (string, error) {
	if repository == nil || !strings.EqualFold(strings.TrimSpace(repository.Provider), githubProviderName) {
		return "", nil
	}
	return github.RepositoryGitHubHost(repository.ProviderHost)
}

// watchTargetHost is repositoryGitHubHost for background watch paths, which
// skip a repository whose host cannot be determined.
func (s *Service) watchTargetHost(ctx context.Context, repositoryID string) (string, bool) {
	host, err := s.repositoryGitHubHost(ctx, repositoryID)
	if err != nil {
		s.logger.Warn("skip PR watch for repository with an invalid GitHub host",
			zap.String("repository_id", repositoryID), zap.Error(err))
		return "", false
	}
	return host, true
}

// applyPushRepositoryHost sets the push identity host from the repository and
// reports false when the host is malformed, so push detection stops.
func (s *Service) applyPushRepositoryHost(identity *pushRepositoryIdentity, repository *models.Repository) bool {
	host, err := githubHostOfRepository(repository)
	if err != nil {
		s.logger.Warn("skip push detection for repository with an invalid GitHub host",
			zap.String("repository_id", identity.repositoryID), zap.Error(err))
		return false
	}
	identity.host = host
	return true
}
