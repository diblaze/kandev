package lifecycle

import (
	"strconv"
	"strings"
	"testing"

	"github.com/kandev/kandev/internal/agent/agents"
	"github.com/kandev/kandev/internal/gitconfigenv"
)

const (
	tenantContainerHelper = `!f() { echo "username=x-access-token"; echo "password=${GH_TOKEN:-${GITHUB_TOKEN}}"; }; f`
	serverContainerHelper = `!f() { echo "username=x-access-token"; echo "password=${GH_ENTERPRISE_TOKEN:-${GITHUB_ENTERPRISE_TOKEN}}"; }; f`
)

func containerGitConfigEntries(t *testing.T, credentials map[string]string) []gitconfigenv.Entry {
	t.Helper()
	env, err := (&ContainerManager{}).buildEnvVars(ContainerConfig{AgentConfig: agents.NewMockAgent(), Credentials: credentials})
	if err != nil {
		t.Fatalf("buildEnvVars() error = %v", err)
	}
	values := make(map[string]string, len(env))
	for _, entry := range env {
		if key, value, ok := strings.Cut(entry, "="); ok && strings.HasPrefix(key, "GIT_CONFIG_") {
			values[key] = value
		}
	}
	count, err := strconv.Atoi(values["GIT_CONFIG_COUNT"])
	if err != nil {
		t.Fatalf("container GIT_CONFIG_COUNT = %q: %v", values["GIT_CONFIG_COUNT"], err)
	}
	entries := make([]gitconfigenv.Entry, 0, count)
	for index := range count {
		suffix := strconv.Itoa(index)
		entries = append(entries, gitconfigenv.Entry{
			Key: values["GIT_CONFIG_KEY_"+suffix], Value: values["GIT_CONFIG_VALUE_"+suffix],
		})
	}
	return entries
}

func hasContainerGitConfigEntry(entries []gitconfigenv.Entry, key, value string) bool {
	for _, entry := range entries {
		if entry.Key == key && entry.Value == value {
			return true
		}
	}
	return false
}

func TestContainerGitHubHostsConfigureListedHostsWithTheirTokenVariable(t *testing.T) {
	entries := containerGitConfigEntries(t, map[string]string{
		"GH_TOKEN": "tenant-token", "GH_HOST": "octo.ghe.com", "GH_ENTERPRISE_TOKEN": "server-token",
		"KANDEV_GITHUB_HOSTS": "octo.ghe.com,other.ghe.com,ghe.example.com",
	})

	for _, want := range []gitconfigenv.Entry{
		{Key: "credential.https://octo.ghe.com.helper", Value: tenantContainerHelper},
		{Key: "url.https://octo.ghe.com/.insteadOf", Value: "git@octo.ghe.com:"},
		{Key: "url.https://octo.ghe.com/.insteadOf", Value: "octo@octo.ghe.com:"},
		{Key: "url.https://octo.ghe.com/.insteadOf", Value: "ssh://git@octo.ghe.com/"},
		{Key: "credential.https://ghe.example.com.helper", Value: serverContainerHelper},
		{Key: "url.https://ghe.example.com/.insteadOf", Value: "git@ghe.example.com:"},
		{Key: "url.https://ghe.example.com/.insteadOf", Value: "ssh://git@ghe.example.com/"},
	} {
		if !hasContainerGitConfigEntry(entries, want.Key, want.Value) {
			t.Errorf("container Git config is missing %s=%s: %#v", want.Key, want.Value, entries)
		}
	}
	for _, entry := range entries {
		if entry.Key == "credential.https://github.com.helper" || strings.Contains(entry.Key, "other.ghe.com") {
			t.Errorf("a GH_TOKEN bound to octo.ghe.com configured another host: %s=%s", entry.Key, entry.Value)
		}
	}
}

func TestContainerGitHubHostsKeepGitHubDotComTokenOffTenants(t *testing.T) {
	entries := containerGitConfigEntries(t, map[string]string{
		"GH_TOKEN": "public-token", "KANDEV_GITHUB_HOSTS": "octo.ghe.com",
	})

	if !hasContainerGitConfigEntry(entries, "credential.https://github.com.helper", tenantContainerHelper) {
		t.Fatalf("github.com helper missing: %#v", entries)
	}
	for _, entry := range entries {
		if strings.Contains(entry.Key, "octo.ghe.com") {
			t.Fatalf("a GH_TOKEN without GH_HOST configured a tenant: %#v", entries)
		}
	}
}

func TestContainerGitHubHostsMalformedGHHostConfiguresNoTokenHelper(t *testing.T) {
	entries := containerGitConfigEntries(t, map[string]string{
		"GH_TOKEN": "public-token", "GH_ENTERPRISE_TOKEN": "server-token", "GH_HOST": "https://octo.ghe.com",
		"KANDEV_GITHUB_HOSTS": "octo.ghe.com,ghe.example.com",
	})

	for _, entry := range entries {
		if strings.HasPrefix(entry.Key, "credential.") {
			t.Fatalf("a malformed GH_HOST configured a token helper: %#v", entries)
		}
	}
}

func TestContainerGitHubHostsSkipHostsWithoutTheirToken(t *testing.T) {
	entries := containerGitConfigEntries(t, map[string]string{
		"GH_TOKEN": "public-token", "KANDEV_GITHUB_HOSTS": "ghe.example.com",
	})

	for _, entry := range entries {
		if strings.Contains(entry.Key, "ghe.example.com") {
			t.Fatalf("GH_TOKEN-only container configured the enterprise server host: %#v", entries)
		}
	}
	if !hasContainerGitConfigEntry(entries, "credential.https://github.com.helper", tenantContainerHelper) {
		t.Fatalf("github.com helper missing: %#v", entries)
	}
}

func TestContainerGitHubHostsEnterpriseTokenDoesNotConfigureGitHubDotCom(t *testing.T) {
	entries := containerGitConfigEntries(t, map[string]string{
		"GH_ENTERPRISE_TOKEN": "server-token", "KANDEV_GITHUB_HOSTS": "ghe.example.com",
	})

	for _, entry := range entries {
		if entry.Key == "credential.https://github.com.helper" {
			t.Fatalf("an enterprise-only container configured github.com: %#v", entries)
		}
	}
	if !hasContainerGitConfigEntry(entries, "credential.https://ghe.example.com.helper", serverContainerHelper) {
		t.Fatalf("ghe.example.com helper missing: %#v", entries)
	}
}
