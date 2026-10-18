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

## Real-provider acceptance blocker

`pi` and `greywall` are not on this session's PATH. The optional handoff `/tmp/handoff-Fhcnxu.md` is unavailable. No explicit private credential source or sandbox policy was supplied for a new acceptance run.

No real-provider prompt was sent. No credential was copied from the retained manual instance. The real-provider runner remains unfinished, and Task 06 stays incomplete. The historical overlay run in [reproduction.md](reproduction.md) does not satisfy this acceptance item.

To complete this item, provide the private Pi/Greywall installation paths, an explicit disposable credential source, model identity, and permitted sandbox network/path policy. Then finish and test the isolated runner, and record fresh discovery, utility tool use, ACP chat, and native passthrough receipts without an overlay. Keep raw traces and credentials private.

The manual instance and deprecated #3812 task remain unchanged. The requirement and system-design pair remains draft.
