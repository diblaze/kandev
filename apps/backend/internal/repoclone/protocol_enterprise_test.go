package repoclone

import "testing"

func TestCloneURLWithHostEnterpriseGitHub(t *testing.T) {
	for _, tc := range []struct {
		name, host, protocol, want string
	}{
		{"github.com ssh", "https://github.com", ProtocolSSH, "git@github.com:org/repo.git"},
		{"ghe.com tenant ssh", "https://octo.ghe.com", ProtocolSSH, "octo@octo.ghe.com:org/repo.git"},
		{"server ssh", "https://ghe.example.com", ProtocolSSH, "git@ghe.example.com:org/repo.git"},
		{"github.com https", "https://github.com", ProtocolHTTPS, "https://github.com/org/repo.git"},
		{"ghe.com tenant https", "https://octo.ghe.com", ProtocolHTTPS, "https://octo.ghe.com/org/repo.git"},
		{"server https", "https://ghe.example.com", ProtocolHTTPS, "https://ghe.example.com/org/repo.git"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := CloneURLWithHost("github", tc.host, "org", "repo", tc.protocol)
			if err != nil || got != tc.want {
				t.Fatalf("CloneURLWithHost = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}

func TestGitHubHTTPSCloneURLEnterpriseHost(t *testing.T) {
	for _, tc := range []struct {
		name, cloneURL, want string
	}{
		{"https on host", "https://ghe.example.com/org/repo.git", "https://ghe.example.com/org/repo.git"},
		{"ssh rewritten to host", "git@ghe.example.com:org/repo.git", "https://ghe.example.com/org/repo.git"},
		{"other host rewritten to provider host", "https://github.com/org/repo.git", "https://ghe.example.com/org/repo.git"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := githubHTTPSCloneURL(tc.cloneURL, "https://ghe.example.com", "org", "repo")
			if err != nil || got != tc.want {
				t.Fatalf("githubHTTPSCloneURL = %q, %v; want %q", got, err, tc.want)
			}
		})
	}
}
