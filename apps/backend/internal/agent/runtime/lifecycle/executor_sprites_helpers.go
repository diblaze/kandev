package lifecycle

import (
	"context"
	"errors"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	sprites "github.com/superfly/sprites-go"
	"go.uber.org/zap"

	commonconfig "github.com/kandev/kandev/internal/common/config"
	"github.com/kandev/kandev/internal/githubauth"
	"github.com/kandev/kandev/internal/scriptengine"
)

var uploadHTTPStatusRE = regexp.MustCompile(`(?i)\b(?:http|status)\s*:?\s*(\d{3})\b`)

// injectGitHubTokenIntoCloneURL injects only GitHub credentials into a clone URL so
// in-container `git clone` works without prompting. The URL host must be
// github.com or a host listed in githubauth.GitHubHostsEnv, and the token comes
// only from the profile variables bound to that host
// (githubauth.ProfileTokenEnvForHost). The fallback variable wins when both
// are present. Uses the documented `https://x-access-token:T@<host>/` form,
// which works for both clone and gh CLI authentication.
//
// GitLab credentials are deliberately provided through an exact-origin
// ephemeral git credential helper in the execution environment, never in a URL.
// Used by both Docker and Sprites executors. SSH→HTTPS rewrite is delegated
// to rewriteGitHubSSHToHTTPS so the two surfaces never drift again.
func injectGitHubTokenIntoCloneURL(cloneURL string, env map[string]string) string {
	hosts := append([]string{"github.com"}, githubauth.EnterpriseHosts(env[githubauth.GitHubHostsEnv])...)
	for _, host := range hosts {
		primary, fallback := githubauth.ProfileTokenEnvForHost(env["GH_HOST"], host)
		token := env[fallback]
		if token == "" {
			token = env[primary]
		}
		if primary == "" || token == "" {
			continue
		}
		httpsURL := cloneURL
		if converted := rewriteGitHubSSHToHTTPS(cloneURL, host); converted != "" {
			httpsURL = converted
		}
		if path, ok := strings.CutPrefix(httpsURL, "https://"+host+"/"); ok {
			return "https://x-access-token:" + token + "@" + host + "/" + path
		}
	}
	return cloneURL
}

// rewriteGitHubSSHToHTTPS returns the https://<host>/ form of an SSH remote on
// host, or "" when remoteURL is not one.
func rewriteGitHubSSHToHTTPS(remoteURL, host string) string {
	for _, prefix := range scriptengine.GitHubSSHPrefixes(host) {
		if path, ok := strings.CutPrefix(remoteURL, prefix); ok {
			return "https://" + host + "/" + path
		}
	}
	return ""
}

func (r *SpritesExecutor) writeFileWithRetry(
	ctx context.Context,
	sprite *sprites.Sprite,
	path string,
	data []byte,
	mode os.FileMode,
) error {
	backoff := 700 * time.Millisecond
	var lastErr error
	for attempt := 1; attempt <= spriteUploadMaxRetries+1; attempt++ {
		err := sprite.Filesystem().WriteFileContext(ctx, path, data, mode)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt > spriteUploadMaxRetries || !isTransientUploadError(err) || ctx.Err() != nil {
			break
		}

		jitter := time.Duration(rand.Intn(300)) * time.Millisecond
		wait := backoff + jitter
		r.logger.Warn("retrying sprite file upload after transient error",
			zap.String("path", path),
			zap.Int("attempt", attempt),
			zap.Duration("retry_in", wait),
			zap.Error(err))
		time.Sleep(wait)
		backoff *= 2
	}
	return lastErr
}

func isTransientUploadError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	msg := strings.ToLower(err.Error())
	if status := extractUploadHTTPStatus(msg); status != 0 {
		if status == 408 || status == 429 || status >= 500 {
			return true
		}
	}
	return strings.Contains(msg, "client.timeout exceeded while awaiting headers") ||
		strings.Contains(msg, "request canceled") ||
		strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "text file busy") ||
		strings.Contains(msg, "connection reset") ||
		strings.Contains(msg, "temporary")
}

func extractUploadHTTPStatus(msg string) int {
	matches := uploadHTTPStatusRE.FindStringSubmatch(msg)
	if len(matches) < 2 {
		return 0
	}
	code, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0
	}
	return code
}

