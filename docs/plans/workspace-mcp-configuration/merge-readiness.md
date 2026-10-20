# PR 3293 merge readiness

Work resumed on another machine on 2026-10-20. The published checkpoint was
verified and conflicts were resolved with current main. Integrated local checks
and bounded source review are recorded below. Final capture publication,
current-head CI and final review of the test/documentation delta remain pending.
The user requires confirmation before merging.

The earlier receipts below are historical. Their local files and review session
are unavailable on this machine. They do not establish current readiness.

## Resumed delivery

The current task is `31c5c5a9-c42d-4782-9a72-82a525ffee0b`. Its Kandev plan owns
the current verification ledger. The canonical delivery branch remains
`feature/plan-workspace-mcp-c-dcc`. The old integration alias is historical.

A separate clone isolates delivery from a broken ref in the shared repository.
Current-main conflict resolution preserves MCP selection state and the newer
initial-workspace-layout controls. It also retains durable workspace path
reconciliation alongside MCP repository identity collection.

Fresh frontend checks passed: the initial affected suite (255 tests across
14 files), the extended task-edit/merge regressions (96 tests), typecheck with
an explicit 6 GiB heap, full ESLint, localization, and the web build. The added
coverage checks MCP rejection/retry before launch, explicit empty selections,
clean/started edits, a real editor retry, and combined MCP/layout create/reset
behavior. The final editor test cleanup also passed its focused rerun.

Real PostgreSQL MCP deletion and snapshot tests passed with race detection.
The first Go run failed and is retained as diagnostic evidence. Its failures
included an incorrect streams package path and fixture assumptions about
`/bin/sleep`, `/usr/bin`, and a usable shell PATH with an empty environment.
A private Bubblewrap namespace supplies those paths without changing the host.
Six packages passed in the corrected run. Lifecycle exceeded the ten-minute
package runner limit without an assertion failure. A separate bounded partition
processed its remaining 487 declared tests. The union accounts for all 2,725
declared lifecycle tests: 2,704 passed and 21 skipped, with no missing names or
test assertion failures. Skips include opt-in container/provider checks,
macOS-only checks, release/browser/subprocess helpers, a case-insensitive
filesystem check and unavailable zsh fixtures. They are not passing coverage.
This is partitioned evidence, not a claim that the timed-out command passed.

The pruned-delivery startup fixture separately failed on both the integrated
source and exact main `e3ef99a98f8d340f2447650cb4318b00977eb44e`. Its context
expired during schema setup, before recovery. The fixture now initializes its
database before creating the unchanged three-second recovery context. Both
delivery modes pass. This is a test-budget correction, not a production change
or a claim that baseline evidence excludes every possible PR contribution.
The workspace placement/MCP repository-scope integration regression also passes.

Current Go lint identified repeated literals. The integrated source now reuses
the typed stdio constant, shares Claude's reserved server name between config
and delivery evidence, and reuses the existing non-durable storage reason.
The affected passthrough, Claude strategy and native compatibility tests passed
after these behavior-preserving changes.

The latest main update changes delivery-retention scheduling. Worker tests and
its two new repository regressions passed on SQLite and PostgreSQL. A broader
SQLite retention run reached its five-minute runner limit during schema setup.
It remains failed evidence; the focused checks do not imply that the broad run
completed or that every retention behavior was verified locally.

The current web, native runtime, delivery fixture and plugin package built
successfully. Eighteen affected browser cases passed across desktop, phone,
narrow fine-pointer and coarse-pointer tablet layouts, with one worker and
retries disabled. They include 320/393/767px Connections geometry, incremental
shell reload/cancellation, and combined MCP/layout selection and persistence.

The initial desktop run passed eight cases and failed the new 28px assertion
because it sampled the dialog during its opening animation at 27.407px. The
geometry helper now awaits existing finite animations before measurement.
No size assertion or product animation changed. The desktop and tablet geometry
cases each passed three subsequent runs. All three phone sizes passed again,
as did the combined Advanced case and a separate phone capture flow.
The Advanced capture now scrolls both settings into view before taking the image.

