package backendapp

import (
	"context"
	"errors"
	"testing"

	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/github"
	taskservice "github.com/kandev/kandev/internal/task/service"
)

func TestCodeHostBranchListerHostsRouteToTheRepositoryHost(t *testing.T) {
	service := github.NewService(nil, github.AuthMethodPAT, nil, newStatusSummaryTestStore(t), nil, logger.Default())
	t.Cleanup(service.Stop)
	adapter := codeHostBranchListerAdapter{github: service}
	source := taskservice.RemoteBranchSource{WorkspaceID: "ws-1", Provider: "github", Owner: "octo", Name: "app"}

	source.ProviderHost = "https://octo.ghe.com"
	_, enterpriseErr := adapter.ListRepoBranches(context.Background(), source)
	source.ProviderHost = "https://octo.ghe.com:8443"
	_, malformedErr := adapter.ListRepoBranches(context.Background(), source)

	var notConnected *github.HostNotConnectedError
	if !errors.As(enterpriseErr, &notConnected) || notConnected.Host != "octo.ghe.com" {
		t.Fatalf("enterprise error = %v, want the octo.ghe.com connection to be resolved", enterpriseErr)
	}
	if malformedErr == nil || errors.As(malformedErr, &notConnected) {
		t.Fatalf("malformed host error = %v, want a host validation error", malformedErr)
	}
}
