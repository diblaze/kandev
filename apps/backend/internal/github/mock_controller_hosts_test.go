package github

import (
	"context"
	"net/http"
	"testing"
)

func TestMockControllerWorkspaceConnectionsUseHostQuery(t *testing.T) {
	router, _, store := setupWorkspaceAuthMockController(t)
	for _, host := range []string{"", enterpriseTestHost} {
		response := serveMockJSON(t, router, http.MethodPut,
			"/api/v1/github/mock/workspace-connections/workspace-1?host="+host,
			`{"source":"gh_cli","status":"active","login":"alice"}`)
		if response.Code != http.StatusOK {
			t.Fatalf("seed %q connection: %d %s", host, response.Code, response.Body.String())
		}
	}

	deleted := serveMockJSON(t, router, http.MethodDelete,
		"/api/v1/github/mock/workspace-connections/workspace-1?host="+defaultGitHubHost, "")
	if deleted.Code != http.StatusOK {
		t.Fatalf("delete github.com connection: %d %s", deleted.Code, deleted.Body.String())
	}
	invalid := serveMockJSON(t, router, http.MethodPut,
		"/api/v1/github/mock/workspace-connections/workspace-1?host=bad/host",
		`{"source":"gh_cli","status":"active","login":"alice"}`)

	connections, err := store.ListWorkspaceConnections(context.Background(), "workspace-1")
	if err != nil || len(connections) != 1 || connections[0].GitHubHost != enterpriseTestHost {
		t.Fatalf("connections = %+v, err %v", connections, err)
	}
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid host status = %d", invalid.Code)
	}
}