The isolated review session is `213febdc-c9bc-4579-a513-5dd0415997fc`.
Its bounded review of the immutable integration commit found no new blocking
issues and marked the task-edit MCP coverage finding addressed. It reviewed the
seven conflict resolutions, added coverage, deadline placement and literal reuse.
It did not audit every file in the original PR or establish runtime/CI approval.
The final test synchronization and documentation delta requires bounded closure.
No merge, auto-merge, queue admission or follow-up execution is authorized.

## Historical local evidence from the previous machine

| Check | Result |
| --- | --- |
| Full changed and affected frontend tests | 42 files, 478 tests passed |
| Full affected Go runtime | 11 packages passed, including lifecycle and agentctl API |
| Real PostgreSQL | Atomic deletion, rollback and coherent resolution passed |
| Orchestration integration | Backend app, orchestrator, task service/handlers and MCP handlers passed |
| Static checks | Go lint zero issues; frontend lint/typecheck, localization and gofmt passed |
| Documentation | Index/spec lint and coverage of 26 work orders/44 documents passed |
| Artifacts | Web, embedded assets, native runtime, delivery fixture and plugin package built |
| Desktop/tablet browsers | 23 distinct cases passed with retries disabled |
| Phone browsers | 20 distinct cases passed with retries disabled, including 320/393/767px Connections |

The affected settings suites were rerun after the final width repair: seven
desktop and five phone cases passed. The remaining browser flows use unchanged
production code. The MCP dialog keeps its explicit single-column grid. No later Go runtime
change invalidates the full runtime run. All review findings through the fifth
isolated round are addressed. The latest CI candidate additionally repairs
frontend/browser fixtures and a prompt-preview dismissal race; its exact-head
review and CI are recorded separately below.

The unchanged terminal integration fixture passes with a controlled POSIX shell;
the host's customized interactive shell is unsuitable for that fixture. Pending
MCP tests use a busy-until-cancel mock without an unnecessary post-cancel delay.
No production setting was changed to make either fixture pass. Earlier timeout,
failed and historical-preview runs are not counted as successful evidence.

Main :9998 retains its original process and health 200. The old pod-local preview
is not claimed as the final tested artifact. The managed browser runs use fresh,
disposable data and the current rebuilt runtime.

## Integration and remediation record

User scope on 2026-10-19: finish and review the existing implementation, resolve
valid findings and PR comments, pass CI, then request confirmation before merging.
No merge or auto-merge is authorized yet.

## Deferred work

Four queued children were created under the resumed task:

- Management usage/global-profile navigation and neutral delivery presentation
  (points 3/4): `2adba1ca-619e-4bfc-b705-3bbc628df4dd`.
- Session-only inherited exclusions (point 1):
  `a3588dfb-7c25-4f91-9a3d-665ce89c6605`.
- Optional assignment after registration (point 2):
  `58d6bce2-eab4-4ac6-9ac4-50a31011e1e1`.
- Built-in management and tool pruning (point 5):
  `6d1b4323-e821-4599-acea-07273a60dd78`.

All four have automatic starts disabled and no sessions. Each requires explicit
user authorization before work. Each must start from actual current main
containing the final parent merge, preserving user edits.

The previous machine's six-file management design package and Platform patch
were not supplied or recovered. The new child's plan records scope and this
gap. It does not claim to contain the archived package. These features remain
excluded from this PR.

## Integration identity

Published PR branch: `feature/plan-workspace-mcp-c-dcc`, captured remote head
`2ff981eb1ba07927e600e7f5cbc4e99b53961d05`. The local takeover branch had
rebased-equivalent older history; its two authored commits were replayed onto
that exact published head without dropping the remote's task-field/E2E fixes.
Pre-integration archive: `archive/pr3293-before-upstream-integration-20261019`.

Captured live base: `b544f15d81ca45a2bfbc8c61d695d7d818aefecd`. The current-base
merge required 62 conflict resolutions, including runtime journal/adoption and
owner wiring, coordinator/settings routes, translation catalogs and older E2E
stabilization work. Preserve the newer base behavior alongside MCP selection.
Accepted setup recording follows main's extracted passthrough-reset owner.
The main child-first E2E reset supersedes the old retry-based helper; current
Chart/provider/touch/ETag tests supersede the old boundary-specific variants.
Catalog JSON key merge had no overlapping changed-value conflicts.

