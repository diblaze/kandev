package backendapp

import (
	"context"
	"testing"

	taskmodels "github.com/kandev/kandev/internal/task/models"
)

func TestContributionDestinationHostsVerifyOnTheTargetHost(t *testing.T) {
	verifier := &fakeContributionDestinationVerifier{}
	authorizer := &githubBrokerScopeAuthorizer{provider: verifier}
	destination := taskmodels.ContributionDestination{
		SourceRepository: taskmodels.ContributionDestinationRepository{Host: "octo.ghe.com", Path: "acme/app", ProviderID: "1"},
		TargetRepository: taskmodels.ContributionDestinationRepository{Host: "octo.ghe.com", Path: "fork/app", ProviderID: "2"},
	}

	if err := authorizer.verifyContributionDestination(context.Background(), "ws-1", destination, "fork", "app"); err != nil {
		t.Fatalf("verifyContributionDestination: %v", err)
	}

	if len(verifier.hosts) != 1 || verifier.hosts[0] != "octo.ghe.com" {
		t.Fatalf("verified hosts = %v, want [octo.ghe.com]", verifier.hosts)
	}
}
