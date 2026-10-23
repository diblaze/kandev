package gitbase

import (
	"context"
	"testing"

	"github.com/kandev/kandev/internal/task/models"
)

const enterpriseHost = "octo.ghe.com"

func enterprisePRBase() models.PRBase {
	base := testPRBase()
	base.Target.HeadRepository.Host = enterpriseHost
	base.Target.HeadRepository.RemoteURL = "https://" + enterpriseHost + "/fork/widget.git"
	base.Target.TargetRepository.Host = enterpriseHost
	base.Target.TargetRepository.RemoteURL = "https://" + enterpriseHost + "/upstream/widget.git"
	return base
}

func TestMaterializeEnterpriseBaseUsesCheckoutSSHTransport(t *testing.T) {
	for origin, want := range map[string]string{
		"octo@octo.ghe.com:fork/widget.git":       "octo@octo.ghe.com:upstream/widget.git",
		"ssh://octo@octo.ghe.com/fork/widget.git": "ssh://octo@octo.ghe.com/upstream/widget.git",
	} {
		t.Run(origin, func(t *testing.T) {
			base := enterprisePRBase()
			remoteName := base.Target.ComparisonRemoteName()
			fake := &gitFake{outputs: map[string]string{
				"config --get remote.origin.url":                                  origin,
				"rev-parse --verify " + base.Target.ComparisonRef() + "^{commit}": "0123456789abcdef0123456789abcdef01234567",
			}, errors: map[string]error{}}

			if _, err := Materialize(context.Background(), fake.run, base); err != nil {
				t.Fatalf("Materialize(): %v", err)
			}
			if !hasCommand(fake.commands, "remote", "add", "--no-tags", remoteName, want) {
				t.Fatalf("commands = %#v; want SSH comparison remote %q", fake.commands, want)
			}
		})
	}
}

func TestMaterializeEnterpriseBaseIgnoresOriginOnAnotherHost(t *testing.T) {
	base := enterprisePRBase()
	remoteName := base.Target.ComparisonRemoteName()
	fake := &gitFake{outputs: map[string]string{
		"config --get remote.origin.url":                                  "octo@ghe.example.com:fork/widget.git",
		"rev-parse --verify " + base.Target.ComparisonRef() + "^{commit}": "0123456789abcdef0123456789abcdef01234567",
	}, errors: map[string]error{}}

	if _, err := Materialize(context.Background(), fake.run, base); err != nil {
		t.Fatalf("Materialize(): %v", err)
	}
	if !hasCommand(fake.commands, "remote", "add", "--no-tags", remoteName, base.Target.TargetRepository.RemoteURL) {
		t.Fatalf("commands = %#v; want the HTTPS target remote", fake.commands)
	}
}

func TestMaterializeEnterpriseBaseUpdatesSameRepositoryRemote(t *testing.T) {
	base := enterprisePRBase()
	remoteName := base.Target.ComparisonRemoteName()
	fake := &gitFake{outputs: map[string]string{
		"config --get remote.origin.url":                                  "octo@octo.ghe.com:fork/widget.git",
		"config --get remote." + remoteName + ".url":                      base.Target.TargetRepository.RemoteURL,
		"rev-parse --verify " + base.Target.ComparisonRef() + "^{commit}": "0123456789abcdef0123456789abcdef01234567",
	}, errors: map[string]error{}}

	if _, err := Materialize(context.Background(), fake.run, base); err != nil {
		t.Fatalf("Materialize(): %v", err)
	}
	if !hasCommand(fake.commands, "remote", "set-url", remoteName, "octo@octo.ghe.com:upstream/widget.git") {
		t.Fatalf("commands = %#v; want same-identity comparison remote updated to SSH", fake.commands)
	}
}
