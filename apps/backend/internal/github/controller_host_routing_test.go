package github

import (
	"net/http"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

type hostRoutedRoute struct {
	method, path string
	body         map[string]any
}

func hostRoutedRoutes() []hostRoutedRoute {
	const pr = "/api/v1/github/prs/octo/app/1"
	const repo = "/api/v1/github/repos/octo/app"
	return []hostRoutedRoute{
		{http.MethodGet, pr, nil},
		{http.MethodGet, pr + "/info", nil},
		{http.MethodGet, pr + "/status", nil},
		{http.MethodPost, pr + "/reviews", map[string]any{"event": "COMMENT", "body": "ok"}},
		{http.MethodPost, pr + "/requested-reviewers", map[string]any{"reviewers": []string{"hubot"}}},
		{http.MethodPut, pr + "/merge", map[string]any{"merge_method": "merge"}},
		{http.MethodGet, "/api/v1/github/issues/octo/app/1/info", nil},
		{http.MethodGet, "/api/v1/github/orgs", nil},
		{http.MethodGet, "/api/v1/github/repos", nil},
		{http.MethodGet, "/api/v1/github/repos/search?org=octo", nil},
		{http.MethodGet, repo + "/branches", nil},
		{http.MethodGet, repo + "/merge-methods", nil},
		{http.MethodGet, repo + "/merge-access", nil},
		{http.MethodGet, "/api/v1/github/user/prs", nil},
		{http.MethodGet, "/api/v1/github/user/issues", nil},
	}
}

func newHostRoutingRouter(t *testing.T) (*gin.Engine, hostRoutingFixture) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	fixture := newHostRoutingFixture(t)
	router := gin.New()
	NewController(fixture.service, newControllerTestLogger()).RegisterHTTPRoutes(router)
	return router, fixture
}

func withQuery(path, query string) string {
	if strings.Contains(path, "?") {
		return path + "&" + query
	}
	return path + "?" + query
}

func assertLoadedOnly(t *testing.T, fixture hostRoutingFixture, want string) {
	t.Helper()
	loaded := fixture.connections.loadedHosts()
	if len(loaded) == 0 {
		t.Fatal("route resolved no credential")
	}
	for _, host := range loaded {
		if host != want {
			t.Fatalf("resolved hosts = %v, want only %s", loaded, want)
		}
	}
}

func TestControllerHostRoutingPassesHostQuery(t *testing.T) {
	for _, route := range hostRoutedRoutes() {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			router, fixture := newHostRoutingRouter(t)

			doControllerJSON(t, router, route.method,
				withQuery(route.path, "workspace_id="+testWorkspaceID+"&host="+enterpriseTestHost), route.body)

			assertLoadedOnly(t, fixture, enterpriseTestHost)
		})
	}
}

func TestControllerHostRoutingDefaultsMissingHostToGitHubDotCom(t *testing.T) {
	for _, route := range hostRoutedRoutes() {
		t.Run(route.method+" "+route.path, func(t *testing.T) {
			router, fixture := newHostRoutingRouter(t)

			doControllerJSON(t, router, route.method, withQuery(route.path, "workspace_id="+testWorkspaceID), route.body)

			assertLoadedOnly(t, fixture, defaultGitHubHost)
		})
	}
}

func TestControllerHostRoutingBatchStatusReadsBodyHost(t *testing.T) {
	router, fixture := newHostRoutingRouter(t)

	doControllerJSON(t, router, http.MethodPost, "/api/v1/github/prs/statuses", map[string]any{
		"workspace_id": testWorkspaceID, "host": enterpriseTestHost,
		"refs": []map[string]any{{"owner": "octo", "repo": "app", "number": 1}},
	})

	assertLoadedOnly(t, fixture, enterpriseTestHost)
}

func TestControllerHostRoutingNamesUnconnectedHost(t *testing.T) {
	router, _ := newHostRoutingRouter(t)

	response := doControllerJSON(t, router, http.MethodGet,
		"/api/v1/github/prs/octo/app/1/info?workspace_id="+testWorkspaceID+"&host="+unconnectedTestHost, nil)

	var body struct {
		Code string `json:"code"`
		Host string `json:"host"`
	}
	decodeControllerBody(t, response, &body)
	if body.Code != githubNotConfiguredCode || body.Host != unconnectedTestHost {
		t.Fatalf("status %d body = %s", response.Code, response.Body)
	}
}
