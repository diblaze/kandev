package executor

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/githubauth"
	"github.com/kandev/kandev/internal/task/models"
)

func configureGitHubHostsEnvTestExecutor(t *testing.T) *Executor {
	t.Helper()
	t.Setenv("PATH", t.TempDir())
	executor := newTestExecutor(t, &mockAgentManager{}, newMockRepository())
	executor.SetTaskGitCredentialPolicyResolver(fakeTaskGitCredentialPolicyResolver{
		policy: TaskGitCredentialPolicy{Mode: taskGitCredentialsModeExecutor},
	})
	return executor
}

func TestGitHubHostsEnvListsAttachedEnterpriseHostsSorted(t *testing.T) {
	executor := configureGitHubHostsEnvTestExecutor(t)
	gitlab := &repoInfo{RepositoryID: "repo-gitlab", Repository: &models.Repository{
		Provider: gitLabProviderID, ProviderHost: "https://gitlab.example.com", ProviderOwner: "acme", ProviderName: "widgets",
		RemoteURL: "https://gitlab.example.com/acme/widgets.git",
	}}
	req := executorHostGHBridgeRequest(map[string]string{})

	err := executor.configureGitCredentialBrokerForRepositories(context.Background(), req, []*repoInfo{
		executorHostGHBridgeRepository("repo-public", "github.com"),
		executorHostGHBridgeRepository("repo-tenant", "octo.ghe.com"),
		executorHostGHBridgeRepository("repo-server", "ghe.example.com"),
		gitlab,
	})
	if err != nil {
		t.Fatalf("configureGitCredentialBrokerForRepositories() error = %v", err)
	}

	if got := req.Env[githubauth.GitHubHostsEnv]; got != "ghe.example.com,octo.ghe.com" {
		t.Fatalf("%s = %q, want ghe.example.com,octo.ghe.com", githubauth.GitHubHostsEnv, got)
	}
}

func TestGitHubHostsEnvIsRemovedWithoutEnterpriseRepositories(t *testing.T) {
	tests := []struct {
		name  string
		infos []*repoInfo
	}{
		{name: "public repository only", infos: []*repoInfo{executorHostGHBridgeRepository("repo-public", "github.com")}},
		{name: "no repositories"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := configureGitHubHostsEnvTestExecutor(t)
			req := executorHostGHBridgeRequest(map[string]string{githubauth.GitHubHostsEnv: "stale.example.com"})

			if err := executor.configureGitCredentialBrokerForRepositories(context.Background(), req, test.infos); err != nil {
				t.Fatalf("configureGitCredentialBrokerForRepositories() error = %v", err)
			}

			if value, found := req.Env[githubauth.GitHubHostsEnv]; found {
				t.Fatalf("%s = %q, want removed", githubauth.GitHubHostsEnv, value)
			}
		})
	}
}
