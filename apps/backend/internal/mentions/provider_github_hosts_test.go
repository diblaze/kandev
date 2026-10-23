package mentions

import (
	"context"
	"errors"
	"testing"

	"github.com/kandev/kandev/internal/github"
	apiv1 "github.com/kandev/kandev/pkg/api/v1"
)

const mentionEnterpriseHost = "octo.ghe.com"

func TestGitHubProviderHostsSearchEveryConnectedHost(t *testing.T) {
	service := &fakeGitHubMentionService{
		hosts: []string{"github.com", mentionEnterpriseHost},
		prs:   []*github.PR{{NodeID: "PR_public", Number: 18, Title: "Public", RepoOwner: "acme", RepoName: "widgets"}},
		hostPRs: map[string][]*github.PR{mentionEnterpriseHost: {
			{NodeID: "PR_enterprise", Number: 3, Title: "Enterprise", RepoOwner: "org", RepoName: "repo"},
		}},
	}

	candidates, err := NewGitHubPullRequestProvider(service).Search(context.Background(),
		SearchRequest{WorkspaceID: "workspace-1", Query: "fix", Limit: 5})
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	want := []Candidate{
		{ID: "PR_public", Key: "acme/widgets#18", Title: "Public",
			URL: "https://github.com/acme/widgets/pull/18", Scope: "github.com/acme/widgets"},
		{ID: "PR_enterprise", Key: "org/repo#3", Title: "Enterprise",
			URL: "https://octo.ghe.com/org/repo/pull/3", Scope: "octo.ghe.com/org/repo"},
	}
	if len(candidates) != len(want) || candidates[0] != want[0] || candidates[1] != want[1] {
		t.Fatalf("candidates = %#v, want %#v", candidates, want)
	}
}

func TestGitHubProviderHostsKeepResultsOfHealthyHosts(t *testing.T) {
	service := &fakeGitHubMentionService{
		hosts: []string{mentionEnterpriseHost},
		err:   github.ErrNoClient,
		hostIssues: map[string][]*github.Issue{mentionEnterpriseHost: {
			{NodeID: "I_enterprise", Number: 7, Title: "Enterprise", RepoOwner: "org", RepoName: "repo"},
		}},
	}

	candidates, err := NewGitHubIssueProvider(service).Search(context.Background(),
		SearchRequest{WorkspaceID: "workspace-1", Query: "fix", Limit: 5})
	if err != nil || len(candidates) != 1 || candidates[0].Scope != "octo.ghe.com/org/repo" {
		t.Fatalf("Search = %#v, %v; want the enterprise candidate", candidates, err)
	}

	service.hostErrs = map[string]error{mentionEnterpriseHost: errors.New("upstream down")}
	_, err = NewGitHubIssueProvider(service).Search(context.Background(),
		SearchRequest{WorkspaceID: "workspace-1", Query: "fix", Limit: 5})
	var providerErr *ProviderError
	if !errors.As(err, &providerErr) {
		t.Fatalf("error = %v, want a provider error when every host fails", err)
	}
}

func enterpriseMentionReference() apiv1.EntityReference {
	scope := mentionEnterpriseHost + "/org/repo"
	return apiv1.EntityReference{
		Version: apiv1.EntityReferenceVersion, Ref: canonicalRef("github", "pull_request", scope, "PR_enterprise"),
		Provider: "github", Kind: "pull_request", ID: "PR_enterprise", Key: "org/repo#3", Title: "Enterprise",
		URL: "https://octo.ghe.com/org/repo/pull/3", Scope: scope,
	}
}

func TestGitHubProviderHostsAuthorizeEnterpriseReferenceOnItsHost(t *testing.T) {
	for _, tc := range []struct {
		name       string
		filterHost string
		wantErr    bool
	}{
		{"repository on the reference host", mentionEnterpriseHost, false},
		{"same repository on github.com", "", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeGitHubMentionService{settings: &github.WorkspaceSettings{
				WorkspaceID: "workspace-1", RepoScopeMode: github.RepoScopeModeRepos,
				RepoScopeRepos: []github.RepoFilter{{Host: tc.filterHost, Owner: "org", Name: "repo"}},
			}}
			authorizer := NewGitHubPullRequestProvider(service).(ReferenceAuthorizer)

			err := authorizer.AuthorizeReference(context.Background(), ReferenceAuthorizationRequest{
				WorkspaceID: "workspace-1", Purpose: ReferencePurposeSubmission, Reference: enterpriseMentionReference(),
			})
			if (err != nil) != tc.wantErr {
				t.Fatalf("AuthorizeReference() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestGitHubProviderHostsRejectReferenceURLOffTheScopeHost(t *testing.T) {
	service := &fakeGitHubMentionService{settings: &github.WorkspaceSettings{
		WorkspaceID: "workspace-1", RepoScopeMode: github.RepoScopeModeAll,
	}}
	authorizer := NewGitHubPullRequestProvider(service).(ReferenceAuthorizer)
	reference := enterpriseMentionReference()
	reference.URL = "https://github.com/org/repo/pull/3"

	if err := authorizer.AuthorizeReference(context.Background(), ReferenceAuthorizationRequest{
		WorkspaceID: "workspace-1", Purpose: ReferencePurposeSubmission, Reference: reference,
	}); err == nil {
		t.Fatal("a reference whose URL is off its scope host was authorized")
	}
}
