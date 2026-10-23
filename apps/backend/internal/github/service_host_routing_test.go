package github

import (
	"context"
	"errors"
	"sync"
	"testing"
)

const unconnectedTestHost = "ghe.example.com"

// hostRecordingConnections serves one connection per host and records every
// host a credential resolution loads, cache hits included.
type hostRecordingConnections struct {
	mu          sync.Mutex
	connections map[string]*WorkspaceConnection
	loaded      []string
}

func (c *hostRecordingConnections) GetWorkspaceConnection(_ context.Context, _ string, host string) (*WorkspaceConnection, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loaded = append(c.loaded, host)
	return c.connections[host], nil
}

func (c *hostRecordingConnections) GetUserConnection(context.Context, string, string, string) (*UserConnection, error) {
	return nil, nil
}

func (c *hostRecordingConnections) loadedHosts() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]string(nil), c.loaded...)
}

type hostClientProvider struct {
	mu      sync.Mutex
	clients map[string]Client
	calls   int
}

func (p *hostClientProvider) ResolveAutomation(
	_ context.Context, connection *WorkspaceConnection, _ ResolveCredentialRequest,
) (*ResolvedCredential, error) {
	p.mu.Lock()
	p.calls++
	p.mu.Unlock()
	return &ResolvedCredential{
		Client:       p.clients[connection.GitHubHost],
		Capabilities: allTokenCapabilities(),
		Principal:    AuthPrincipal{Kind: AuthPrincipalHuman, Source: ConnectionSourcePAT, Login: connection.Login},
	}, nil
}

type noopTaskDeleter struct{}

func (noopTaskDeleter) DeleteTask(context.Context, string) error { return nil }

type hostRoutingFixture struct {
	service     *Service
	connections *hostRecordingConnections
	provider    *hostClientProvider
}

func newHostRoutingFixture(t *testing.T) hostRoutingFixture {
	t.Helper()
	clients := map[string]Client{}
	connections := map[string]*WorkspaceConnection{}
	for _, host := range []string{defaultGitHubHost, enterpriseTestHost} {
		client := NewMockClient()
		client.AddRepos("octo", []GitHubRepo{{FullName: "octo/app", Owner: "octo", Name: "app"}})
		clients[host] = client
		connections[host] = &WorkspaceConnection{WorkspaceID: testWorkspaceID, Source: ConnectionSourcePAT,
			GitHubHost: host, Login: "login-" + host, Status: ConnectionStatusActive, CredentialGeneration: 1}
	}
	service := NewService(nil, AuthMethodPAT, nil, newTestStore(t), nil, testLogger(t))
	recording := &hostRecordingConnections{connections: connections}
	provider := &hostClientProvider{clients: clients}
	service.resolver = NewCredentialResolver(recording, nil)
	service.resolver.SetAutomationProvider(provider)
	service.taskDeleter = noopTaskDeleter{}
	return hostRoutingFixture{service: service, connections: recording, provider: provider}
}

type hostRoutedCall struct {
	group string
	name  string
	call  func(context.Context, *Service, string) error
}