func (r *SpritesExecutor) buildSpriteEnv(env map[string]string, startup ...commonconfig.AgentctlStartupConfig) []string {
	merged := env
	if len(startup) > 0 {
		merged = mergeAgentctlStartupEnvironment(env, startup[0])
	}
	result := make([]string, 0, len(merged))
	for k, v := range merged {
		if k == githubauth.CredentialHelperPathEnv {
			continue
		}
		result = append(result, k+"="+v)
	}
	if hasManagedGitHubBrokerEnv(env) {
		result = append(result, githubauth.CredentialHelperPathEnv+"="+remoteAgentctlExecutablePath)
	}
	return result
}

type spritesStepKey string

const (
	spriteStepCreateSprite       spritesStepKey = "create_sprite"
	spriteStepResolveAgentctl    spritesStepKey = "resolve_agentctl"
	spriteStepUploadAgentctl     spritesStepKey = "upload_agentctl"
	spriteStepUploadCredentials  spritesStepKey = "upload_credentials"
	spriteStepRunPrepareScript   spritesStepKey = "run_prepare_script"
	spriteStepWaitHealthy        spritesStepKey = "wait_healthy"
	spriteStepAgentInstance      spritesStepKey = "agent_instance"
	spriteStepApplyNetworkPolicy spritesStepKey = "apply_network_policy"
)

// spritesProgressPlan is mutable so the reconnect path can swap to the fresh
// plan mid-flight when a stale sandbox falls back to provisioning a new one.
// The reporter created by newSpritesStepReporter resolves indexes lazily off
// the live plan, so callers don't need to re-create it after replacePlan.
type spritesProgressPlan struct {
	steps   []spritesStepKey
	indexes map[spritesStepKey]int
}

func newSpritesProgressPlan(reconnect bool) *spritesProgressPlan {
	if reconnect {
		return buildPlan([]spritesStepKey{
			spriteStepCreateSprite,
			spriteStepWaitHealthy,
			spriteStepAgentInstance,
		})
	}
	return buildPlan([]spritesStepKey{
		spriteStepCreateSprite,
		spriteStepResolveAgentctl,
		spriteStepUploadAgentctl,
		spriteStepUploadCredentials,
		spriteStepRunPrepareScript,
		spriteStepWaitHealthy,
		spriteStepAgentInstance,
		spriteStepApplyNetworkPolicy,
	})
}

func buildPlan(steps []spritesStepKey) *spritesProgressPlan {
	indexes := make(map[spritesStepKey]int, len(steps))
	for i, key := range steps {
		indexes[key] = i
	}
	return &spritesProgressPlan{steps: steps, indexes: indexes}
}

func (p *spritesProgressPlan) total() int {
	if p == nil {
		return 0
	}
	return len(p.steps)
}

func (p *spritesProgressPlan) has(key spritesStepKey) bool {
	if p == nil {
		return false
	}
	_, ok := p.indexes[key]
	return ok
}

func (p *spritesProgressPlan) index(key spritesStepKey) int {
	if p == nil {
		return -1
	}
	idx, ok := p.indexes[key]
	if !ok {
		return -1
	}
	return idx
}

// replacePlan swaps the active plan in place (used by the reconnect-fell-back-
// to-fresh path). The reporter dispatches off the live indexes, so existing
// closures keep working after replacement.
func (p *spritesProgressPlan) replacePlan(steps []spritesStepKey) {
	if p == nil {
		return
	}
	indexes := make(map[spritesStepKey]int, len(steps))
	for i, key := range steps {
		indexes[key] = i
	}
	p.steps = steps
	p.indexes = indexes
}

// newSpritesStepReporter creates a reporting function that calls OnProgress if
// non-nil. Index/total are resolved against the live plan on each call so
// replacePlan is reflected in subsequent reports.
func newSpritesStepReporter(onProgress PrepareProgressCallback, plan *spritesProgressPlan) func(spritesStepKey, PrepareStep) {
	return func(key spritesStepKey, step PrepareStep) {
		if onProgress == nil {
			return
		}
		idx := plan.index(key)
		if idx < 0 {
			return
		}
		onProgress(step, idx, plan.total())
	}
}

// lastLines returns the last n lines of s.
func lastLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= n {
		return s
	}
	return strings.Join(lines[len(lines)-n:], "\n")
}
