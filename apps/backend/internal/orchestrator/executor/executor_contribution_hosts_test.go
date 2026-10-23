package executor

import (
	"context"
	"strings"
	"testing"

	"github.com/kandev/kandev/internal/task/models"
)

const contributionHostsBrokerURL = "https://kandev.example/api/github/credentials/resolve"

func contributionHostsRequest() *LaunchAgentRequest {
	return &LaunchAgentRequest{
		TaskID: "task-1", WorkspaceID: "workspace-1", SessionID: "session-1",
		ExecutorType: string(models.ExecutorTypeRemoteDocker), Env: map[string]string{},
	}
}

func contributionHostsRepository(host string) *models.Repository {
	return &models.Repository{
		Provider: gitHubProviderID, ProviderHost: "https://" + host, ProviderRepoID: "100",
		ProviderOwner: "acme", ProviderName: "widget", RemoteURL: "https://" + host + "/acme/widget.git",
	}
}

func contributionHostsDestination(host string) *models.ContributionDestination {
	return &models.ContributionDestination{
		Version:  models.ContributionDestinationVersion,
		Provider: models.ContributionDestinationProviderGitHub,
		SourceRepository: models.ContributionDestinationRepository{
			Host: host, Path: "acme/widget", ProviderID: "100", RemoteURL: "https://" + host + "/acme/widget.git",
		},
		TargetRepository: models.ContributionDestinationRepository{
			Host: host, Path: "agent/widget", ProviderID: "200", RemoteURL: "https://" + host + "/agent/widget.git",
		},
		CredentialBinding: &models.ContributionDestinationCredentialBinding{
			Source: "pat", Login: "automation", CredentialGeneration: 7,
		},
	}
}

func contributionHostsRemoteContribution(host string) *models.RemoteContribution {
	return &models.RemoteContribution{
		Version:              models.RemoteContributionVersion,
		Provider:             models.RemoteContributionProviderGitHub,
		Kind:                 models.RemoteContributionKindPullRequest,
		CanonicalURL:         "https://" + host + "/acme/widget/pull/7",
		Number:               7,
		State:                models.RemoteContributionStateOpen,
		BaseBranch:           "main",
		HeadBranch:           "feature/remote",
		HeadSHA:              strings.Repeat("a", 40),
		CollaborationAllowed: true,
		SourceRepository: models.RemoteContributionRepository{
			Host: host, Path: "contributor/widget-fork", RemoteURL: "https://" + host + "/contributor/widget-fork.git",
		},
	}
}

func TestContributionHostsDestinationScopeUsesTheRepositoryHost(t *testing.T) {
	issuer := &fakeGitHubCredentialLeaseIssuer{lease: GitHubCredentialLease{Token: "opaque-lease"}}
	exec := newTestExecutor(t, &mockAgentManager{}, newMockRepository())
	exec.SetGitHubCredentialBroker(issuer, contributionHostsBrokerURL)
	info := &repoInfo{
		RepositoryID: "repo-1", Repository: contributionHostsRepository("octo.ghe.com"),
		ContributionDestination: contributionHostsDestination("octo.ghe.com"),
	}

	if err := exec.configureGitHubCredentialBroker(context.Background(), contributionHostsRequest(), info); err != nil {
		t.Fatalf("configureGitHubCredentialBroker() error = %v", err)
	}

	if len(issuer.requests) != 2 {
		t.Fatalf("lease requests = %+v, want canonical and destination scopes", issuer.requests)
	}
	for _, scope := range issuer.requests {
		if scope.Host != "octo.ghe.com" {
			t.Fatalf("lease scope host = %q, want octo.ghe.com: %+v", scope.Host, scope)
		}
	}
	if issuer.requests[1].Path != "/agent/widget.git" {
		t.Fatalf("destination scope = %+v", issuer.requests[1])
	}
}

func TestContributionHostsRemoteContributionScopeUsesTheRepositoryHost(t *testing.T) {
	issuer := &fakeGitHubCredentialLeaseIssuer{lease: GitHubCredentialLease{Token: "opaque-lease"}}
	exec := newTestExecutor(t, &mockAgentManager{}, newMockRepository())
	exec.SetGitHubCredentialBroker(issuer, contributionHostsBrokerURL)
	info := &repoInfo{
		RepositoryID: "repo-1", Repository: contributionHostsRepository("ghe.example.com"),
		RemoteContribution: contributionHostsRemoteContribution("ghe.example.com"),
	}

	if err := exec.configureGitHubCredentialBroker(context.Background(), contributionHostsRequest(), info); err != nil {
		t.Fatalf("configureGitHubCredentialBroker() error = %v", err)
	}

	if len(issuer.requests) != 2 || issuer.requests[1].Host != "ghe.example.com" ||
		issuer.requests[1].Path != "/contributor/widget-fork.git" {
		t.Fatalf("lease requests = %+v, want a fork scope on ghe.example.com", issuer.requests)
	}
}

func TestContributionHostsRejectBindingsOnAnotherHost(t *testing.T) {
	tests := []struct {
		name string
		info *repoInfo
	}{
		{name: "destination on github.com for a tenant repository", info: &repoInfo{
			RepositoryID: "repo-1", Repository: contributionHostsRepository("octo.ghe.com"),
			ContributionDestination: contributionHostsDestination("github.com"),
		}},
		{name: "destination on a tenant for a github.com repository", info: &repoInfo{
			RepositoryID: "repo-1", Repository: contributionHostsRepository("github.com"),
			ContributionDestination: contributionHostsDestination("octo.ghe.com"),
		}},
		{name: "fork on another enterprise host", info: &repoInfo{
			RepositoryID: "repo-1", Repository: contributionHostsRepository("octo.ghe.com"),
			RemoteContribution: contributionHostsRemoteContribution("ghe.example.com"),
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			issuer := &fakeGitHubCredentialLeaseIssuer{lease: GitHubCredentialLease{Token: "opaque-lease"}}
			exec := newTestExecutor(t, &mockAgentManager{}, newMockRepository())
			exec.SetGitHubCredentialBroker(issuer, contributionHostsBrokerURL)

			err := exec.configureGitHubCredentialBroker(context.Background(), contributionHostsRequest(), test.info)

			if err == nil {
				t.Fatalf("configureGitHubCredentialBroker() accepted a binding on another host: %+v", issuer.requests)
			}
			for _, scope := range issuer.requests {
				if scope.Path != "/acme/widget.git" {
					t.Fatalf("lease issued for a binding on another host: %+v", scope)
				}
			}
		})
	}
}
