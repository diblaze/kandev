package service

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/kandev/kandev/internal/task/models"
)

type workspaceGitHubHostsStub struct {
	hosts []string
	err   error
}

func (s workspaceGitHubHostsStub) WorkspaceGitHubHosts(context.Context, string) ([]string, error) {
	return s.hosts, s.err
}

func writeOriginRepo(t *testing.T, origin string) string {
	t.Helper()
	repoPath := t.TempDir()
	makeRepo(t, repoPath)
	config := "[core]\n\trepositoryformatversion = 0\n[remote \"origin\"]\n\turl = " + origin +
		"\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n"
	if err := os.WriteFile(filepath.Join(repoPath, ".git", "config"), []byte(config), 0o644); err != nil {
		t.Fatalf("write .git/config: %v", err)
	}
	return repoPath
}

func TestEnterpriseRepositoryIdentityLocalOrigin(t *testing.T) {
	for _, tc := range []struct {
		name                                       string
		origin                                     string
		reader                                     *workspaceGitHubHostsStub
		wantProvider, wantHost, wantOwner, wantURL string
	}{
		{
			name:         "connected ghe.com tenant with scp origin",
			origin:       "org@octo.ghe.com:org/repo.git",
			reader:       &workspaceGitHubHostsStub{hosts: []string{"github.com", "octo.ghe.com"}},
			wantProvider: "github", wantHost: "https://octo.ghe.com", wantOwner: "org",
			wantURL: "octo@octo.ghe.com:org/repo.git",
		},
		{
			name:         "connected server host with https origin",
			origin:       "https://ghe.example.com/org/repo.git",
			reader:       &workspaceGitHubHostsStub{hosts: []string{"ghe.example.com"}},
			wantProvider: "github", wantHost: "https://ghe.example.com", wantOwner: "org",
			wantURL: "https://ghe.example.com/org/repo.git",
		},
		{
			name:    "unconnected host",
			origin:  "git@ghe.example.com:org/repo.git",
			reader:  &workspaceGitHubHostsStub{hosts: []string{"octo.ghe.com"}},
			wantURL: "ssh://ghe.example.com/org/repo",
		},
		{
			name:    "reader error",
			origin:  "https://ghe.example.com/org/repo.git",
			reader:  &workspaceGitHubHostsStub{err: errors.New("store unavailable")},
			wantURL: "https://ghe.example.com/org/repo",
		},
		{
			name:    "no reader",
			origin:  "https://ghe.example.com/org/repo.git",
			wantURL: "https://ghe.example.com/org/repo",
		},
		{
			name:         "github.com unchanged",
			origin:       "git@github.com:org/repo.git",
			reader:       &workspaceGitHubHostsStub{hosts: []string{"github.com"}},
			wantProvider: "github", wantHost: "https://github.com", wantOwner: "org",
			wantURL: "git@github.com:org/repo.git",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, repo := createTestService(t)
			ctx := context.Background()
			if err := repo.CreateWorkspace(ctx, &models.Workspace{ID: "ws-1", Name: "Workspace"}); err != nil {
				t.Fatalf("CreateWorkspace: %v", err)
			}
			if tc.reader != nil {
				svc.SetWorkspaceGitHubHostsReader(tc.reader)
			}
			created, err := svc.CreateRepository(ctx, &CreateRepositoryRequest{
				WorkspaceID: "ws-1", Name: "repo", SourceType: sourceTypeLocal,
				LocalPath: writeOriginRepo(t, tc.origin),
			})
			if err != nil {
				t.Fatalf("CreateRepository: %v", err)
			}
			stored, err := repo.GetRepository(ctx, created.ID)
			if err != nil {
				t.Fatalf("GetRepository: %v", err)
			}
			wantName := ""
			if tc.wantProvider != "" {
				wantName = "repo"
			}
			if stored.Provider != tc.wantProvider || stored.ProviderHost != tc.wantHost ||
				stored.ProviderOwner != tc.wantOwner || stored.ProviderName != wantName || stored.RemoteURL != tc.wantURL {
				t.Fatalf("stored provider=%q host=%q owner=%q name=%q remote=%q, want %q %q %q %q %q",
					stored.Provider, stored.ProviderHost, stored.ProviderOwner, stored.ProviderName, stored.RemoteURL,
					tc.wantProvider, tc.wantHost, tc.wantOwner, wantName, tc.wantURL)
			}
		})
	}
}
