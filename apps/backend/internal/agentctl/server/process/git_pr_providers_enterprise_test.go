package process

import (
	"testing"

	"github.com/kandev/kandev/internal/agentctl/server/config"
	"github.com/kandev/kandev/internal/common/logger"
	"github.com/kandev/kandev/internal/githubauth"
)

func enterprisePRProviderOperator(t *testing.T, agentEnv ...string) *GitOperator {
	t.Helper()
	log, _ := logger.NewLogger(logger.LoggingConfig{Level: "error", Format: "json"})
	return NewManager(&config.InstanceConfig{WorkDir: t.TempDir(), AgentEnv: agentEnv}, log).GitOperator()
}

func TestEnterprisePRProviderTreatsListedHostsAsGitHub(t *testing.T) {
	op := enterprisePRProviderOperator(t, gitHubHostsEnv+"=octo.ghe.com,ghe.example.com")
	for _, remote := range []string{
		"https://octo.ghe.com/acme/widgets.git",
		"octo@octo.ghe.com:acme/widgets.git",
		"git@ghe.example.com:acme/widgets.git",
		"ssh://git@ghe.example.com/acme/widgets.git",
	} {
		if got := op.detectPRProvider(remote); got != prProviderGitHub {
			t.Errorf("detectPRProvider(%q) = %q, want github", remote, got)
		}
	}
}

func TestEnterprisePRProviderLeavesUnlistedHostsUnsupported(t *testing.T) {
	tests := []struct {
		name string
		env  []string
	}{
		{name: "no variable"},
		{name: "another host listed", env: []string{gitHubHostsEnv + "=ghe.example.com"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			op := enterprisePRProviderOperator(t, test.env...)
			if got := op.detectPRProvider("https://octo.ghe.com/acme/widgets.git"); got != "" {
				t.Fatalf("detectPRProvider() = %q, want unsupported", got)
			}
		})
	}
}

func TestEnterprisePRProviderPackageDetectionReadsProcessEnvironment(t *testing.T) {
	t.Setenv(gitHubHostsEnv, "octo.ghe.com")
	if got := detectPRProvider("https://octo.ghe.com/acme/widgets.git"); got != prProviderGitHub {
		t.Fatalf("detectPRProvider() = %q, want github", got)
	}
}

func TestEnterprisePRProviderVariableMatchesTheExecutorContract(t *testing.T) {
	if gitHubHostsEnv != githubauth.GitHubHostsEnv {
		t.Fatalf("gitHubHostsEnv = %q, want %q", gitHubHostsEnv, githubauth.GitHubHostsEnv)
	}
}
