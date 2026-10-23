package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/require"

	"github.com/kandev/kandev/internal/github"
)

const enterpriseTriggerHost = "octo.ghe.com"

type hostTriggerSecrets struct{}

func (hostTriggerSecrets) List(context.Context) ([]*github.SecretListItem, error) { return nil, nil }
func (hostTriggerSecrets) Reveal(_ context.Context, key string) (string, error) {
	for _, host := range []string{"github.com", enterpriseTriggerHost} {
		if key == github.WorkspacePATSecretKey("ws-1", host) {
			return "token-" + host, nil
		}
	}
	return "", fmt.Errorf("unexpected fixture credential key: %s", key)
}

// hostTriggerTransport answers PR searches and records the API host of each.
type hostTriggerTransport struct {
	hosts []string
}

func (r *hostTriggerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.hosts = append(r.hosts, req.URL.Host)
	web := strings.TrimPrefix(req.URL.Host, "api.")
	body := fmt.Sprintf(`{"items":[{"id":17,"number":17,"title":"Host fixture","state":"open","html_url":"https://%s/acme/widget/pull/17","repository_url":"https://%s/repos/acme/widget","user":{"login":"alice"}}]}`, web, req.URL.Host)
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

func newHostTriggerFixture(t *testing.T) (*Service, *GitHubEvaluator, *Automation, *hostTriggerTransport) {
	t.Helper()
	svc := newTestService(t)
	t.Cleanup(svc.eventBus.Close)
	db, err := sqlx.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	store, err := github.NewStore(db, db)
	require.NoError(t, err)
	for _, host := range []string{"github.com", enterpriseTriggerHost} {
		require.NoError(t, store.UpsertWorkspaceConnection(context.Background(), &github.WorkspaceConnection{
			WorkspaceID: "ws-1", Source: github.ConnectionSourcePAT, GitHubHost: host,
			Login: "fixture-owner", Status: github.ConnectionStatusActive, CredentialGeneration: 1,
		}))
	}
	transport := &hostTriggerTransport{}
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
	githubService := github.NewService(nil, github.AuthMethodPAT, hostTriggerSecrets{}, store, svc.eventBus, svc.logger)
	t.Cleanup(githubService.Stop)
	return svc, NewGitHubEvaluator(svc, githubService, svc.logger), createTestAutomation(t, svc, "ws-1"), transport
}

func TestGitHubPRTriggerHostsPollTheRepositoryHost(t *testing.T) {
	cases := []struct {
		name, repos, apiHost, dedupKey string
	}{
		{"enterprise host", `[{"host":"` + enterpriseTriggerHost + `","owner":"acme","name":"widget"}]`,
			"api." + enterpriseTriggerHost, "pr:" + enterpriseTriggerHost + "/acme/widget#17"},
		{"no host", `[{"owner":"acme","name":"widget"}]`, "api.github.com", "pr:acme/widget#17"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc, evaluator, automation, transport := newHostTriggerFixture(t)
			_, err := svc.AddTrigger(context.Background(), &AddTriggerRequest{
				AutomationID: automation.ID, Type: TriggerTypeGitHubPR, Enabled: true,
				Config: json.RawMessage(`{"events":["opened"],"repos":` + tc.repos + `}`),
			})
			require.NoError(t, err)

			evaluator.evaluatePRTriggers(context.Background())

			require.Equal(t, []string{tc.apiHost}, transport.hosts)
			reserved, err := svc.Store().HasRunWithDedupKey(context.Background(), automation.ID, tc.dedupKey)
			require.NoError(t, err)
			require.True(t, reserved, "dedup key %s", tc.dedupKey)
		})
	}
}
