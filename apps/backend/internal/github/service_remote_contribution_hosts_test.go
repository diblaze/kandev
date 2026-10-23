package github

import (
	"context"
	"errors"
	"strings"
	"testing"
)

const enterpriseContributionURL = "https://" + enterpriseTestHost + "/org/repo/pull/7"

func enterpriseContributionPR() *PR {
	return &PR{
		Number: 7, State: "open", RepoOwner: "org", RepoName: "repo",
		HeadRepoOwner: "contributor", HeadRepoName: "repo", HeadRepoID: 42,
		BaseRepoID: 99, BaseRepoOwner: "org", BaseRepoName: "repo", BaseDefaultBranch: "main",
		HeadBranch: "feature/remote", HeadSHA: strings.Repeat("a", 40), BaseBranch: "main",
		MaintainerCanModify: true, HTMLURL: enterpriseContributionURL,
	}
}

func TestRemoteContributionHostsParseURLOnAnyGitHubHost(t *testing.T) {
	host, owner, repo, number, err := parseGitHubContributionURL(enterpriseContributionURL)
	if err != nil || host != enterpriseTestHost || owner != "org" || repo != "repo" || number != 7 {
		t.Fatalf("parseGitHubContributionURL() = %q %q %q %d, %v", host, owner, repo, number, err)
	}
	for _, raw := range []string{"http://octo.ghe.com/org/repo/pull/7", "https://octo.ghe.com:8443/org/repo/pull/7"} {
		if _, _, _, _, err := parseGitHubContributionURL(raw); err == nil {
			t.Errorf("parseGitHubContributionURL(%q) returned nil error", raw)
		}
	}
}

func TestRemoteContributionHostsBuildRepositoriesOnTheItemHost(t *testing.T) {
	resolution, err := buildGitHubRemoteContribution(enterpriseTestHost, "org", "repo", 7, enterpriseContributionPR())
	if err != nil {
		t.Fatalf("buildGitHubRemoteContribution() error = %v", err)
	}
	source := resolution.Binding.SourceRepository
	if resolution.Binding.CanonicalURL != enterpriseContributionURL ||
		source.Host != enterpriseTestHost || source.RemoteURL != "https://octo.ghe.com/contributor/repo.git" {
		t.Fatalf("binding = %+v, want source and canonical URL on %s", resolution.Binding, enterpriseTestHost)
	}
	if resolution.TargetHost != "https://"+enterpriseTestHost ||
		resolution.TargetRemoteURL != "https://octo.ghe.com/org/repo.git" {
		t.Fatalf("target = %q %q, want %s", resolution.TargetHost, resolution.TargetRemoteURL, enterpriseTestHost)
	}
}

func TestRemoteContributionHostsResolveOnlyThroughTheURLHost(t *testing.T) {
	fixture := newHostRoutingFixture(t)
	enterprise := fixture.provider.clients[enterpriseTestHost].(*MockClient)
	enterprise.AddRepos("org", []GitHubRepo{{FullName: "org/repo", Owner: "org", Name: "repo"}})
	enterprise.AddPR(enterpriseContributionPR())

	if _, err := fixture.service.ResolveRemoteContributionForWorkspace(
		context.Background(), testWorkspaceID, "", enterpriseContributionURL,
	); err != nil {
		t.Fatalf("ResolveRemoteContributionForWorkspace() error = %v", err)
	}
	assertOnlyHostLoaded(t, fixture, enterpriseTestHost)

	_, err := fixture.service.ResolveRemoteContributionForWorkspace(
		context.Background(), testWorkspaceID, "", "https://"+unconnectedTestHost+"/org/repo/pull/7")
	var notConnected *HostNotConnectedError
	if !errors.As(err, &notConnected) || notConnected.Host != unconnectedTestHost {
		t.Fatalf("error = %v, want HostNotConnectedError for %s", err, unconnectedTestHost)
	}
}
