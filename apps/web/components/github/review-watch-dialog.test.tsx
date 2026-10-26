import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ReviewWatchDialog } from "./review-watch-dialog";

const ENTERPRISE_HOST = "octo.ghe.com";

const promptEditor = vi.hoisted(() => vi.fn());

const store = {
  workspaces: { activeId: "ws-1", items: [] },
  workflows: { items: [{ id: "workflow", name: "Delivery", hidden: false }] },
  agentProfiles: { items: [] },
  features: { dynamicAgentRouting: true },
  executors: { items: [] },
  prompts: { items: [], loaded: true, loading: false },
  githubStatus: {
    byWorkspaceId: {
      "ws-1": {
        status: {
          hosts: [
            { github_host: "github.com", source: "pat", status: "active" },
            { github_host: ENTERPRISE_HOST, source: "pat", status: "active" },
          ],
        },
      },
    },
  },
};

vi.mock("@/components/state-provider", () => ({
  useAppStore: (selector: (value: typeof store) => unknown) => selector(store),
  useOptionalAppStore: (selector: (value: typeof store) => unknown) => selector(store),
}));
vi.mock("@/hooks/domains/settings/use-settings-data", () => ({ useSettingsData: vi.fn() }));
vi.mock("@/hooks/use-workflows", () => ({ useWorkflows: vi.fn() }));
vi.mock("@/hooks/use-workflow-steps", () => ({
  useWorkflowSteps: () => ({ steps: [], loading: false }),
  stepPlaceholder: () => "Select step",
}));
vi.mock("@/components/github/repo-filter-selector", () => ({
  RepoFilterSelector: () => <div>Repository filters</div>,
}));
vi.mock("@/components/settings/settings-prompt-editor", () => ({
  SettingsPromptEditor: (props: Record<string, unknown>) => {
    promptEditor(props);
    return <div data-testid="review-watch-prompt-editor" />;
  },
}));

afterEach(() => {
  cleanup();
  promptEditor.mockReset();
});

describe("ReviewWatchDialog", () => {
  it("explains lifecycle retention for Auto and the Always delete override", () => {
    render(
      <ReviewWatchDialog
        open
        onOpenChange={vi.fn()}
        watch={null}
        workspaceId="ws-1"
        onCreate={vi.fn()}
        onUpdate={vi.fn()}
      />,
    );

    expect(screen.getByText(/user engagement or enabled PR lifecycle prompts/i)).not.toBeNull();
    const cleanupField = screen.getByText("Cleanup behavior").parentElement;
    const cleanupSelect = cleanupField?.querySelector("button");
    if (!cleanupSelect) throw new Error("Cleanup policy selector was not rendered");
    fireEvent.click(cleanupSelect);
    fireEvent.click(screen.getByRole("option", { name: "Always delete" }));
    expect(screen.getByText(/Always delete.*overrides retention/i)).not.toBeNull();
  });

  it("uses the shared editor with placeholders and saved-prompt references", () => {
    render(
      <ReviewWatchDialog
        open
        onOpenChange={vi.fn()}
        watch={null}
        workspaceId="ws-1"
        onCreate={vi.fn()}
        onUpdate={vi.fn()}
      />,
    );

    expect(promptEditor).toHaveBeenCalledWith(
      expect.objectContaining({
        promptReferences: true,
        testId: "github-review-watch-prompt-editor",
        placeholders: expect.arrayContaining([expect.objectContaining({ key: "pr.title" })]),
      }),
    );
  });
});

describe("ReviewWatchDialog GitHub hosts", () => {
  it("keeps an enterprise watch on its host when saved", async () => {
    const onUpdate = vi.fn().mockResolvedValue(undefined);
    render(
      <ReviewWatchDialog
        open
        onOpenChange={vi.fn()}
        watch={{
          id: "watch-1",
          workspace_id: "ws-1",
          github_host: ENTERPRISE_HOST,
          workflow_id: "workflow",
          workflow_step_id: "step",
          repos: [],
          agent_profile_id: "",
          executor_profile_id: "",
          prompt: "Review it",
          review_scope: "user",
          custom_query: "type:pr",
          enabled: true,
          poll_interval_seconds: 300,
          cleanup_policy: "auto",
          last_polled_at: null,
          created_at: "",
          updated_at: "",
        }}
        workspaceId="ws-1"
        onCreate={vi.fn()}
        onUpdate={onUpdate}
      />,
    );

    expect(screen.getByTestId("github-watch-host-select").textContent).toContain(ENTERPRISE_HOST);
    fireEvent.click(screen.getByRole("button", { name: "Update" }));

    await vi.waitFor(() =>
      expect(onUpdate).toHaveBeenCalledWith(
        "watch-1",
        expect.objectContaining({ github_host: ENTERPRISE_HOST }),
      ),
    );
  });
});
