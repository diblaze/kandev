package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/kandev/kandev/internal/github"
	"github.com/kandev/kandev/internal/task/models"
	sqliterepo "github.com/kandev/kandev/internal/task/repository/sqlite"
)

const enterpriseRepoHost = "octo.ghe.com"

func seedGitHubHostRepository(t *testing.T, repo *sqliterepo.Repository, providerHost string) {
	t.Helper()
	ctx := context.Background()
	now := time.Now().UTC()
	seedSession(t, repo, "t1", "s1", "step1")
	if err := repo.CreateRepository(ctx, &models.Repository{
		ID: "repo1", WorkspaceID: "ws1", Name: "kandev", SourceType: "provider", Provider: "github",
		ProviderHost: providerHost, ProviderOwner: "myorg", ProviderName: "kandev", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if err := repo.CreateTaskRepository(ctx, &models.TaskRepository{
		ID: "tr1", TaskID: "t1", RepositoryID: "repo1", CheckoutBranch: "main", CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("create task repository: %v", err)
	}
	session, _ := repo.GetTaskSession(ctx, "s1")
	session.RepositoryID = "repo1"
	if err := repo.UpdateTaskSession(ctx, session); err != nil {
		t.Fatalf("update session: %v", err)
	}
}

func newGitHubHostService(t *testing.T, providerHost string) (*Service, *mockGitHubService) {
	t.Helper()
	svc, ghSvc, _ := newGitHubHostServiceWithRepo(t, providerHost)
	return svc, ghSvc
}

func newGitHubHostServiceWithRepo(t *testing.T, providerHost string) (*Service, *mockGitHubService, *sqliterepo.Repository) {
	t.Helper()
	repo := setupTestRepo(t)
	seedGitHubHostRepository(t, repo, providerHost)
	client := github.NewMockClient()
	client.AddPR(&github.PR{Number: 1, Title: "PR", HTMLURL: "https://" + enterpriseRepoHost + "/myorg/kandev/pull/1",
		RepoOwner: "myorg", RepoName: "kandev", HeadBranch: "main", BaseBranch: "main"})
	ghSvc := &mockGitHubService{client: client}
	svc := createTestService(repo, newMockStepGetter(), newMockTaskRepo())
	svc.SetGitHubService(ghSvc)
	return svc, ghSvc, repo
}

func assertOnlyHost(t *testing.T, ghSvc *mockGitHubService, want string) {
	t.Helper()
	hosts := ghSvc.recordedHosts()
	if len(hosts) == 0 {
		t.Fatal("no host-routed GitHub call was made")
	}
	for _, host := range hosts {
		if host != want {
			t.Fatalf("GitHub calls used hosts %v, want only %q", hosts, want)
		}
	}
}

func TestGitHubRepositoryHostRoutesRepositoryDrivenCalls(t *testing.T) {
	for name, run := range map[string]func(context.Context, *Service){
		"push detection":   func(ctx context.Context, svc *Service) { svc.detectPushAndAssociatePR(ctx, "s1", "t1", "", "main") },
		"session PR watch": func(ctx context.Context, svc *Service) { svc.ensureSessionPRWatch(ctx, "t1", "s1", "main") },
		"on-demand check": func(ctx context.Context, svc *Service) {
			if _, err := svc.CheckSessionPR(ctx, "t1", "s1"); err != nil {
				t.Fatalf("CheckSessionPR: %v", err)
			}
		},
		"branch switch": func(ctx context.Context, svc *Service) {
			svc.resetPRWatchForBranchSwitch(ctx, "t1", "s1", "", "feature-y")
		},
	} {
		for _, tc := range []struct{ providerHost, want string }{
			{"https://" + enterpriseRepoHost, enterpriseRepoHost},
			{"", "github.com"},
		} {
			t.Run(name+"/"+tc.providerHost, func(t *testing.T) {
				svc, ghSvc := newGitHubHostService(t, tc.providerHost)

				run(context.Background(), svc)

				assertOnlyHost(t, ghSvc, tc.want)
			})
		}
	}
}

func TestGitHubRepositoryHostWatchTargetsCarryHost(t *testing.T) {
	svc, _, repo := newGitHubHostServiceWithRepo(t, "https://"+enterpriseRepoHost)
	attachWatchTestEnvironment(t, repo, []*models.TaskEnvironmentRepo{
		{ID: "wt1", RepositoryID: "repo1", WorktreeID: "tree1", WorktreeBranch: "main", Status: "active"},
	})

	targets, err := svc.ListTasksNeedingPRWatch(context.Background())

	if err != nil || len(targets) == 0 {
		t.Fatalf("ListTasksNeedingPRWatch = %+v, %v; want a target", targets, err)
	}
	for _, target := range targets {
		if target.GitHubHost != enterpriseRepoHost {
			t.Fatalf("watch target = %+v, want host %s", target, enterpriseRepoHost)
		}
	}
}

func TestGitHubRepositoryHostMalformedHostMakesNoGitHubCall(t *testing.T) {
	svc, ghSvc := newGitHubHostService(t, "https://octo.ghe.com:8443")

	svc.detectPushAndAssociatePR(context.Background(), "s1", "t1", "", "main")
	svc.ensureSessionPRWatch(context.Background(), "t1", "s1", "main")
	_, err := svc.CheckSessionPR(context.Background(), "t1", "s1")

	if err == nil {
		t.Fatal("CheckSessionPR accepted a malformed repository host")
	}
	if hosts := ghSvc.recordedHosts(); len(hosts) != 0 {
		t.Fatalf("malformed host still reached GitHub with hosts %v", hosts)
	}
}

func TestGitHubRepositoryHostTaskPRCallsUseTaskPRHost(t *testing.T) {
	svc, ghSvc := newGitHubHostService(t, "")
	pr := &github.TaskPR{TaskID: "t1", WorkspaceID: "ws1", GitHubHost: enterpriseRepoHost,
		Owner: "myorg", Repo: "kandev", PRNumber: 1, HeadSHA: "abc"}
	ghSvc.taskPR = pr

	svc.executeCIAutoMerge(context.Background(), pr, "signature")
	svc.handleTaskPRCIAutoFixLegacy(context.Background(), pr, &github.TaskCIOptionsResponse{})

	assertOnlyHost(t, ghSvc, enterpriseRepoHost)
}
