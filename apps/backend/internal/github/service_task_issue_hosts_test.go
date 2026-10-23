package github

import (
	"context"
	"errors"
	"testing"

	taskmodels "github.com/kandev/kandev/internal/task/models"
)

const enterpriseIssueURL = "https://" + enterpriseTestHost + "/octo/app/issues/7"

func newTaskIssueHostsFixture(t *testing.T) (hostRoutingFixture, *fakeTaskIssueStore) {
	t.Helper()
	fixture := newHostRoutingFixture(t)
	enterprise := fixture.provider.clients[enterpriseTestHost].(*MockClient)
	enterprise.AddIssue(&Issue{
		Number: 7, Title: "Enterprise issue", HTMLURL: enterpriseIssueURL,
		RepoOwner: "octo", RepoName: "app", State: "open",
	})
	store := &fakeTaskIssueStore{
		task:     &taskmodels.Task{ID: "task-1", WorkspaceID: testWorkspaceID, Title: "Task"},
		entities: map[string]*taskmodels.Repository{},
	}
	fixture.service.SetTaskIssueStore(store)
	return fixture, store
}

func TestTaskIssueHostsLinkIssueURLOnConnectedEnterpriseHost(t *testing.T) {
	fixture, store := newTaskIssueHostsFixture(t)

	link, err := fixture.service.LinkTaskIssueForUser(context.Background(), "", "task-1",
		LinkTaskIssueRequest{Issue: enterpriseIssueURL})
	if err != nil {
		t.Fatalf("LinkTaskIssueForUser: %v", err)
	}
	if link.IssueURL != enterpriseIssueURL || store.updated[taskMetaIssueURL] != enterpriseIssueURL {
		t.Fatalf("link = %+v, stored = %v, want %s", link, store.updated[taskMetaIssueURL], enterpriseIssueURL)
	}
	for _, host := range fixture.connections.loadedHosts() {
		if host != enterpriseTestHost {
			t.Fatalf("issue link resolved the %s connection", host)
		}
	}
}

func TestTaskIssueHostsLinkIssueURLOnUnconnectedHostFails(t *testing.T) {
	fixture, store := newTaskIssueHostsFixture(t)

	_, err := fixture.service.LinkTaskIssueForUser(context.Background(), "", "task-1",
		LinkTaskIssueRequest{Issue: "https://" + unconnectedTestHost + "/octo/app/issues/7"})

	var notConnected *HostNotConnectedError
	if !errors.As(err, &notConnected) || notConnected.Host != unconnectedTestHost {
		t.Fatalf("error = %v, want HostNotConnectedError for %s", err, unconnectedTestHost)
	}
	if store.updated != nil {
		t.Fatalf("unconnected host stored a link: %v", store.updated)
	}
}

func TestTaskIssueHostsLinkIssueNumberOnRequestHost(t *testing.T) {
	fixture, store := newTaskIssueHostsFixture(t)

	_, err := fixture.service.LinkTaskIssueForUser(context.Background(), "", "task-1",
		LinkTaskIssueRequest{Host: enterpriseTestHost, Owner: "octo", Repo: "app", Number: 7})
	if err != nil {
		t.Fatalf("LinkTaskIssueForUser: %v", err)
	}
	if store.updated[taskMetaIssueURL] != enterpriseIssueURL {
		t.Fatalf("stored URL = %v, want %s", store.updated[taskMetaIssueURL], enterpriseIssueURL)
	}
}

func TestTaskIssueHostsRejectIssueOnAnotherHostThanTaskRepository(t *testing.T) {
	fixture, store := newTaskIssueHostsFixture(t)
	store.repos = []*taskmodels.TaskRepository{{TaskID: "task-1", RepositoryID: "repo-1"}}
	store.entities["repo-1"] = &taskmodels.Repository{ID: "repo-1", Provider: githubProviderName,
		ProviderHost: "https://github.com", ProviderOwner: "octo", ProviderName: "app"}

	_, err := fixture.service.LinkTaskIssueForUser(context.Background(), "", "task-1",
		LinkTaskIssueRequest{Issue: enterpriseIssueURL})

	if !errors.Is(err, ErrIssueRepositoryMismatch) {
		t.Fatalf("error = %v, want ErrIssueRepositoryMismatch", err)
	}
}

func TestTaskIssueHostsListKeepsEnterpriseIssueLinks(t *testing.T) {
	link, ok := taskIssueLinkFromMetadata(taskIssueMetadataRow{
		TaskID:   "task-1",
		Metadata: `{"issue_url":"` + enterpriseIssueURL + `","issue_number":7}`,
	})
	if !ok || link.IssueURL != enterpriseIssueURL || link.Owner != "octo" || link.Repo != "app" {
		t.Fatalf("taskIssueLinkFromMetadata = %+v, %v", link, ok)
	}
}
