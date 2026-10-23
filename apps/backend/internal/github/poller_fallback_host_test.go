package github

import "testing"

func TestPRWatchFallbackTargetKeySeparatesHosts(t *testing.T) {
	public := &PRWatch{WorkspaceID: "ws", Owner: "o", Repo: "r", Branch: "b", PRNumber: 7}
	enterprise := *public
	enterprise.GitHubHost = "octo.ghe.com"
	explicitPublic := *public
	explicitPublic.GitHubHost = defaultGitHubHost

	if prWatchFallbackTargetKey(public) == prWatchFallbackTargetKey(&enterprise) {
		t.Fatal("watches on different hosts share a fallback target key")
	}
	if prWatchFallbackTargetKey(public) != prWatchFallbackTargetKey(&explicitPublic) {
		t.Fatal("an empty host and github.com produce different fallback target keys")
	}
}
