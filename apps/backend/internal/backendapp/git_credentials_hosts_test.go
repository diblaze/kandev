package backendapp

import (
	"context"
	"strings"
	"testing"

	"github.com/kandev/kandev/internal/gitcredentials"
	taskmodels "github.com/kandev/kandev/internal/task/models"
)

func gitCredentialHostsRepo(t *testing.T, host string, metadata map[string]interface{}) *fakeGitHubBrokerTaskRepository {
	t.Helper()
	return &fakeGitHubBrokerTaskRepository{
		task:    &taskmodels.Task{ID: "task-1", WorkspaceID: "workspace-1"},
		session: &taskmodels.TaskSession{ID: "session-1", TaskID: "task-1", State: taskmodels.TaskSessionStateRunning},
		repository: &taskmodels.Repository{
			ID: "repository-1", WorkspaceID: "workspace-1", Provider: "github", ProviderHost: "https://" + host,
			ProviderRepoID: "100", ProviderOwner: "acme", ProviderName: "widget",
			RemoteURL: "https://" + host + "/acme/widget.git",
		},
		links: []*taskmodels.TaskRepository{{TaskID: "task-1", RepositoryID: "repository-1", Metadata: metadata}},
	}
}

func gitCredentialHostsScope(host, path, identity, parent string) gitcredentials.Scope {
	return gitcredentials.Scope{
		ProviderID: "github", WorkspaceID: "workspace-1", TaskID: "task-1", SessionID: "session-1",
		RepositoryID: "repository-1", Host: host, Path: path, IdentityProviderID: identity, ParentProviderID: parent,
	}
}

func gitCredentialHostsDestinationMetadata(t *testing.T, host string) map[string]interface{} {
	t.Helper()
	destination := taskmodels.ContributionDestination{
		Version:  taskmodels.ContributionDestinationVersion,
		Provider: taskmodels.ContributionDestinationProviderGitHub,
		SourceRepository: taskmodels.ContributionDestinationRepository{
			Host: host, Path: "acme/widget", ProviderID: "100", RemoteURL: "https://" + host + "/acme/widget.git",
		},
		TargetRepository: taskmodels.ContributionDestinationRepository{
			Host: host, Path: "agent/widget", ProviderID: "200", RemoteURL: "https://" + host + "/agent/widget.git",
		},
	}
	metadata := map[string]interface{}{}
	if err := taskmodels.PutContributionDestination(metadata, &destination); err != nil {
		t.Fatalf("PutContributionDestination() error = %v", err)
	}
	return metadata
}

func gitCredentialHostsForkMetadata(t *testing.T, host string) map[string]interface{} {
	t.Helper()
	binding := taskmodels.RemoteContribution{
		Version:              taskmodels.RemoteContributionVersion,
		Provider:             taskmodels.RemoteContributionProviderGitHub,
		Kind:                 taskmodels.RemoteContributionKindPullRequest,
		CanonicalURL:         "https://" + host + "/acme/widget/pull/7",
		Number:               7,
		State:                taskmodels.RemoteContributionStateOpen,
		BaseBranch:           "main",
		HeadBranch:           "feature/remote",
		HeadSHA:              strings.Repeat("a", 40),
		CollaborationAllowed: true,
		SourceRepository: taskmodels.RemoteContributionRepository{
			Host: host, Path: "contributor/widget-fork", RemoteURL: "https://" + host + "/contributor/widget-fork.git",
		},
	}
	metadata := map[string]interface{}{}
	if err := taskmodels.PutRemoteContribution(metadata, &binding); err != nil {
		t.Fatalf("PutRemoteContribution() error = %v", err)
	}
	return metadata
}

func TestGitCredentialHostsAuthorizeEnterpriseScopesOnTheRepositoryHost(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		metadata func(*testing.T, string) map[string]interface{}
		scope    gitcredentials.Scope
	}{
		{name: "canonical tenant repository", host: "octo.ghe.com",
			scope: gitCredentialHostsScope("octo.ghe.com", "/acme/widget.git", "100", "")},
		{name: "canonical server repository", host: "ghe.example.com",
			scope: gitCredentialHostsScope("ghe.example.com", "/acme/widget.git", "100", "")},
		{name: "contribution destination on the tenant", host: "octo.ghe.com", metadata: gitCredentialHostsDestinationMetadata,
			scope: gitCredentialHostsScope("octo.ghe.com", "/agent/widget.git", "200", "100")},
		{name: "contribution fork on the server", host: "ghe.example.com", metadata: gitCredentialHostsForkMetadata,
			scope: gitCredentialHostsScope("ghe.example.com", "/contributor/widget-fork.git", "", "")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var metadata map[string]interface{}
			if test.metadata != nil {
				metadata = test.metadata(t, test.host)
			}
			verifier := &fakeContributionDestinationVerifier{}
			authorizer := &githubBrokerScopeAuthorizer{repo: gitCredentialHostsRepo(t, test.host, metadata), provider: verifier}

			if err := authorizer.AuthorizeGitCredential(context.Background(), test.scope); err != nil {
				t.Fatalf("AuthorizeGitCredential() error = %v", err)
			}
			for _, host := range verifier.hosts {
				if host != test.host {
					t.Fatalf("destination verified on %q, want %q", host, test.host)
				}
			}
		})
	}
}

func TestGitCredentialHostsDenyScopesOffTheRepositoryHost(t *testing.T) {
	tests := []struct {
		name     string
		repoHost string
		metadata func(*testing.T, string) map[string]interface{}
		bindHost string
		scope    gitcredentials.Scope
	}{
		{name: "tenant scope for a github.com repository", repoHost: "github.com",
			scope: gitCredentialHostsScope("octo.ghe.com", "/acme/widget.git", "100", "")},
		{name: "github.com scope for a tenant repository", repoHost: "octo.ghe.com",
			scope: gitCredentialHostsScope("github.com", "/acme/widget.git", "100", "")},
		{name: "server scope for a tenant repository", repoHost: "octo.ghe.com",
			scope: gitCredentialHostsScope("ghe.example.com", "/acme/widget.git", "100", "")},
		{name: "stale provider identity on the tenant", repoHost: "octo.ghe.com",
			scope: gitCredentialHostsScope("octo.ghe.com", "/acme/widget.git", "999", "")},
		{name: "github.com destination for a tenant repository", repoHost: "octo.ghe.com",
			metadata: gitCredentialHostsDestinationMetadata, bindHost: "github.com",
			scope: gitCredentialHostsScope("octo.ghe.com", "/agent/widget.git", "200", "100")},
		{name: "github.com fork for a tenant repository", repoHost: "octo.ghe.com",
			metadata: gitCredentialHostsForkMetadata, bindHost: "github.com",
			scope: gitCredentialHostsScope("octo.ghe.com", "/contributor/widget-fork.git", "", "")},
		{name: "tenant fork under a github.com scope", repoHost: "github.com",
			metadata: gitCredentialHostsForkMetadata, bindHost: "octo.ghe.com",
			scope: gitCredentialHostsScope("github.com", "/contributor/widget-fork.git", "", "")},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var metadata map[string]interface{}
			if test.metadata != nil {
				metadata = test.metadata(t, test.bindHost)
			}
			authorizer := &githubBrokerScopeAuthorizer{
				repo: gitCredentialHostsRepo(t, test.repoHost, metadata), provider: &fakeContributionDestinationVerifier{},
			}

			if err := authorizer.AuthorizeGitCredential(context.Background(), test.scope); err == nil {
				t.Fatalf("AuthorizeGitCredential() accepted %+v", test.scope)
			}
		})
	}
}
