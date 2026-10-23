package backendapp

import (
	"context"
	"testing"

	taskmodels "github.com/kandev/kandev/internal/task/models"
)

func TestGitHubBrokerScopeAuthorizerUsesTheLeaseHost(t *testing.T) {
	repo := &fakeGitHubBrokerTaskRepository{
		task:    &taskmodels.Task{ID: "task-ghe", WorkspaceID: "workspace-ghe"},
		session: &taskmodels.TaskSession{ID: "session-ghe", TaskID: "task-ghe", State: taskmodels.TaskSessionStateRunning},
		repository: &taskmodels.Repository{
			ID: "repository-ghe", WorkspaceID: "workspace-ghe", Provider: "github",
			ProviderHost: "https://octo.ghe.com", ProviderOwner: "octo", ProviderName: "widget",
		},
		links: []*taskmodels.TaskRepository{{TaskID: "task-ghe", RepositoryID: "repository-ghe"}},
	}
	authorizer := &githubBrokerScopeAuthorizer{repo: repo}
	authorize := func(host string) error {
		return authorizer.AuthorizeGitHubRepository(
			context.Background(), "workspace-ghe", "task-ghe", "session-ghe", "repository-ghe", host, "octo", "widget",
		)
	}
	if err := authorize("octo.ghe.com"); err != nil {
		t.Fatalf("lease on the repository host was denied: %v", err)
	}
	if err := authorize("github.com"); err == nil {
		t.Fatal("lease on another host was accepted")
	}
}
