package service

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/task/models"
)

func TestRemoteRepositoryEnterpriseRegistration(t *testing.T) {
	for _, tc := range []struct {
		name, remoteURL, provider string
		hosts                     []string
		wantHost, wantRemote      string
	}{
		{
			name: "connected server host over https", remoteURL: "https://ghe.example.com/org/repo",
			hosts: []string{"ghe.example.com"}, wantHost: "https://ghe.example.com",
			wantRemote: "https://ghe.example.com/org/repo.git",
		},
		{
			name: "connected tenant scp url with non-git user", remoteURL: "org@octo.ghe.com:org/repo.git",
			hosts: []string{"octo.ghe.com"}, wantHost: "https://octo.ghe.com",
			wantRemote: "octo@octo.ghe.com:org/repo.git",
		},
		{
			name: "connected host with github provider hint", remoteURL: "https://ghe.example.com/org/repo",
			provider: "github", hosts: []string{"ghe.example.com"}, wantHost: "https://ghe.example.com",
			wantRemote: "https://ghe.example.com/org/repo.git",
		},
		{name: "unconnected host", remoteURL: "https://ghe.example.com/org/repo", hosts: []string{"octo.ghe.com"}},
		{name: "unconnected host with github provider hint", remoteURL: "https://ghe.example.com/org/repo", provider: "github"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, repo := createTestService(t)
			ctx := context.Background()
			createRepositorySelectionWorkspace(t, repo)
			svc.SetWorkspaceGitHubHostsReader(workspaceGitHubHostsStub{hosts: tc.hosts})
			registered, created, err := svc.RegisterRemoteRepository(ctx, &RegisterRemoteRepositoryRequest{
				WorkspaceID: "ws-1", RemoteURL: tc.remoteURL, Provider: tc.provider, DefaultBranch: "main",
			})
			if tc.wantHost == "" {
				assertRepositorySelectionError(t, err, RepositorySelectionErrorInvalid, "")
				if repositories, listErr := repo.ListRepositories(ctx, "ws-1"); listErr != nil || len(repositories) != 0 {
					t.Fatalf("repositories=%+v err=%v, want no writes", repositories, listErr)
				}
				return
			}
			if err != nil || !created {
				t.Fatalf("RegisterRemoteRepository created=%v err=%v", created, err)
			}
			stored, err := repo.GetRepository(ctx, registered.ID)
			if err != nil {
				t.Fatalf("GetRepository: %v", err)
			}
			if stored.Provider != "github" || stored.ProviderHost != tc.wantHost || stored.ProviderOwner != "org" ||
				stored.ProviderName != "repo" || stored.RemoteURL != tc.wantRemote {
				t.Fatalf("stored provider=%q host=%q owner=%q name=%q remote=%q",
					stored.Provider, stored.ProviderHost, stored.ProviderOwner, stored.ProviderName, stored.RemoteURL)
			}
		})
	}
}

func TestRemoteRepositoryEnterpriseTaskCreation(t *testing.T) {
	svc, _, repo := createTestService(t)
	ctx := context.Background()
	createRepositorySelectionWorkspace(t, repo)
	svc.SetWorkspaceGitHubHostsReader(workspaceGitHubHostsStub{hosts: []string{"ghe.example.com"}})
	if _, err := svc.CreateTask(ctx, &CreateTaskRequest{
		WorkspaceID: "ws-1", WorkflowID: "wf-1", WorkflowStepID: "step-1", Title: "enterprise task",
		Repositories: []TaskRepositoryInput{{RemoteURL: "https://ghe.example.com/org/repo", BaseBranch: "main"}},
	}); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	repositories, err := repo.ListRepositories(ctx, "ws-1")
	if err != nil || len(repositories) != 1 {
		t.Fatalf("repositories=%+v err=%v, want one", repositories, err)
	}
	if got := repositories[0]; got.Provider != "github" || got.ProviderHost != "https://ghe.example.com" ||
		got.ProviderOwner != "org" || got.ProviderName != "repo" {
		t.Fatalf("repository provider=%q host=%q owner=%q name=%q", got.Provider, got.ProviderHost, got.ProviderOwner, got.ProviderName)
	}
}

func TestRemoteRepositoryEnterpriseWorkspaceSource(t *testing.T) {
	for _, tc := range []struct {
		name, provider string
		hosts          []string
		wantErr        bool
	}{
		{name: "connected host", hosts: []string{"ghe.example.com"}},
		{name: "unconnected host with github provider hint", provider: "github", wantErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc, _, repo := createTestService(t)
			ctx := context.Background()
			createRepositorySelectionWorkspace(t, repo)
			svc.SetWorkspaceGitHubHostsReader(workspaceGitHubHostsStub{hosts: tc.hosts})
			source := WorkspaceSourceInput{
				Kind: WorkspaceSourceRepository, RemoteURL: "https://ghe.example.com/org/repo", Provider: tc.provider, BaseBranch: "main",
			}
			preview, previewErr := svc.previewWorkspaceRepositoryEntity(ctx, "ws-1", source)
			id, _, _, resolveErr := svc.resolveRepositoryWorkspaceSource(ctx, &models.Task{WorkspaceID: "ws-1"}, source)
			if tc.wantErr {
				if previewErr == nil || resolveErr == nil {
					t.Fatalf("preview err=%v resolve err=%v, want both rejected", previewErr, resolveErr)
				}
				return
			}
			if previewErr != nil || preview.ProviderHost != "https://ghe.example.com" || preview.Provider != "github" {
				t.Fatalf("preview=%+v err=%v", preview, previewErr)
			}
			stored, err := repo.GetRepository(ctx, id)
			if resolveErr != nil || err != nil || stored.ProviderHost != "https://ghe.example.com" {
				t.Fatalf("resolve err=%v stored=%+v get err=%v", resolveErr, stored, err)
			}
		})
	}
}