func hostRoutedCalls() []hostRoutedCall {
	const ws, owner, repo = testWorkspaceID, "octo", "app"
	return []hostRoutedCall{
		{"pr reads", "GetPRForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetPRForWorkspace(ctx, ws, "", h, owner, repo, 1)
			return err
		}},
		{"pr reads", "GetPRForAutomation", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetPRForAutomation(ctx, ws, h, owner, repo, 1)
			return err
		}},
		{"pr reads", "FindPRByBranchForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.FindPRByBranchForWorkspace(ctx, ws, h, owner, repo, "feat")
			return err
		}},
		{"pr writes", "MergePRForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, _, err := s.MergePRForWorkspace(ctx, ws, "", h, owner, repo, 1, "merge")
			return err
		}},
		{"pr writes", "MergePRForAutomation", func(ctx context.Context, s *Service, h string) error {
			return s.MergePRForAutomation(ctx, ws, h, owner, repo, 1, "merge", "abc123")
		}},
		{"reviews", "SubmitReviewForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.SubmitReviewForWorkspace(ctx, ws, "", h, owner, repo, 1, "APPROVE", "")
			return err
		}},
		{"reviews", "RequestReviewersForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.RequestReviewersForWorkspace(ctx, ws, "", h, owner, repo, 1, []string{"hubot"})
			return err
		}},
		{"checks", "GetPRStatusForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetPRStatusForWorkspace(ctx, ws, "", h, owner, repo, 1)
			return err
		}},
		{"checks", "GetPRStatusesBatchForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetPRStatusesBatchForWorkspace(ctx, ws, "", h, []PRRef{{Owner: owner, Repo: repo, Number: 1}})
			return err
		}},
		{"checks", "GetPRFeedbackForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetPRFeedbackForWorkspace(ctx, ws, "", h, owner, repo, 1)
			return err
		}},
		{"checks", "GetPRFeedbackForAutomation", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetPRFeedbackForAutomation(ctx, ws, h, owner, repo, 1)
			return err
		}},
		{"checks", "InvalidatePRFeedbackForAutomation", func(ctx context.Context, s *Service, h string) error {
			return s.InvalidatePRFeedbackForAutomation(ctx, ws, h, owner, repo, 1)
		}},
		{"files and commits", "GetPRFilesForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetPRFilesForWorkspace(ctx, ws, "", h, owner, repo, 1)
			return err
		}},
		{"files and commits", "GetPRCommitsForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetPRCommitsForWorkspace(ctx, ws, "", h, owner, repo, 1)
			return err
		}},
		{"files and commits", "GetPRCommitDetailForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetPRCommitDetailForWorkspace(ctx, ws, "", h, owner, repo, "0123456789abcdef0123456789abcdef01234567")
			return err
		}},
		{"issues", "GetIssueForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetIssueForWorkspace(ctx, ws, "", h, owner, repo, 1)
			return err
		}},
		{"branches and merge methods", "ListRepoBranchesForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.ListRepoBranchesForWorkspace(ctx, ws, h, owner, repo)
			return err
		}},
		{"branches and merge methods", "GetRepoMergeMethodsForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetRepoMergeMethodsForWorkspace(ctx, ws, "", h, owner, repo)
			return err
		}},
		{"branches and merge methods", "GetRepoMergeAccessForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetRepoMergeAccessForWorkspace(ctx, ws, "", h, owner, repo)
			return err
		}},
		{"contents", "ListRepoDirectoryForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.ListRepoDirectoryForWorkspace(ctx, ws, h, owner, repo, "", "")
			return err
		}},
		{"contents", "GetRepoFileContentForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.GetRepoFileContentForWorkspace(ctx, ws, h, owner, repo, "README.md", "")
			return err
		}},
		{"searches", "SearchUserPRsPagedForWorkspaceUser", func(ctx context.Context, s *Service, h string) error {
			_, err := s.SearchUserPRsPagedForWorkspaceUser(ctx, ws, "", h, "", "is:open", 1, 10)
			return err
		}},
		{"searches", "SearchUserIssuesPagedForWorkspaceUser", func(ctx context.Context, s *Service, h string) error {
			_, err := s.SearchUserIssuesPagedForWorkspaceUser(ctx, ws, "", h, "", "is:open", 1, 10)
			return err
		}},
		{"searches", "SearchUserPRsForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.SearchUserPRsForWorkspace(ctx, ws, "", h, "", "is:open")
			return err
		}},
		{"searches", "SearchUserIssuesForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.SearchUserIssuesForWorkspace(ctx, ws, "", h, "", "is:open")
			return err
		}},
		{"searches", "ListUserOrgsForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.ListUserOrgsForWorkspace(ctx, ws, "", h)
			return err
		}},
		{"searches", "SearchOrgReposForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.SearchOrgReposForWorkspace(ctx, ws, h, owner, "", 10)
			return err
		}},
		{"searches", "ListAutomationPRs", func(ctx context.Context, s *Service, h string) error {
			_, err := s.ListAutomationPRs(ctx, ws, h, owner, repo, "is:open")
			return err
		}},
		{"accessible repositories", "ListAccessibleReposForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.ListAccessibleReposForWorkspace(ctx, ws, "", h, "", 10)
			return err
		}},
		{"watches", "CheckPRWatchForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, _, err := s.CheckPRWatchForWorkspace(ctx, &PRWatch{WorkspaceID: ws, GitHubHost: h,
				TaskID: "task-1", Owner: owner, Repo: repo, Branch: "feat"})
			return err
		}},
		{"watches", "CheckReviewWatch", func(ctx context.Context, s *Service, h string) error {
			_, err := s.CheckReviewWatch(ctx, &ReviewWatch{ID: "rw-1", WorkspaceID: ws, GitHubHost: h})
			return err
		}},
		{"watches", "CheckIssueWatch", func(ctx context.Context, s *Service, h string) error {
			_, err := s.CheckIssueWatch(ctx, &IssueWatch{ID: "iw-1", WorkspaceID: ws, GitHubHost: h})
			return err
		}},
		{"watches", "CreateIssueWatchForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.CreateIssueWatchForWorkspace(ctx, &CreateIssueWatchRequest{WorkspaceID: ws, GitHubHost: h,
				WorkflowID: "wf", WorkflowStepID: "step", AgentProfileID: "agent", ExecutorProfileID: "exec"})
			return err
		}},
		{"cleanup", "CleanupMergedReviewTasks", func(ctx context.Context, s *Service, h string) error {
			_, err := s.CleanupMergedReviewTasks(ctx, &ReviewWatch{ID: "rw-1", WorkspaceID: ws, GitHubHost: h})
			return err
		}},
		{"cleanup", "CleanupClosedIssueTasks", func(ctx context.Context, s *Service, h string) error {
			_, err := s.CleanupClosedIssueTasks(ctx, &IssueWatch{ID: "iw-1", WorkspaceID: ws, GitHubHost: h})
			return err
		}},
		{"mentions", "SearchMentionIssuesForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.SearchMentionIssuesForWorkspace(ctx, ws, h, "fix", 5)
			return err
		}},
		{"mentions", "SearchMentionPullRequestsForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.SearchMentionPullRequestsForWorkspace(ctx, ws, h, "fix", 5)
			return err
		}},
		{"forks", "ResolveContributionForkForWorkspace", func(ctx context.Context, s *Service, h string) error {
			_, err := s.ResolveContributionForkForWorkspace(ctx, ws, h, owner, repo, false)
			return err
		}},
	}
}

