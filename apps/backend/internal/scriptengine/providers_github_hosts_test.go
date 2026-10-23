package scriptengine

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	tenantHelperLine = `git config --global --replace-all credential.https://octo.ghe.com.helper ` +
		`'!/bin/sh -c "echo username=x-access-token; echo password=${GH_TOKEN:-${GITHUB_TOKEN}}"'`
	serverHelperLine = `git config --global --replace-all credential.https://ghe.example.com.helper ` +
		`'!/bin/sh -c "echo username=x-access-token; echo password=${GH_ENTERPRISE_TOKEN:-${GITHUB_ENTERPRISE_TOKEN}}"'`
	publicHelperKey = "credential.https://github.com.helper"
)

func gitHubHostsAuthSetup(env map[string]string) string {
	return GitHubAuthProvider(env)()["github.auth_setup"]
}

func TestGitHubAuthHostsConfigureEachListedHostWithItsTokenVariable(t *testing.T) {
	setup := gitHubHostsAuthSetup(map[string]string{
		"GH_TOKEN": "tenant-token", "GH_HOST": "octo.ghe.com", "GH_ENTERPRISE_TOKEN": "server-token",
		"KANDEV_GITHUB_HOSTS": "octo.ghe.com,other.ghe.com,ghe.example.com",
	})

	for _, unwanted := range []string{publicHelperKey, "credential.https://other.ghe.com.helper"} {
		if strings.Contains(setup, unwanted) {
			t.Errorf("a GH_TOKEN bound to octo.ghe.com configured %s:\n%s", unwanted, setup)
		}
	}
	for _, want := range []string{
		tenantHelperLine, serverHelperLine,
		"gh config set git_protocol https --host github.com 2>/dev/null || true",
		"gh config set git_protocol https --host octo.ghe.com 2>/dev/null || true",
		"gh config set git_protocol https --host ghe.example.com 2>/dev/null || true",
	} {
		if !strings.Contains(setup, want) {
			t.Errorf("auth setup is missing %q:\n%s", want, setup)
		}
	}
	if strings.Contains(setup, "tenant-token") || strings.Contains(setup, "server-token") {
		t.Fatalf("token value appears literally in the script:\n%s", setup)
	}
}

func TestGitHubAuthHostsConfigureOnlyHostsWhoseTokenIsSet(t *testing.T) {
	serverOnly := gitHubHostsAuthSetup(map[string]string{
		"GH_ENTERPRISE_TOKEN": "server-token", "KANDEV_GITHUB_HOSTS": "ghe.example.com",
	})
	if !strings.Contains(serverOnly, serverHelperLine) || strings.Contains(serverOnly, publicHelperKey) {
		t.Fatalf("enterprise-only setup should configure only ghe.example.com:\n%s", serverOnly)
	}

	publicOnly := gitHubHostsAuthSetup(map[string]string{
		"GH_TOKEN": "public-token", "KANDEV_GITHUB_HOSTS": "ghe.example.com",
	})
	if strings.Contains(publicOnly, "credential.https://ghe.example.com.helper") {
		t.Fatalf("a GH_TOKEN-only setup configured the enterprise server host:\n%s", publicOnly)
	}
}

func TestGitHubAuthHostsKeepAnUnboundGitHubTokenOnGitHubDotCom(t *testing.T) {
	setup := gitHubHostsAuthSetup(map[string]string{"GH_TOKEN": "public-token", "KANDEV_GITHUB_HOSTS": "octo.ghe.com"})
	if !strings.Contains(setup, publicHelperKey) || strings.Contains(setup, "credential.https://octo.ghe.com.helper") {
		t.Fatalf("a GH_TOKEN without GH_HOST should configure only github.com:\n%s", setup)
	}
}

func TestGitHubAuthHostsMalformedGHHostConfiguresNoTokenHelper(t *testing.T) {
	setup := gitHubHostsAuthSetup(map[string]string{
		"GH_TOKEN": "public-token", "GH_ENTERPRISE_TOKEN": "server-token", "GH_HOST": "octo.ghe.com/acme",
		"KANDEV_GITHUB_HOSTS": "octo.ghe.com,ghe.example.com",
	})
	if strings.Contains(setup, "git config --global --replace-all credential.") {
		t.Fatalf("a malformed GH_HOST configured a token helper:\n%s", setup)
	}
}

func TestGitHubAuthHostsIgnoreUnsafeListedHosts(t *testing.T) {
	setup := gitHubHostsAuthSetup(map[string]string{
		"GH_TOKEN": "public-token", "KANDEV_GITHUB_HOSTS": "evil.example.com;touch /tmp/x,$(id).example.com",
	})
	if strings.Contains(setup, "evil") || strings.Contains(setup, "$(id)") {
		t.Fatalf("unsafe host reached the script:\n%s", setup)
	}
}

// TestGitHubAuthHostsGitReceivesOnlyTheHostTokenVariable runs the setup against
// a real git and asks git for each host's credential, so a token can only be
// read back on the host it is bound to.
func TestGitHubAuthHostsGitReceivesOnlyTheHostTokenVariable(t *testing.T) {
	for _, tool := range []string{"git", "sh"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s not available", tool)
		}
	}
	home := t.TempDir()
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "gh"), []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake gh: %v", err)
	}
	env := []string{
		"HOME=" + home, "XDG_CONFIG_HOME=" + home, "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0",
		"GH_TOKEN=tenant-token", "GH_HOST=octo.ghe.com", "GH_ENTERPRISE_TOKEN=server-token",
		"PATH=" + bin + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	setup := gitHubHostsAuthSetup(map[string]string{
		"GH_TOKEN": "tenant-token", "GH_HOST": "octo.ghe.com", "GH_ENTERPRISE_TOKEN": "server-token",
		"KANDEV_GITHUB_HOSTS": "octo.ghe.com,other.ghe.com,ghe.example.com",
	})
	cmd := exec.Command("sh", "-c", setup)
	cmd.Env = env
	cmd.Dir = home
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("auth setup failed: %v: %s", err, out)
	}

	for host, want := range map[string]string{"octo.ghe.com": "tenant-token", "ghe.example.com": "server-token"} {
		fill := exec.Command("git", "credential", "fill")
		fill.Env = env
		fill.Dir = home
		fill.Stdin = strings.NewReader("protocol=https\nhost=" + host + "\n\n")
		out, err := fill.CombinedOutput()
		if err != nil {
			t.Fatalf("git credential fill for %s: %v: %s", host, err, out)
		}
		if !strings.Contains(string(out), "password="+want+"\n") {
			t.Fatalf("git credential for %s = %q, want password %s", host, out, want)
		}
	}
	for _, host := range []string{"github.com", "other.ghe.com", "other.example.com"} {
		unbound := exec.Command("git", "credential", "fill")
		unbound.Env = env
		unbound.Dir = home
		unbound.Stdin = strings.NewReader("protocol=https\nhost=" + host + "\n\n")
		if out, err := unbound.CombinedOutput(); err == nil {
			t.Fatalf("git returned a credential for %s, which no profile token is bound to: %s", host, out)
		}
	}
}
