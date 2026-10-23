package models

import "testing"

func enterpriseRemoteContribution(host string) RemoteContribution {
	return RemoteContribution{
		Version:      RemoteContributionVersion,
		Provider:     RemoteContributionProviderGitHub,
		Kind:         RemoteContributionKindPullRequest,
		CanonicalURL: "https://" + host + "/org/repo/pull/7",
		Number:       7,
		State:        RemoteContributionStateOpen,
		BaseBranch:   "main",
		HeadBranch:   "feature/fix",
		HeadSHA:      "0123456789abcdef0123456789abcdef01234567",
		SourceRepository: RemoteContributionRepository{
			Host: host, Path: "contributor/repo", ProviderID: "42",
			RemoteURL: "https://" + host + "/contributor/repo.git",
		},
	}
}

func enterpriseContributionDestination(host string) ContributionDestination {
	destination := testContributionDestination()
	destination.SourceRepository.Host = host
	destination.SourceRepository.RemoteURL = "https://" + host + "/kdlbs/kandev.git"
	destination.TargetRepository.Host = host
	destination.TargetRepository.RemoteURL = "https://" + host + "/alice/kandev.git"
	return destination
}

func TestRemoteContributionAcceptsEnterpriseGitHubHost(t *testing.T) {
	contribution := enterpriseRemoteContribution("octo.ghe.com")
	if err := contribution.Validate(); err != nil {
		t.Fatalf("Validate() rejected an enterprise pull request: %v", err)
	}
}

func TestRemoteContributionRejectsGitHubHostWithPort(t *testing.T) {
	contribution := enterpriseRemoteContribution("octo.ghe.com:8443")
	if err := contribution.Validate(); err == nil {
		t.Fatal("Validate() accepted a GitHub host with a port")
	}
}

func TestContributionDestinationAcceptsEnterpriseGitHubHost(t *testing.T) {
	destination := enterpriseContributionDestination("octo.ghe.com")
	if err := destination.Validate(); err != nil {
		t.Fatalf("Validate() rejected an enterprise destination: %v", err)
	}
}

func TestContributionDestinationRejectsGitHubHostWithPort(t *testing.T) {
	destination := enterpriseContributionDestination("octo.ghe.com:8443")
	if err := destination.Validate(); err == nil {
		t.Fatal("Validate() accepted a GitHub host with a port")
	}
}
