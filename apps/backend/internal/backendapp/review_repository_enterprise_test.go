package backendapp

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jmoiron/sqlx"

	"github.com/kandev/kandev/internal/db"
	"github.com/kandev/kandev/internal/github"
	"github.com/kandev/kandev/internal/repoclone"
	taskservice "github.com/kandev/kandev/internal/task/service"
)

type hostRecordingGitHubCloneCredentials struct {
	hosts []string
}

func (c *hostRecordingGitHubCloneCredentials) ResolveGitCredential(
	_ context.Context, _, _, host, _, _ string,
) (string, string, error) {
	c.hosts = append(c.hosts, host)
	return "x-access-token", "token-for-" + host, nil
}

func TestRepositoryCloneCredentialProviderRoutesGitHubByProviderHost(t *testing.T) {
	for _, tc := range []struct {
		name, providerHost, wantHost string
		wantErr                      bool
	}{
		{name: "enterprise host", providerHost: "https://octo.ghe.com", wantHost: "octo.ghe.com"},
		{name: "github.com", providerHost: "https://github.com", wantHost: "github.com"},
		{name: "legacy empty host", providerHost: "", wantHost: "github.com"},
		{name: "malformed host", providerHost: "ftp://octo.ghe.com", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			credentials := &hostRecordingGitHubCloneCredentials{}
			provider := repositoryCloneCredentialProvider{github: credentials}
			_, password, err := provider.ResolveGitCredential(t.Context(), repoclone.GitCredentialRequest{
				WorkspaceID: "workspace-1", Provider: "github", ProviderHost: tc.providerHost,
				CloneURL: "https://octo.ghe.com/org/repo.git", Owner: "org", Name: "repo",
			})
			if tc.wantErr {
				if err == nil || len(credentials.hosts) != 0 {
					t.Fatalf("err=%v hosts=%v, want error and no credential lookup", err, credentials.hosts)
				}
				return
			}
			if err != nil || password != "token-for-"+tc.wantHost {
				t.Fatalf("password=%q err=%v, want credential for %s", password, err, tc.wantHost)
			}
		})
	}
}

func TestResolveForReviewClonesOnGivenHost(t *testing.T) {
	harness := newBootStateTestHarness(t)
	ctx := context.Background()
	workspace, err := harness.taskSvc.CreateWorkspace(ctx, &taskservice.CreateWorkspaceRequest{Name: "Workspace"})
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	paths := []string{filepath.Join(t.TempDir(), "tenant"), filepath.Join(t.TempDir(), "dotcom")}
	for _, path := range paths {
		runGit(t, "", "init", "--quiet", path)
	}
	cloner := &recordingReviewRepositoryCloner{protocol: repoclone.ProtocolSSH, paths: paths}
	adapter := &repositoryResolverAdapter{cloner: cloner, taskSvc: harness.taskSvc, logger: newTestLogger()}

	tenantID, _, err := adapter.ResolveForReview(ctx, workspace.ID, "github", "octo.ghe.com", "org", "repo", "")
	if err != nil {
		t.Fatalf("enterprise ResolveForReview: %v", err)
	}
	if _, _, err := adapter.ResolveForReview(ctx, workspace.ID, "github", "", "org", "repo", ""); err != nil {
		t.Fatalf("github.com ResolveForReview: %v", err)
	}
	again, _, err := adapter.ResolveForReview(ctx, workspace.ID, "github", "octo.ghe.com", "org", "repo", "")
	if err != nil || again != tenantID {
		t.Fatalf("repeat enterprise ResolveForReview = %q, %v; want existing %q", again, err, tenantID)
	}

	wantURLs := []string{"octo@octo.ghe.com:org/repo.git", "git@github.com:org/repo.git"}
	wantHosts := []string{"https://octo.ghe.com", "https://github.com"}
	if !reflect.DeepEqual(cloner.cloneURLs, wantURLs) || !reflect.DeepEqual(cloner.providerHosts, wantHosts) {
		t.Fatalf("clone URLs=%v hosts=%v, want %v %v", cloner.cloneURLs, cloner.providerHosts, wantURLs, wantHosts)
	}
	stored, err := harness.taskSvc.GetRepository(ctx, tenantID)
	if err != nil || stored.ProviderHost != "https://octo.ghe.com" || stored.LocalPath != paths[0] {
		t.Fatalf("stored=%+v err=%v", stored, err)
	}
}

func TestWireGitHubRepositoryIdentityClassifiesConnectedHosts(t *testing.T) {
	harness := newBootStateTestHarness(t)
	ctx := context.Background()
	workspace, err := harness.taskSvc.CreateWorkspace(ctx, &taskservice.CreateWorkspaceRequest{Name: "Workspace"})
	if err != nil {
		t.Fatalf("CreateWorkspace: %v", err)
	}
	dbConn, err := db.OpenSQLite(filepath.Join(t.TempDir(), "github.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	sqlxDB := sqlx.NewDb(dbConn, "sqlite3")
	t.Cleanup(func() { _ = sqlxDB.Close() })
	if _, err := sqlxDB.Exec(`CREATE TABLE workspaces (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatalf("create workspaces table: %v", err)
	}
	if _, err := sqlxDB.Exec(`INSERT INTO workspaces (id) VALUES (?)`, workspace.ID); err != nil {
		t.Fatalf("insert workspace: %v", err)
	}
	store, err := github.NewStore(sqlxDB, sqlxDB)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	if err := store.UpsertWorkspaceConnection(ctx, &github.WorkspaceConnection{
		WorkspaceID: workspace.ID, Source: github.ConnectionSourcePAT, GitHubHost: "octo.ghe.com",
		Login: "octocat", Status: github.ConnectionStatusActive, CredentialGeneration: 1,
	}); err != nil {
		t.Fatalf("UpsertWorkspaceConnection: %v", err)
	}
	githubSvc := github.NewService(nil, github.AuthMethodPAT, nil, store, nil, newTestLogger())
	t.Cleanup(githubSvc.Stop)

	wireGitHubRepositoryIdentity(harness.taskSvc, githubSvc, newTestLogger())

	repoPath := filepath.Join(t.TempDir(), "repo")
	runGit(t, "", "init", "--quiet", repoPath)
	runGit(t, repoPath, "remote", "add", "origin", "org@octo.ghe.com:org/repo.git")
	created, err := harness.taskSvc.CreateRepository(ctx, &taskservice.CreateRepositoryRequest{
		WorkspaceID: workspace.ID, Name: "repo", SourceType: "local", LocalPath: repoPath,
	})
	if err != nil {
		t.Fatalf("CreateRepository: %v", err)
	}
	if created.Provider != "github" || created.ProviderHost != "https://octo.ghe.com" {
		t.Fatalf("repository provider=%q host=%q, want github on https://octo.ghe.com", created.Provider, created.ProviderHost)
	}
}
