package lifecycle

import "testing"

func TestCloneURLGitHubHostsInjectOnlyTheHostTokenVariable(t *testing.T) {
	listed := "octo.ghe.com,ghe.example.com"
	tests := []struct {
		name string
		url  string
		env  map[string]string
		want string
	}{
		{name: "tenant https", url: "https://octo.ghe.com/acme/widget.git",
			env:  map[string]string{"GH_TOKEN": "tenant-token", "GH_HOST": "octo.ghe.com", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://x-access-token:tenant-token@octo.ghe.com/acme/widget.git"},
		{name: "server scp", url: "git@ghe.example.com:acme/widget.git",
			env:  map[string]string{"GH_ENTERPRISE_TOKEN": "server-token", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://x-access-token:server-token@ghe.example.com/acme/widget.git"},
		{name: "server ssh scheme", url: "ssh://git@ghe.example.com/acme/widget.git",
			env:  map[string]string{"GITHUB_ENTERPRISE_TOKEN": "server-token", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://x-access-token:server-token@ghe.example.com/acme/widget.git"},
		{name: "public token never reaches the server", url: "https://ghe.example.com/acme/widget.git",
			env:  map[string]string{"GH_TOKEN": "public-token", "GITHUB_TOKEN": "public-token", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://ghe.example.com/acme/widget.git"},
		{name: "enterprise token never reaches github.com", url: "https://github.com/acme/widget.git",
			env:  map[string]string{"GH_ENTERPRISE_TOKEN": "server-token", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://github.com/acme/widget.git"},
		{name: "enterprise token never reaches the tenant", url: "https://octo.ghe.com/acme/widget.git",
			env:  map[string]string{"GH_ENTERPRISE_TOKEN": "server-token", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://octo.ghe.com/acme/widget.git"},
		{name: "unlisted host is left alone", url: "https://octo.ghe.com/acme/widget.git",
			env:  map[string]string{"GH_TOKEN": "public-token"},
			want: "https://octo.ghe.com/acme/widget.git"},
		{name: "github.com-bound token never reaches a listed tenant", url: "https://octo.ghe.com/acme/widget.git",
			env:  map[string]string{"GH_TOKEN": "public-token", "GITHUB_TOKEN": "public-token", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://octo.ghe.com/acme/widget.git"},
		{name: "tenant-bound token reaches the tenant", url: "octo@octo.ghe.com:acme/widget.git",
			env:  map[string]string{"GH_TOKEN": "tenant-token", "GH_HOST": "octo.ghe.com", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://x-access-token:tenant-token@octo.ghe.com/acme/widget.git"},
		{name: "tenant-bound token never reaches github.com", url: "https://github.com/acme/widget.git",
			env:  map[string]string{"GH_TOKEN": "tenant-token", "GH_HOST": "octo.ghe.com", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://github.com/acme/widget.git"},
		{name: "tenant-bound token never reaches another tenant", url: "https://other.ghe.com/acme/widget.git",
			env:  map[string]string{"GH_TOKEN": "tenant-token", "GH_HOST": "octo.ghe.com", "KANDEV_GITHUB_HOSTS": "octo.ghe.com,other.ghe.com"},
			want: "https://other.ghe.com/acme/widget.git"},
		{name: "malformed GH_HOST injects no token", url: "https://github.com/acme/widget.git",
			env:  map[string]string{"GH_TOKEN": "public-token", "GH_HOST": "https://github.com"},
			want: "https://github.com/acme/widget.git"},
		{name: "lookalike host is left alone", url: "https://octo.ghe.com.evil.example/acme/widget.git",
			env:  map[string]string{"GH_TOKEN": "public-token", "KANDEV_GITHUB_HOSTS": listed},
			want: "https://octo.ghe.com.evil.example/acme/widget.git"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := injectGitHubTokenIntoCloneURL(test.url, test.env); got != test.want {
				t.Fatalf("injectGitHubTokenIntoCloneURL(%q) = %q, want %q", test.url, got, test.want)
			}
		})
	}
}
