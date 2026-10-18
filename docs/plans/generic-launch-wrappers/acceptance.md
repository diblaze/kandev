# Generic wrapper acceptance

Date: 2026-10-18. Source branch: `feature/investigate-pi-agent-6ac`.
The implementation is under review. PR #4538 stays draft. This record does not certify real sandbox enforcement.

## Backend checks

The following focused checks passed from `apps/backend`:

```bash
go test -trimpath ./internal/agent/launchwrapper ./internal/agent/managedruntime ./internal/agent/settings/cliflags
go test -trimpath ./internal/agent/launchwrapper ./internal/agent/runtime/agentctl ./internal/agent/hostutility ./internal/agent/runtime/lifecycle ./internal/agentctl/server/utility ./internal/agentctl/server/process ./internal/agentctl/server/api -run 'GenericWrapper|BuildAgentCommand_CommandPrefix|Launch_PromotesWorkspaceOnlyExecution|Manager_ProfileCLIFlagTokens|ProfileProbe|ProfileCapability' -count=1
go test -trimpath -race ./internal/agent/launchwrapper ./internal/agent/runtime/environment ./internal/agent/hostutility ./internal/agent/runtime/agentctl ./internal/agent/runtime/lifecycle ./internal/agentctl/server/process ./internal/agentctl/server/utility ./internal/agentctl/server/api -run 'GenericWrapper|LaunchWrapper|ResolveProfile|ProfileProbe|ProfileCapability|ACP.*Cancel' -count=1
go test -trimpath -race ./internal/agentctl/server/process -run 'GenericWrapper|StartOneShot|BuildFinalCommandPreparesOneShot' -count=1
```

A broad process run also exposed a nil-configuration regression in the new stderr filter. The existing stderr-exit test failed with a panic. The filter now preserves that case. `go test -trimpath -race ./internal/agentctl/server/process -run 'GenericWrapper|Stderr|StartOneShot|BuildFinalCommandPreparesOneShot' -count=1` passed after the correction.

The last listed command was run after correcting the one-shot startup path. Its regression test first failed because no wrapper receipt existed. It now proves both initial and continuation execution pass through the wrapper.

Broad hostutility, settings/handlers, runtime/agentctl, and runtime/environment packages passed. Broader API, utility, process, lifecycle, and controller runs did not pass in this host environment. These specific failures also occur on unchanged PR HEAD `e45477c896ae9326324b035d58bddf1e15b1eb5f`:

- `TestOpenCodeInstallPreparesSelectedManagedRuntime`.
- `TestRuntimeObservationRejectsOversizedVersionOutput`.
- `TestAgentctlResolverRemoteBinaryNotFoundErrorNamesPlatformAndEnv`.
- `TestRestartRecoveryRetiresOnlyProvenDeadLoadedConversation`, whose fixtures require unavailable `/bin/sleep`.
- `TestLSPStreamTerminatesWhenCredentialRotates`.
- `TestHandleLSPStreamBridgesFramesAndStopsOwnedProcess`.
- `TestHandleLSPStreamAutoInstallHandsFirstClientMessageToBridge`.
- `TestGitStatusHTTPReturnsRecoveredEnrichedCapture`.
- `TestGitStatusHTTPWarnsOnceAfterContinuousCaptureChanges`.
- `TestGitStatusMultiRetriesOnlyTheRepositoryWithChangedEvidence`.
- `TestWorkspaceGitMalformedNumstatDetailsUnavailable`.
- `TestWorkspaceGitEmptyNumstatDetailsReady`.

A later full process-package run completed in 107 seconds with only the two listed numstat test failures. The lifecycle suite exceeded a four-minute whole-package limit while running Kubernetes tests. It did not report a wrapper test failure. This run is incomplete, not a passing broad-suite result.

The baseline comparison used an isolated source archive. The controller comparison used an overlay containing unchanged HEAD files only. Neither comparison is acceptance evidence for a provider exception. Production implementation and browser builds use current source without an overlay.

Changed-Go lint passed with golangci-lint 2.9.0. A separate temporary lint directory avoided another worktree's lock. The commit hook runs the same lint after the final stderr correction. The changed-file lint command passes every changed Go path to `bash scripts/lint-go-changed`. Expanded command:

