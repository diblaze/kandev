package orchestrator

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/kandev/kandev/internal/common/logger"
)

func newAutomationHostsService(t *testing.T) (*Service, *hostRecordingResolver, *mockGitHubService) {
	t.Helper()
	log, err := logger.NewFromZap(zap.NewNop())
	require.NoError(t, err)
	resolver := &hostRecordingResolver{}
	githubService := &mockGitHubService{}
	return &Service{repositoryResolver: resolver, githubService: githubService, logger: log}, resolver, githubService
}

func TestGitHubPRTriggerRepositoryResolvesOnTheHTMLURLHost(t *testing.T) {
	for _, tc := range []struct {
		name, htmlURL string
		wantHosts     []string
	}{
		{"enterprise pull request", "https://octo.ghe.com/org/repo/pull/3", []string{"octo.ghe.com"}},
		{"github.com pull request", "https://github.com/org/repo/pull/3", []string{"github.com"}},
		{"payload without URL", "", []string{""}},
		{"malformed URL", "http://octo.ghe.com/org/repo/pull/3", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, resolver, _ := newAutomationHostsService(t)
			data, err := json.Marshal(map[string]any{"repo": "org/repo", "html_url": tc.htmlURL, "head_branch": "feat"})
			require.NoError(t, err)

			svc.resolveGitHubPRTriggerRepository(context.Background(), "workspace-1", data)

			require.Equal(t, tc.wantHosts, resolver.hosts)
		})
	}
}

func TestAutomationPRAssociationUsesTheHTMLURLHost(t *testing.T) {
	svc, _, githubService := newAutomationHostsService(t)
	data := json.RawMessage(`{"repo":"org/repo","number":3,"html_url":"https://octo.ghe.com/org/repo/pull/3"}`)

	svc.associateAutomationPR(context.Background(), "workspace-1", "task-1", "repo-1", data)

	require.Equal(t, 1, githubService.associateCalls)
	require.Equal(t, []string{"octo.ghe.com"}, githubService.recordedHosts())
	require.Equal(t, "workspace-1", githubService.lastAssociateWorkspaceID)
}
