package backendapp

import (
	"context"
	"testing"

	githubpkg "github.com/kandev/kandev/internal/github"
	executorpkg "github.com/kandev/kandev/internal/orchestrator/executor"
)

func TestGitHubPRBaseResolverHostsRouteToTheBaseRepositoryHost(t *testing.T) {
	pr := &githubpkg.PR{Number: 42, RepoOwner: "upstream", RepoName: "widgets", HeadRepoOwner: "upstream",
		HeadRepoName: "widgets", HeadBranch: "feature", BaseBranch: "main"}
	for _, tc := range []struct {
		name   string
		linked []*githubpkg.TaskPR
		lookup executorpkg.PRBaseLookup
		want   string
	}{
		{"attached repository host", nil, executorpkg.PRBaseLookup{ProviderHost: "https://octo.ghe.com"}, "octo.ghe.com"},
		{"legacy attached repository", nil, executorpkg.PRBaseLookup{}, "github.com"},
		{"linked task PR host", []*githubpkg.TaskPR{{TaskID: "task-1", RepositoryID: "repo-1", PRNumber: 42,
			HeadBranch: "feature", GitHubHost: "octo.ghe.com", Owner: "upstream", Repo: "widgets"}},
			executorpkg.PRBaseLookup{ProviderHost: "https://github.com"}, "octo.ghe.com"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &githubPRBaseLookupServiceStub{prs: tc.linked, pr: pr}
			lookup := tc.lookup
			lookup.TaskID, lookup.TaskRepositoryID, lookup.RepositoryID = "task-1", "task-repo-1", "repo-1"
			lookup.Number, lookup.CheckoutBranch = 42, "feature"
			lookup.AttachedOwner, lookup.AttachedRepository = "upstream", "widgets"

			_, _ = githubPRBaseResolver{service: service}.ResolvePRBase(context.Background(), "workspace-1", lookup)

			if len(service.getHosts) != 1 || service.getHosts[0] != tc.want {
				t.Fatalf("PR lookup hosts = %v, want [%s]", service.getHosts, tc.want)
			}
		})
	}
}

func TestGitHubPRBaseResolverHostsRejectMalformedRepositoryHost(t *testing.T) {
	service := &githubPRBaseLookupServiceStub{}
	lookup := executorpkg.PRBaseLookup{TaskRepositoryID: "task-repo-1", Number: 42,
		ProviderHost: "https://octo.ghe.com:8443", AttachedOwner: "upstream", AttachedRepository: "widgets"}

	_, err := githubPRBaseResolver{service: service}.ResolvePRBase(context.Background(), "workspace-1", lookup)

	if err == nil || len(service.getHosts) != 0 {
		t.Fatalf("error = %v, lookups %v; want a host error and no lookup", err, service.getHosts)
	}
}