```bash
bash scripts/lint-go-changed \
  apps/backend/internal/agent/hostutility/manager.go \
  apps/backend/internal/agent/hostutility/manager_test.go \
  apps/backend/internal/agent/hostutility/profile_capability_cache.go \
  apps/backend/internal/agent/hostutility/profile_probe.go \
  apps/backend/internal/agent/hostutility/profile_probe_test.go \
  apps/backend/internal/agent/hostutility/public.go \
  apps/backend/internal/agent/managedruntime/npm_project.go \
  apps/backend/internal/agent/runtime/agentctl/client.go \
  apps/backend/internal/agent/runtime/agentctl/utility.go \
  apps/backend/internal/agent/runtime/lifecycle/command.go \
  apps/backend/internal/agent/runtime/lifecycle/manager_interaction.go \
  apps/backend/internal/agent/runtime/lifecycle/manager_kubernetes_refresh.go \
  apps/backend/internal/agent/runtime/lifecycle/manager_launch.go \
  apps/backend/internal/agent/runtime/lifecycle/manager_launch_test.go \
  apps/backend/internal/agent/runtime/lifecycle/manager_lifecycle.go \
  apps/backend/internal/agent/runtime/lifecycle/manager_passthrough.go \
  apps/backend/internal/agent/runtime/lifecycle/manager_passthrough_test.go \
  apps/backend/internal/agent/runtime/lifecycle/types.go \
  apps/backend/internal/agent/runtime/lifecycle/utility.go \
  apps/backend/internal/agent/runtime/lifecycle/utility_test.go \
  apps/backend/internal/agent/settings/cliflags/tokenise.go \
  apps/backend/internal/agent/settings/cliflags/tokenise_test.go \
  apps/backend/internal/agentctl/server/api/auth.go \
  apps/backend/internal/agentctl/server/api/identity.go \
  apps/backend/internal/agentctl/server/api/server.go \
  apps/backend/internal/agentctl/server/config/config.go \
  apps/backend/internal/agentctl/server/process/interactive_lifecycle.go \
  apps/backend/internal/agentctl/server/process/interactive_runner.go \
  apps/backend/internal/agentctl/server/process/manager.go \
  apps/backend/internal/agentctl/server/process/manager_managed_git_handoff_test.go \
  apps/backend/internal/agentctl/server/utility/acp_executor.go \
  apps/backend/internal/agentctl/server/utility/acp_executor_process_unix_test.go \
  apps/backend/internal/agentctl/server/utility/codex_app_server.go \
  apps/backend/internal/agentctl/server/utility/handler.go \
  apps/backend/internal/agentctl/server/utility/types.go \
  apps/backend/internal/backendapp/main.go \
  apps/backend/internal/agent/hostutility/profile_environment.go \
  apps/backend/internal/agent/launchwrapper/codec.go \
  apps/backend/internal/agent/launchwrapper/environment.go \
  apps/backend/internal/agent/launchwrapper/executable.go \
  apps/backend/internal/agent/launchwrapper/prepare.go \
  apps/backend/internal/agent/launchwrapper/prepare_test.go \
  apps/backend/internal/agent/launchwrapper/spec.go \
  apps/backend/internal/agent/launchwrapper/spec_test.go \
  apps/backend/internal/agent/runtime/agentctl/generic_wrapper_test.go \
  apps/backend/internal/agent/runtime/agentctl/launch_wrapper.go \
  apps/backend/internal/agent/runtime/environment/profile.go \
  apps/backend/internal/agent/runtime/environment/profile_test.go \
  apps/backend/internal/agent/runtime/lifecycle/generic_wrapper_passthrough_test.go \
  apps/backend/internal/agent/runtime/lifecycle/generic_wrapper_recovery_test.go \
  apps/backend/internal/agent/runtime/lifecycle/generic_wrapper_utility_test.go \
  apps/backend/internal/agent/runtime/lifecycle/launch_wrapper_recovery.go \
  apps/backend/internal/agent/runtime/lifecycle/utility_environment.go \
  apps/backend/internal/agentctl/server/api/generic_wrapper_configure_test.go \
  apps/backend/internal/agentctl/server/api/generic_wrapper_test.go \
  apps/backend/internal/agentctl/server/api/launch_wrapper.go \
  apps/backend/internal/agentctl/server/process/generic_wrapper_interactive_test.go \
  apps/backend/internal/agentctl/server/process/generic_wrapper_test.go \
  apps/backend/internal/agentctl/server/process/launch_wrapper.go \
  apps/backend/internal/agentctl/server/utility/generic_wrapper_failure_test.go \
  apps/backend/internal/agentctl/server/utility/generic_wrapper_test.go \
  apps/backend/internal/agentctl/server/utility/launch_wrapper.go
```

## Browser and documentation checks