func TestServiceHostRoutingUsesOnlyTheItemHostConnection(t *testing.T) {
	for _, tc := range hostRoutedCalls() {
		t.Run(tc.group+"/"+tc.name, func(t *testing.T) {
			fixture := newHostRoutingFixture(t)

			err := tc.call(context.Background(), fixture.service, enterpriseTestHost)

			var notConnected *HostNotConnectedError
			if errors.As(err, &notConnected) {
				t.Fatalf("connected host reported not connected: %v", err)
			}
			loaded := fixture.connections.loadedHosts()
			if len(loaded) == 0 {
				t.Fatalf("no credential was resolved (err %v)", err)
			}
			for _, host := range loaded {
				if host != enterpriseTestHost {
					t.Fatalf("resolved hosts = %v, want only %s (err %v)", loaded, enterpriseTestHost, err)
				}
			}
		})
	}
}

func TestServiceHostRoutingFailsClosedForUnconnectedHost(t *testing.T) {
	for _, tc := range hostRoutedCalls() {
		t.Run(tc.group+"/"+tc.name, func(t *testing.T) {
			fixture := newHostRoutingFixture(t)

			err := tc.call(context.Background(), fixture.service, unconnectedTestHost)

			var notConnected *HostNotConnectedError
			if !errors.As(err, &notConnected) || notConnected.Host != unconnectedTestHost {
				t.Fatalf("error = %v, want HostNotConnectedError for %s", err, unconnectedTestHost)
			}
			if fixture.provider.calls != 0 {
				t.Fatalf("unconnected host built %d clients", fixture.provider.calls)
			}
			for _, host := range fixture.connections.loadedHosts() {
				if host != unconnectedTestHost {
					t.Fatalf("unconnected item loaded the %s connection", host)
				}
			}
		})
	}
}

func TestServiceHostRoutingDefaultsEmptyHostToGitHubDotCom(t *testing.T) {
	fixture := newHostRoutingFixture(t)

	_, err := fixture.service.GetPRForWorkspace(context.Background(), testWorkspaceID, "", "", "octo", "app", 1)

	var notConnected *HostNotConnectedError
	if errors.As(err, &notConnected) {
		t.Fatalf("empty host reported not connected: %v", err)
	}
	for _, host := range fixture.connections.loadedHosts() {
		if host != defaultGitHubHost {
			t.Fatalf("empty host resolved %s", host)
		}
	}
}

func TestServiceHostRoutingCacheScopeSeparatesHosts(t *testing.T) {
	public := &ResolvedCredential{Principal: AuthPrincipal{WorkspaceID: testWorkspaceID, Login: "octocat",
		Kind: AuthPrincipalHuman, Source: ConnectionSourcePAT, GitHubHost: defaultGitHubHost}}
	enterprise := cloneResolvedCredential(public)
	enterprise.Principal.GitHubHost = enterpriseTestHost

	if credentialCacheScope(public, CredentialPurposeAutomation) == credentialCacheScope(enterprise, CredentialPurposeAutomation) {
		t.Fatal("two hosts with the same identity share a service cache scope")
	}
}
