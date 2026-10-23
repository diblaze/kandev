package executor

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/task/models"
)

func TestResolveTaskRepoInfo_SameRepositoryEnterprisePRResolvesBase(t *testing.T) {
	const host = "octo.ghe.com"
	repository := models.ComparisonTargetRepository{
		Host: host, Path: "org/widgets", RemoteURL: "https://" + host + "/org/widgets.git",
	}
	target := models.ComparisonTarget{
		Version: models.ComparisonTargetVersion, Provider: models.ComparisonTargetProviderGitHub,
		Kind: models.ComparisonTargetKindPullRequest, Number: 42,
		HeadBranch: "feature", TargetBranch: "release",
		HeadRepository: repository, TargetRepository: repository,
	}
	resolver := &recordingPRBaseResolver{result: &models.PRBase{Target: target, OID: "0123456789abcdef0123456789abcdef01234567"}}
	repo := newMockRepository()
	repo.repositories["repo-1"] = &models.Repository{
		ID: "repo-1", WorkspaceID: "workspace-1", SourceType: sourceTypeLocal, LocalPath: t.TempDir(),
		Provider: "github", ProviderHost: "https://" + host, ProviderOwner: "org", ProviderName: "widgets",
		DefaultBranch: "main",
	}
	exec := newTestExecutor(t, &mockAgentManager{}, repo)
	exec.SetPRBaseResolver(resolver)

	info, err := exec.resolveTaskRepoInfo(context.Background(), &models.TaskRepository{
		ID: "task-repo-1", TaskID: "task-1", RepositoryID: "repo-1",
		BaseBranch: "main", CheckoutBranch: "feature", Metadata: map[string]interface{}{"pr_number": 42},
	})
	if err != nil {
		t.Fatalf("resolveTaskRepoInfo() error: %v", err)
	}
	if info.PRBase == nil || info.PRBase.Target.TargetRepository.Host != host || info.BaseBranch != "release" {
		t.Fatalf("resolved PR base = %#v, branch = %q, want the %s base on release", info.PRBase, info.BaseBranch, host)
	}
}
