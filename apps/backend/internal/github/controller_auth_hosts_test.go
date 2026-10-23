package github

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTwoHostControllerRouter(t *testing.T) (*gin.Engine, *Service) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	service := newTwoHostStatusService(t)
	router := gin.New()
	NewController(service, newControllerTestLogger()).RegisterHTTPRoutes(router)
	return router, service
}

func TestControllerStatusProjectsHostQuery(t *testing.T) {
	router, _ := newTwoHostControllerRouter(t)

	response := doControllerJSON(t, router, http.MethodGet,
		"/api/v1/github/status?workspace_id=ws-1&host="+enterpriseTestHost, nil)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body)
	}
	var body struct {
		Automation struct {
			GitHubHost string `json:"github_host"`
		} `json:"automation"`
		Hosts []WorkspaceHostStatus `json:"hosts"`
	}
	decodeControllerBody(t, response, &body)
	if body.Automation.GitHubHost != enterpriseTestHost || len(body.Hosts) != 2 {
		t.Fatalf("status body = %+v", body)
	}
}

func TestControllerDeleteWorkspaceConnectionRemovesOnlyQueriedHost(t *testing.T) {
	router, service := newTwoHostControllerRouter(t)

	response := doControllerJSON(t, router, http.MethodDelete,
		"/api/v1/github/workspace-connection?workspace_id=ws-1&host="+enterpriseTestHost, nil)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d; body = %s", response.Code, response.Body)
	}
	hosts, err := service.WorkspaceGitHubHosts(t.Context(), "ws-1")
	if err != nil || len(hosts) != 1 || hosts[0] != defaultGitHubHost {
		t.Fatalf("hosts after delete = %v, err %v", hosts, err)
	}
}

func TestWriteGitHubAuthErrorNamesUnconnectedHost(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest(http.MethodGet, "/api/v1/github/status", nil)

	writeGitHubAuthError(ctx, fmt.Errorf("resolve: %w", &HostNotConnectedError{Host: "ghe.example.com"}))

	var body struct {
		Code string `json:"code"`
		Host string `json:"host"`
	}
	decodeControllerBody(t, response, &body)
	notConfiguredStatus, _ := githubAuthErrorResponse(ErrGitHubNotConfigured)
	if response.Code != notConfiguredStatus || body.Code != githubNotConfiguredCode || body.Host != "ghe.example.com" {
		t.Fatalf("response %d %+v", response.Code, body)
	}
}

func TestSelectedCLIAccountMatchesEachHost(t *testing.T) {
	hosts := []WorkspaceHostStatus{
		{GitHubHost: defaultGitHubHost, Source: ConnectionSourceGHCLI, Login: "octocat"},
		{GitHubHost: enterpriseTestHost, Source: ConnectionSourceGHCLI, Login: "hubot"},
	}
	for _, account := range []GHAccount{
		{Host: defaultGitHubHost, Login: "octocat"}, {Host: enterpriseTestHost, Login: "hubot"},
	} {
		if !selectedCLIAccount(account, hosts) {
			t.Fatalf("account %+v is not selected", account)
		}
	}
	if selectedCLIAccount(GHAccount{Host: enterpriseTestHost, Login: "octocat"}, hosts) {
		t.Fatal("account of another host is selected")
	}
}
