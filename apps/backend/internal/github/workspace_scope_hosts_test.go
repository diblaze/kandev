package github

import (
	"context"
	"errors"
	"testing"
)

func TestWorkspaceScopeHostsRepositoryListMatchesOnlyItsHost(t *testing.T) {
	service := NewService(nil, AuthMethodPAT, nil, newTestStore(t), nil, testLogger(t))
	ctx := context.Background()
	if err := service.UpsertWorkspaceSettings(ctx, &WorkspaceSettings{
		WorkspaceID:   testWorkspaceID,
		RepoScopeMode: RepoScopeModeRepos,
		RepoScopeRepos: []RepoFilter{
			{Host: "OCTO.ghe.com", Owner: "octo", Name: "app"},
			{Owner: "acme", Name: "web"},
		},
	}); err != nil {
		t.Fatalf("save workspace settings: %v", err)
	}

	for _, tc := range []struct {
		host, owner, repo string
		allowed           bool
	}{
		{enterpriseTestHost, "octo", "app", true},
		{defaultGitHubHost, "octo", "app", false},
		{"", "acme", "web", true},
		{defaultGitHubHost, "acme", "web", true},
		{enterpriseTestHost, "acme", "web", false},
	} {
		err := service.ensureRepositoryInWorkspaceScope(ctx, testWorkspaceID, tc.host, tc.owner, tc.repo)
		if tc.allowed && err != nil {
			t.Errorf("%s %s/%s denied: %v", tc.host, tc.owner, tc.repo, err)
		}
		if !tc.allowed && !errors.Is(err, ErrRepoNotResolvable) {
			t.Errorf("%s %s/%s error = %v, want out of scope", tc.host, tc.owner, tc.repo, err)
		}
	}
}

func TestWorkspaceScopeHostsSameRepositoryOnTwoHostsIsKeptTwice(t *testing.T) {
	service := NewService(nil, AuthMethodPAT, nil, newTestStore(t), nil, testLogger(t))
	ctx := context.Background()
	if err := service.UpsertWorkspaceSettings(ctx, &WorkspaceSettings{
		WorkspaceID:   testWorkspaceID,
		RepoScopeMode: RepoScopeModeRepos,
		RepoScopeRepos: []RepoFilter{
			{Host: enterpriseTestHost, Owner: "octo", Name: "app"},
			{Host: defaultGitHubHost, Owner: "octo", Name: "app"},
		},
	}); err != nil {
		t.Fatalf("save workspace settings: %v", err)
	}

	settings, err := service.GetWorkspaceSettings(ctx, testWorkspaceID)

	if err != nil {
		t.Fatalf("load workspace settings: %v", err)
	}
	if len(settings.RepoScopeRepos) != 2 || settings.RepoScopeRepos[0].Host != enterpriseTestHost ||
		settings.RepoScopeRepos[1].Host != "" {
		t.Fatalf("scope repos = %+v", settings.RepoScopeRepos)
	}
}