Desktop Chromium: the full settings and terminal selection passed (4 tests in 47.6 seconds). After adding the ACP chat restart case and refining the terminal fixture, both terminal/ACP tests passed again (2 tests in 23.1 seconds). All 5 mobile settings tests passed. The mobile terminal retry passed (1 test in 14.8 seconds), including touch targets, input, restart, and process cleanup. All 11 unique selected browser tests now pass. Tests use an unrelated temporary launcher with a path containing spaces. The launcher verifies the runtime working-directory reference and records its PID before it executes the child. It does not depend on the built-in utility allowlist.

Initial browser setup was blocked by inherited Kandev settings and Chromium shared-library lookup. The retry removes inherited Kandev/agentctl overrides and uses host-installed Nix libraries for the test process only. No manual runtime settings or processes were changed.

The terminal test uses short markers that fit the phone viewport. It waits for idle session state through the API because the desktop workflow stepper is absent on the phone.

The first web typecheck exhausted Node's default 4 GiB heap. `NODE_OPTIONS=--max-old-space-size=8192 pnpm --dir apps/web run typecheck` passed. Targeted ESLint for the six changed browser test/helper files also passed.

The specification validator, full spec lint, 36 spec-lint tests, public-doc tests, and public-doc validation passed. Local PR documentation coverage checks passed for all six work orders. Architecture lint passed. The commit hooks also validate the final documents.

## Fresh Pi/Greywall discovery

On 2026-10-18, the user authorized tool installation. Installed Pi 1.3.0, pi-acp 0.0.34, Greywall 0.3.7, and Greyproxy 0.4.5 in a new private directory outside the repository and `/tmp`. The Greywall and Greyproxy release archives passed their published SHA-256 checksums. The npm installation used exact package versions, disabled lifecycle scripts, and retained its lockfile in the private installation. No global npm package or service was changed.

The host runs NixOS 26.05, Linux 6.18.49, and Node 24.19.0. The installation uses the existing Bubblewrap 0.11.2 and an isolated Nix installation of socat 1.8.1.3. Greywall successfully launched a Node subprocess. This smoke check does not certify sandbox enforcement.

Built `acpdbg` and `agentctl` from commit `23eee8e8b6b673dd1b4921d542b75e30ef7b4142` with `go build -trimpath`, no Go overlay, and no inherited build flags. Fresh disposable homes used only a synthetic OpenAI key. No real provider credential was loaded and no model prompt was sent.

Both direct and Greywall-wrapped `acpdbg` discovery completed `initialize` and `session/new`. Each returned 44 models and 5 modes. The captured frames contained zero `session/prompt` requests.

A separate authenticated agentctl control process then repeated direct and wrapped discovery through `/api/v1/inference/probe`. Both returned success, 44 models, and 5 modes. The inner command used the managed npm marker and the pinned adapter. The generic wrapper envelope used `{{kandev.work_dir}}` and `{{kandev.managed_npm_dir}}`. The wrapped run produced exactly one nonce-bearing launcher receipt before executing Greywall. No provider-specific resolver exception was used.

The sandbox received explicit grants for installation reads and the disposable home, npm cache, temporary directory, working directory, and managed npm directory. It did not use learning mode, watch mode, a provider profile, or an allow-all network rule. Proxy endpoints used dynamically allocated local ports with no proxy listening. The adapter cache was prepared before the check; the probe used npm offline mode. Greyproxy was installed but was not started for these prompt-free checks.

The initial agentctl checks exposed test setup issues. Offline npm first needed its adapter cache prepared. Greywall's Unix socket exceeded Linux's path-length limit under the long private temporary path. A shorter, dynamically allocated temporary directory fixed that failure. Private links to the resolved Nix Node/npm/npx tools fixed a launcher path that was absent inside the sandbox. These corrections changed test setup only. Each agentctl process, disposable home, and temporary directory was removed after its run. The retained manual instance was not used.

## Remaining real-provider acceptance

The optional handoff `/tmp/handoff-Fhcnxu.md` remains unavailable. No explicit disposable provider credential source, model identity, or provider network policy has been supplied. Installation and prompt-free discovery are complete. Live utility tool use, ACP chat, and native passthrough still require those inputs.

No credential was copied from the retained manual instance. The checked-in real-provider runner remains unfinished, and Task 06 stays incomplete. The historical overlay run in [reproduction.md](reproduction.md) does not satisfy the remaining acceptance items. Finish and test the isolated runner, then record fresh live receipts without an overlay. Keep raw traces and credentials private.

The manual instance and deprecated #3812 task remain unchanged. The requirement and system-design pair remains draft.
