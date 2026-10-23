package github

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/common/logger"
	ws "github.com/kandev/kandev/pkg/websocket"
)

func TestWSHandlersRouteOwnerRepoActionsToPayloadHost(t *testing.T) {
	type handler func(*Service, *logger.Logger) func(context.Context, *ws.Message) (*ws.Message, error)
	for name, tc := range map[string]struct {
		handler handler
		payload map[string]any
	}{
		"pr feedback":   {wsGetPRFeedback, map[string]any{"number": 1}},
		"pr files":      {wsGetPRFiles, map[string]any{"number": 1}},
		"pr commits":    {wsGetPRCommits, map[string]any{"number": 1}},
		"pr commit get": {wsGetPRCommitDetail, map[string]any{"sha": "4444444444444444444444444444444444444444"}},
	} {
		for _, host := range []string{enterpriseTestHost, ""} {
			t.Run(name+"/"+host, func(t *testing.T) {
				fixture := newHostRoutingFixture(t)
				payload := map[string]any{"workspace_id": testWorkspaceID, "owner": "octo", "repo": "app"}
				for key, value := range tc.payload {
					payload[key] = value
				}
				if host != "" {
					payload["host"] = host
				}
				message, err := ws.NewRequest("request-1", "github.test", payload)
				if err != nil {
					t.Fatalf("create request: %v", err)
				}

				if _, err := tc.handler(fixture.service, testLogger(t))(context.Background(), message); err != nil {
					t.Fatalf("handler error: %v", err)
				}

				assertLoadedOnly(t, fixture, githubHostOrDefault(host))
			})
		}
	}
}