## Historical independent review

User-requested separate Kandev session: `881edc17-320c-40c3-a8be-6dd4fd28cc86`,
Isolated PR review. It owns a private detached review worktree and has no
implementation or GitHub mutation authority. Parent owns remediation. Initial
snapshot: `d30fed2ba6c2c616f366e0027e1c84699e17079c` plus archived picker/mobile
repair patch (tree `03a75a9d94e42a41992223faa8ad8a5c0c0fa726`). A final integrated
head delta review is required before declaring readiness.

## Verification

Pending completion. Current merged-tree evidence:

- Focused MCP frontend suite: 11 files, 112 tests passed.
- Locale validation passed for all seven shipped languages and pseudo.
- Catalog: 390 decisions/1661 specs; all-spec lint passed.
- PR-documentation coverage: covered, 23 changed work orders/40 referenced
  documents, zero errors.
- Harness validator: 204 files passed; harness/spec-linter unit suites passed
  (19 and 36 cases). API scoped guide: 36 lines, targeted harness hook passed.
- Actual PR delta whitespace against captured main passed. Incoming main's CSV
  line endings/Markdown spacing do not represent authored changes in this PR.
- Backend tests, typecheck, lint, post-integration browser checks, final review
  and exact-head CI remain pending. Earlier preview receipts are historical,
  not proof of this newer merged runtime. Main instance :9998 is untouched.

Final disposition and exact-head results will replace pending statements here.

## Review remediation checkpoints

Round one identified nine valid findings. Real PostgreSQL deletion/snapshot tests, legacy URL import tests, editor-context regressions, argument/disabled-selection tests, required-input validation and exact-operation evidence tests reproduced failures and passed after repair. Round two reviewed immutable tree `8408383d075359fb63662174317ae2156358bdf5` against main `b544f15d81ca45a2bfbc8c61d695d7d818aefecd` and found unsupported Registry argument metadata and project-file provider reload acceptance still incomplete. Both were reproduced and repaired; expanded final validation remains pending.

Additional main integration regressions exposed missing managed-conversation admission on resume and missing visibility for helper-side policy filtering. API tests reproduced both and pass with shared admission across all four session operations. Startup acceptance/publication uses the existing generation lease. No executable placeholders are delivered.

Current validation includes 117 frontend tests across 15 files; frontend typecheck, full lint and localized catalogs; a web build and complete runtime/plugin-fixture build; 48 public documentation pages; 390 decisions/1662 specs; and PR coverage for 26 work orders/44 documents. New source changes invalidate earlier artifact/final-head claims and require the recorded final reruns. Two bounded Go lint attempts timed out while loading packages; neither counts as passing. A third uses trimpath cache identity to match the runtime build.

The installed GitHub CLI 2.45 cannot upload screenshots. A temporary GitHub CLI 2.102.0 was downloaded from the official release and verified against its published SHA-256 digest; native attachment support is confirmed. Publication and final-viewport recapture remain pending.

Round-three delta review closed both round-two findings and identified a P1 lock-order inversion in idle acceptance. A real application regression gated the ACP load, admitted a completion callback and queued a startup writer; it reproduced the deadlock (RED) and passed after the startup lease moved before the prompt lock (GREEN). Ordinary ACP application, project-provider deferral, exact setup acceptance and CLI operation regressions passed in the same focused run. A narrow follow-up audit checks the other generation-lease placements before final review.

The third lint attempt also exceeded context loading. Compiler response files identified internal/orchestrator and internal/task/service as the large jobs, consuming roughly 3–4 GiB each under a 3 GiB soft limit. Their type exports are being warmed sequentially with a bounded 5 GiB limit and 10-minute command limit; lint will retain its normal five-minute budget. No failed or timed-out lint run is counted as passing.


Round four verified both lock-order repairs and found that detached idle updates lacked a host deadline. Controlled unanswered reconnect and stalled restoration tests reproduced retained leases and an expired-context failure write. Both now pass with a two-minute operation deadline and a separate five-second persistence budget, preserving old acceptance and releasing a real queued startup writer. Focused lifecycle, PostgreSQL store, Registry and API checks passed after these changes. Five lint findings from the first successfully loaded analysis were repaired; final lint remains pending.


