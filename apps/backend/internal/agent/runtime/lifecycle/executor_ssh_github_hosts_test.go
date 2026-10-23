package lifecycle

import (
	"strconv"
	"strings"
	"testing"
)

func sshGitConfigHelpers(env map[string]string) map[string]string {
	helpers := map[string]string{}
	count, _ := strconv.Atoi(env["GIT_CONFIG_COUNT"])
	for index := range count {
		key := env["GIT_CONFIG_KEY_"+strconv.Itoa(index)]
		if strings.HasPrefix(key, "credential.") {
			helpers[key] = env["GIT_CONFIG_VALUE_"+strconv.Itoa(index)]
		}
	}
	return helpers
}

func TestSSHGitHubHostsForwardHostListAndEnterpriseTokens(t *testing.T) {
	got := sshRemoteAgentEnv(&ExecutorCreateRequest{Env: map[string]string{
		"KANDEV_GITHUB_HOSTS": "octo.ghe.com,ghe.example.com",
		"GH_ENTERPRISE_TOKEN": "server-token", "GITHUB_ENTERPRISE_TOKEN": "server-fallback",
		"GH_HOST": "octo.ghe.com",
	}})

	for key, want := range map[string]string{
		"KANDEV_GITHUB_HOSTS": "octo.ghe.com,ghe.example.com", "GH_HOST": "octo.ghe.com",
		"GH_ENTERPRISE_TOKEN": "server-token", "GITHUB_ENTERPRISE_TOKEN": "server-fallback",
	} {
		if got[key] != want {
			t.Errorf("remote %s = %q, want %q", key, got[key], want)
		}
	}
}

func TestSSHGitHubHostsConfigureEachHostHelperFromItsTokenVariable(t *testing.T) {
	tests := []struct {
		name   string
		ghHost string
		want   map[string]string
	}{
		{name: "unset GH_HOST binds GH_TOKEN to github.com", want: map[string]string{
			"credential.https://github.com.helper":      tenantContainerHelper,
			"credential.https://ghe.example.com.helper": serverContainerHelper,
		}},
		{name: "tenant GH_HOST binds GH_TOKEN to the tenant only", ghHost: "octo.ghe.com", want: map[string]string{
			"credential.https://octo.ghe.com.helper":    tenantContainerHelper,
			"credential.https://ghe.example.com.helper": serverContainerHelper,
		}},
		{name: "malformed GH_HOST binds no token", ghHost: "https://octo.ghe.com", want: map[string]string{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			env := map[string]string{
				"GH_TOKEN": "public-token", "GH_ENTERPRISE_TOKEN": "server-token",
				"KANDEV_GITHUB_HOSTS": "octo.ghe.com,other.ghe.com,ghe.example.com",
			}
			if test.ghHost != "" {
				env["GH_HOST"] = test.ghHost
			}
			got := sshRemoteContributionEnv(&ExecutorCreateRequest{Env: env}, "/home/agent/.kandev/bin/agentctl")

			helpers := sshGitConfigHelpers(got)
			for key, value := range test.want {
				if helpers[key] != value {
					t.Errorf("%s = %q, want %q", key, helpers[key], value)
				}
			}
			if len(helpers) != len(test.want) {
				t.Fatalf("remote helpers = %#v, want only %#v", helpers, test.want)
			}
		})
	}
}

func TestSelectedPullRequestCheckoutScrubsEnterpriseTokens(t *testing.T) {
	metadata := map[string]interface{}{metadataCheckoutRef: "refs/pull/7/head"}
	env := selectedCheckoutAgentEnv(map[string]string{
		"GH_ENTERPRISE_TOKEN": "server-token", "GITHUB_ENTERPRISE_TOKEN": "server-fallback",
	}, metadata)

	for _, key := range []string{"GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN"} {
		if _, found := env[key]; found {
			t.Errorf("agent env kept %s for a pull-request checkout", key)
		}
		if !strings.Contains(selectedCheckoutCredentialScrubScript(metadata), " "+key) {
			t.Errorf("scrub script does not unset %s", key)
		}
	}
}
