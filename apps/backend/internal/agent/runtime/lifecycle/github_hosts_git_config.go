package lifecycle

import (
	"fmt"
	"strconv"

	"github.com/kandev/kandev/internal/githubauth"
	"github.com/kandev/kandev/internal/scriptengine"
)

// githubTokenCredentialHelper answers Git for host with the profile token
// variables bound to it, read when Git asks, so no token is written into
// config. ok is false when env sets no token bound to host.
func githubTokenCredentialHelper(env map[string]string, host string) (helper string, ok bool) {
	primary, fallback := githubauth.ProfileTokenEnvForHost(env["GH_HOST"], host)
	if primary == "" || (env[primary] == "" && env[fallback] == "") {
		return "", false
	}
	return fmt.Sprintf(`!f() { echo "username=x-access-token"; echo "password=${%s:-${%s}}"; }; f`, primary, fallback), true
}

// containerGitConfig is the indexed Git config of a container: SSH-to-HTTPS
// rewrites for github.com, and a token helper for github.com and for every
// listed GitHub host with a profile token bound to it, with that host's
// rewrites.
func containerGitConfig(credentials map[string]string) map[string]string {
	entries := [][2]string{
		{"safe.directory", "*"},
		{"url.https://github.com/.insteadOf", "git@github.com:"},
		{"url.https://github.com/.insteadOf", "ssh://git@github.com/"},
	}
	if helper, ok := githubTokenCredentialHelper(credentials, "github.com"); ok {
		entries = append(entries, [2]string{gitHubCredentialHelperConfigKey, helper})
	}
	for _, host := range githubauth.EnterpriseHosts(credentials[githubauth.GitHubHostsEnv]) {
		helper, ok := githubTokenCredentialHelper(credentials, host)
		if !ok {
			continue
		}
		for _, prefix := range scriptengine.GitHubSSHPrefixes(host) {
			entries = append(entries, [2]string{"url.https://" + host + "/.insteadOf", prefix})
		}
		entries = append(entries, [2]string{"credential.https://" + host + ".helper", helper})
	}
	config := map[string]string{"GIT_CONFIG_COUNT": strconv.Itoa(len(entries))}
	for index, entry := range entries {
		config["GIT_CONFIG_KEY_"+strconv.Itoa(index)] = entry[0]
		config["GIT_CONFIG_VALUE_"+strconv.Itoa(index)] = entry[1]
	}
	return config
}
