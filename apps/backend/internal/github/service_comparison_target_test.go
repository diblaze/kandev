package github

import "testing"

func TestGitHubComparisonTargetCandidateUsesExplicitForkAndBaseRepositories(t *testing.T) {
	candidate, ok := githubComparisonTargetCandidate(defaultGitHubHost, &PR{
		Number:        17,
		HeadBranch:    "feature/fork",
		BaseBranch:    "main",
		HeadRepoID:    42,
		HeadRepoOwner: "contributor",
		HeadRepoName:  "widget-fork",
		BaseRepoID:    99,
		BaseRepoOwner: "upstream",
		BaseRepoName:  "widget",
	})
	if !ok {
		t.Fatal("githubComparisonTargetCandidate() rejected a complete PR identity")
	}
	if got := candidate.HeadRepository.Path; got != "contributor/widget-fork" {
		t.Fatalf("head repository = %q, want contributor/widget-fork", got)
	}
	if got := candidate.TargetRepository.Path; got != "upstream/widget" {
		t.Fatalf("target repository = %q, want upstream/widget", got)
	}
	if candidate.HeadRepository.ProviderID != "42" || candidate.TargetRepository.ProviderID != "99" {
		t.Fatalf("repository IDs = (%q, %q), want (42, 99)", candidate.HeadRepository.ProviderID, candidate.TargetRepository.ProviderID)
	}
}

func TestGitHubComparisonTargetCandidateRejectsMissingHeadRepositoryIdentity(t *testing.T) {
	if _, ok := githubComparisonTargetCandidate(defaultGitHubHost, &PR{
		Number:        17,
		HeadBranch:    "feature/fork",
		BaseBranch:    "main",
		BaseRepoOwner: "upstream",
		BaseRepoName:  "widget",
	}); ok {
		t.Fatal("githubComparisonTargetCandidate() accepted a PR without explicit head identity")
	}
}

func TestGitHubComparisonTargetCandidateUsesItemHost(t *testing.T) {
	candidate, ok := githubComparisonTargetCandidate(enterpriseTestHost, &PR{
		Number: 3, HeadBranch: "feature", BaseBranch: "main",
		HeadRepoOwner: "contributor", HeadRepoName: "app", BaseRepoOwner: "octo", BaseRepoName: "app",
	})
	if !ok {
		t.Fatal("githubComparisonTargetCandidate() rejected an enterprise PR identity")
	}
	head, target := candidate.HeadRepository, candidate.TargetRepository
	if head.Host != enterpriseTestHost || head.RemoteURL != "https://octo.ghe.com/contributor/app.git" ||
		target.Host != enterpriseTestHost || target.RemoteURL != "https://octo.ghe.com/octo/app.git" {
		t.Fatalf("repositories = %#v / %#v, want both on %s", head, target, enterpriseTestHost)
	}
}
