package backendapp

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/github"
	"github.com/kandev/kandev/internal/task/models"
)

func TestTaskChangeRequestHostsReadStatusOfTheTaskPRHost(t *testing.T) {
	for _, tc := range []struct {
		name string
		prs  []*github.TaskPR
		want string
	}{
		{"enterprise PR", []*github.TaskPR{{TaskID: "task-1", GitHubHost: "octo.ghe.com", PRNumber: 1}}, "octo.ghe.com"},
		{"no PR", nil, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fake := &taskChangeRequestAutomationGitHubFake{prs: tc.prs}

			newGitHubTaskChangeRequestAdapter(fake).ReadTaskChangeRequests(
				context.Background(), &models.Task{ID: "task-1", WorkspaceID: "workspace-1"})

			if len(fake.hosts) != 1 || fake.hosts[0] != tc.want {
				t.Fatalf("status hosts = %q, want [%q]", fake.hosts, tc.want)
			}
		})
	}
}