Round-five isolated review checked tree `29c0f8098f1e5b6b009c22b8d9e85f0d6966367e` against round four, confirmed the host deadline and bounded failure persistence, and found no further concrete issue. No unresolved independent-review finding remains in its inspected paths. Final committed-head identity and integration/CI gates remain separate. Current focused frontend verification passed 107 tests in 13 files. Remote main and the published PR head still match the captured b544/2ff references; the GitHub review-thread helper returned successfully with no actionable entries.


Final local static/runtime checkpoint: changed-package Go lint passed with zero issues after source exports were warmed with an 8 GiB per-process memory target; the normal five-minute lint timeout was retained. Full frontend lint passed after extracting a duplicate style string. Current typecheck, localization, web production build and embedded-asset sync passed. All 36 changed frontend test files passed (416 tests). The full affected Go run passed all 11 packages, including lifecycle (224.197s), API (92.005s), Registry, settings/handlers/store with real PostgreSQL, runtime client/launcher and streams. The earlier cold-cache checkout/startup failures did not recur. Documentation index/spec lint/26-work-order coverage and existing-file gofmt check passed. Native rebuild, integrated browser cases, commit hooks, exact committed-head review and GitHub CI remain pending.


The remaining scoped integration run passed backendapp, orchestrator/executor/message queue, task service and MCP handlers. The unchanged Cursor authentication terminal fixture initially inherited the host customized zsh and failed to exit; rerunning task handlers with `SHELL=/bin/sh` and shell startup variables unset passed (0.269s). This is a controlled test environment, not a production change or a skipped assertion. Main :9998 retained the same process and returned health 200. The native/runtime/plugin-fixture rebuild is underway before managed browser execution.


The rebuilt native runtime, delivery fixture and plugin package passed. Desktop managed E2E initially passed 21/22 cases and exposed a fixture mismatch: `/e2e:cancel-hold` now delays cancellation eight seconds while the default join budget is three seconds, invoking recovery and closing a just-opened dialog. Pending-change tests now keep the turn busy until cancellation but use a zero mock post-cancel delay, wait for observable editable idle, then inspect real MCP acceptance. The repaired desktop case passed with retries disabled (one case, 37.0s runner). Other 21 desktop cases passed against the same source. Phone verification is running. No production behavior was changed for this fixture repair.


Phone E2E passed 19/20 cases, including pending application, 320/393/767px Connections geometry, provenance, settings inputs and recovery. The long unsupported-package case reproduced a real internal overflow: the dialog's implicit grid track expanded to min-content width. An explicit `grid-cols-1` gives that track a zero minimum rather than clipping content. The same long-name regression now shares a helper across desktop and phone settings tests. Current web/native rebuild and scoped lint passed; both complete settings suites are being rerun against this fix. The merge brings 175 staged Go packages through normal hooks, so their test export cache was warmed successfully without running tests or changing the five-minute lint budget.


Published-head CI identified the strict frontend fixture integration gap recorded
in task 11. The failures reproduced (five files/36 failures), then passed with
known-owner MCP read responses, complete mocked form state and an explicit
secrets/MCP/coordinators ordering assertion. Combined affected tests passed
41 files/454 cases; final focused checks passed five files/38 cases, scoped lint
and typecheck. No production/UI code changed. Current-head CI still requires a
fresh passing result after the next normal remediation commit and push.


## Published-head CI remediation

The first complete a57 run ended with 81 passed, 11 skipped and 11 failed
checks. Backend tests passed. Frontend fixture failures are repaired as above.
E2E artifact audit covered all 118 blob reports with no parse errors: 3,998
first-attempt passes, nine flaky verdicts, four failed cases and 40 skipped.
These failures are not counted as passing evidence. PR review evidence was
complete with no unresolved, hidden or actionable threads.

Browser fixture repairs restore current-main worktree-path capture and the
preview screenshot-helper import. The session-deletion test uses the same
captured session ID for its action and assertion. Local path regression was
RED then three cases passed; preview and the full session suite passed under
two CPUs/4 GiB with retries disabled. A changed-E2E identifier audit exposed
only the missing preview helper after path repair.

