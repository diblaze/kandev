package github

import (
	"context"
	"testing"
)

func TestContributionForkHostsBuildDestinationOnTheItemHost(t *testing.T) {
	canonical := testGitHubRepository("kdlbs/kandev", "100", false)
	canonical.CloneURL = "https://" + enterpriseTestHost + "/kdlbs/kandev.git"
	fork := testGitHubRepository("alice/kandev", "200", true)
	fork.CloneURL = ""
	fork.Fork = true
	fork.ParentID = canonical.ID
	fork.ParentFullName = canonical.FullName
	client := &forkResolverFakeClient{
		user:  "alice",
		repos: map[string]*GitHubRepository{"kdlbs/kandev": canonical, "alice/kandev": fork},
	}

	result, err := resolveContributionFork(
		context.Background(), client, AuthPrincipal{Kind: AuthPrincipalHuman, Login: "alice"},
		repoRef{enterpriseTestHost, "kdlbs", "kandev"}, true,
	)
	if err != nil {
		t.Fatalf("resolveContributionFork: %v", err)
	}
	source, target := result.Destination.SourceRepository, result.Destination.TargetRepository
	if source.Host != enterpriseTestHost || source.RemoteURL != canonical.CloneURL ||
		target.Host != enterpriseTestHost || target.RemoteURL != "https://octo.ghe.com/alice/kandev.git" {
		t.Fatalf("destination = %#v / %#v, want both on %s", source, target, enterpriseTestHost)
	}
}
