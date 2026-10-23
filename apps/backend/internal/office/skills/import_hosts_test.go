package skills_test

import (
	"context"
	"errors"
	"testing"

	"github.com/kandev/kandev/internal/github"
	"github.com/kandev/kandev/internal/office/skills"
)

const enterpriseSkillHost = "octo.ghe.com"

type hostRepoReader struct {
	hosts      []string
	workspaces []string
	dirs       map[string][]github.RepoContentEntry
	files      map[string][]byte
}

func (r *hostRepoReader) ListRepoDirectoryForWorkspace(
	_ context.Context, workspaceID, host, _, _, path, _ string,
) ([]github.RepoContentEntry, error) {
	r.hosts, r.workspaces = append(r.hosts, host), append(r.workspaces, workspaceID)
	return r.dirs[path], nil
}

func (r *hostRepoReader) GetRepoFileContentForWorkspace(
	_ context.Context, workspaceID, host, _, _, path, _ string,
) ([]byte, error) {
	r.hosts, r.workspaces = append(r.hosts, host), append(r.workspaces, workspaceID)
	content, ok := r.files[path]
	if !ok {
		return nil, errors.New("not found")
	}
	return content, nil
}

func TestParseSourceHostsRecognizeEnterpriseURL(t *testing.T) {
	ps, err := skills.ParseSource("https://" + enterpriseSkillHost + "/acme/tools/tree/main/skills/deploy")

	if err != nil {
		t.Fatalf("ParseSource: %v", err)
	}
	if ps.Host != enterpriseSkillHost || ps.Owner != "acme" || ps.Repo != "tools" || ps.Slug != "deploy" {
		t.Fatalf("parsed = %+v", ps)
	}
}

func TestParseSourceHostsRejectEnterpriseURLWithPort(t *testing.T) {
	if _, err := skills.ParseSource("https://" + enterpriseSkillHost + ":8443/acme/tools"); err == nil {
		t.Fatal("ParseSource accepted a host with a port")
	}
}

func TestImportFromSourceHostsReadEnterpriseRepoThroughWorkspaceConnection(t *testing.T) {
	svc := newTestSkillService(t)
	reader := &hostRepoReader{
		dirs: map[string][]github.RepoContentEntry{
			"skills": {{Name: "deploy", Path: "skills/deploy", Type: github.RepoContentTypeDir}},
		},
		files: map[string][]byte{"skills/deploy/SKILL.md": []byte("---\nname: Deploy\n---\n# Deploy")},
	}
	svc.SetGitHubRepoReader(reader)

	result, err := svc.ImportFromSource(context.Background(), "ws-1",
		"https://"+enterpriseSkillHost+"/acme/tools", &mockFetcher{responses: map[string]fetchResp{}})

	if err != nil {
		t.Fatalf("import: %v", err)
	}
	if len(result.Skills) != 1 || result.Skills[0].Name != "Deploy" ||
		result.Skills[0].SourceLocator != "https://"+enterpriseSkillHost+"/acme/tools" {
		t.Fatalf("skills = %+v", *result.Skills[0])
	}
	for i, host := range reader.hosts {
		if host != enterpriseSkillHost || reader.workspaces[i] != "ws-1" {
			t.Fatalf("reads = %v in %v, want only %s in ws-1", reader.hosts, reader.workspaces, enterpriseSkillHost)
		}
	}
}

func TestImportFromSourceHostsGitHubDotComKeepsAnonymousFetch(t *testing.T) {
	svc := newTestSkillService(t)
	reader := &hostRepoReader{}
	svc.SetGitHubRepoReader(reader)
	fetcher := &mockFetcher{responses: map[string]fetchResp{
		"https://raw.githubusercontent.com/acme/tools/main/skills/deploy/SKILL.md": {
			body: []byte("---\nname: Deploy\n---\n# Deploy"), status: 200,
		},
	}}

	if _, err := svc.ImportFromSource(context.Background(), "ws-1",
		"https://github.com/acme/tools/tree/main/skills/deploy", fetcher); err != nil {
		t.Fatalf("import: %v", err)
	}

	if len(reader.hosts) != 0 {
		t.Fatalf("github.com import used the workspace reader: %v", reader.hosts)
	}
}