The prompt-history failure reproduced under the same limits. A component
regression showed a delayed hover timer reopening a preview after Escape.
Explicit dismissal now suppresses delayed reopen until a new trigger
interaction; click/keyboard toggles and later hover remain supported. This
changes prompt-preview interaction only, with no MCP layout or copy change.
The original click/Escape browser assertion is retained. After rebuilding,
that plugin scenario and the existing native hover scenario both passed
under the same pressure limits (two tests, 26.2 seconds). The final affected
frontend run passed 42 files/478 tests, including all 24 prompt component
cases; frontend typecheck and scoped lint passed. Final commit review and
current-head CI remain gates.

Other first-run flaky cases were investigated without speculative product
changes: workflow/Azure focused cases and their ordered pressure replays
passed (37 and 12 cases); dynamic fallback desktop passed under pressure;
the combined phone autoscroll, fallback, plan-comment and workspace-clone
suites passed 19 cases. All used retries zero and one worker. Independent
source audits found no concrete new MCP transaction or dynamic-startup lock
defect; the initial transient causes remain unknown. Fresh complete CI on
the remediation commit is still required before merge confirmation.


## Second-head browser fixture synchronization

On da26, CI found two further fixture synchronization failures outside the
feature source. The workflow move-preview helper matched only a task URL and
read a browser response body after UI waits. A controlled competing response
for another destination reproduced a wrong `reuse_current` result where
`create_new` was expected. The helper now matches the destination step, consumes
JSON immediately, observes transport and UI together, and asserts task/step
identity. No body error is swallowed, retried or replaced by a separate read.
All existing move, model, recipient and final-session assertions remain. The
complete spec passed three cases; its ordered desktop CI partition replay
passed 35 cases under two CPUs/4 GiB, one worker and retries disabled.

The Threads deep-link test asserted a chat before its task's session list had
loaded. An eight-second initial GET stimulus reproduced the same loading-state
failure. Reusing the existing client session-settlement helper fixes the test
without changing its rendered assertion timeout or the product loader. The
complete two-case layout suite passes with that real slow read and all stream
bounds, row geometry, unmount/remount and navigation assertions retained.
The exact original CI request timings and CDP resource failure are not proven
from the saved snapshots; these deterministic counterexamples establish the
fixture weaknesses repaired here.

This delta changes only browser tests and this record. Product source,
42-file/478-test unit evidence and published MCP/prompt captures remain
unchanged. Final immutable delta review, normal hooks and fresh committed-head
CI remain required. The optional automated walkthrough separately timed out
twice at its fixed generation deadline; no workflow or check policy was changed.


The repeated mobile freeze case had two observation gaps: an immediate DOM
scroll assignment captured an unsettled baseline, and the substring marker
could match the outgoing prompt before the assistant appended its reply. Its
fixture now completes a native touch gesture, verifies trusted start/move/end,
waits for stable position and geometry, and checks the exact assistant marker.
It also checks preservation immediately after disabling. The final ten-pixel
bounds and two-second budget, real composer/send path and history-depth
preconditions remain. No transcript implementation or tolerance changed.
The old wheel-only helper failed the trusted-touch setup contract; five focused
native-gesture repeats passed under the same resource limits. That setup RED
is not a reproduction or claimed root cause of the intermittent CI offset.
Full phone-suite validation and final committed-head CI remain gates.

The complete da26 artifact audit reconciles 118 blob reports: 4,012 first-attempt
passes, three retry passes, zero final failed cases and 40 skipped. Those three
flaky verdicts are exactly the workflow, Threads and mobile fixtures described
above; fail-on-flaky correctly kept the required E2E gate red. No parse errors
or additional failed cases were found. All six container shards passed.


Final fixture review found and closed a history-precondition gap: the settled
sample now requires both a nonzero history offset and more than 100 pixels
remaining below the viewport. A local controlled schedule that holds settlement
at the bottom was accepted before this check (RED) and rejected after it
(GREEN). The probe changes only the disposable test clone and is not committed.
It does not reproduce the original CI offset. Three edited browser specs pass
all TypeScript diagnostics; the 60-file changed-E2E unresolved-name/import audit
and scoped lint also pass. Existing product screenshots remain representative.
