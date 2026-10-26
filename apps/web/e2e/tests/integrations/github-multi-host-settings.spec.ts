import { test, expect } from "../../fixtures/test-base";
import { waitForHttp } from "../../helpers/causal-waits";
import { KanbanPage } from "../../pages/kanban-page";

const ENTERPRISE_HOST = "octo.ghe.com";
const UNCONNECTED_HOST = "ghe.example.com";

test.describe("GitHub multi-host settings", () => {
  test("shows one card per host and disconnects only the chosen host", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    await apiClient.mockGitHubReset();
    await apiClient.mockGitHubSetWorkspaceConnection(seedData.workspaceId, {
      source: "gh_cli",
      status: "active",
      login: "alice",
    });
    await apiClient.mockGitHubSetWorkspaceConnection(
      seedData.workspaceId,
      { source: "pat", status: "active", login: "octo-bot" },
      ENTERPRISE_HOST,
    );

    await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/integrations/github`);
    const cards = testPage.getByTestId("github-host-card");
    await expect(cards).toHaveCount(2);
    await expect(cards.nth(0)).toHaveAttribute("data-host", "github.com");
    const enterprise = cards.nth(1);
    await expect(enterprise).toHaveAttribute("data-host", ENTERPRISE_HOST);
    await expect(enterprise).toContainText("octo-bot");
    await expect(testPage.getByTestId("github-connect-another-host")).toBeVisible();

    const disconnected = waitForHttp(
      testPage,
      "DELETE",
      /\/api\/v1\/github\/workspace-connection$/,
      {
        predicate: (response) =>
          new URL(response.url()).searchParams.get("host") === ENTERPRISE_HOST,
      },
    );
    await enterprise.getByRole("button", { name: "Disconnect" }).click();
    await disconnected;

    await expect(cards).toHaveCount(1);
    await expect(cards.nth(0)).toHaveAttribute("data-host", "github.com");
  });

  test("connects another host with a token on that host", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    await apiClient.mockGitHubReset();
    await apiClient.mockGitHubSetWorkspaceConnection(seedData.workspaceId, {
      source: "gh_cli",
      status: "active",
      login: "alice",
    });

    await testPage.goto(`/settings/workspaces/${seedData.workspaceId}/integrations/github`);
    await testPage.getByTestId("github-connect-another-host").click();
    const dialog = testPage.getByTestId("github-connection-desktop");
    const host = dialog.getByLabel("GitHub host");
    await expect(host).toHaveValue("github.com");
    await host.fill(ENTERPRISE_HOST);
    await dialog.locator("#github-workspace-token").fill("ghp_e2e_enterprise");

    const saved = waitForHttp(testPage, "PUT", /\/api\/v1\/github\/workspace-connection$/);
    await dialog.getByRole("button", { name: "Save changes" }).click();
    const request = (await saved).request();
    expect(request.postDataJSON()).toMatchObject({ source: "pat", host: ENTERPRISE_HOST });

    await expect(
      testPage.locator(`[data-testid="github-host-card"][data-host="${ENTERPRISE_HOST}"]`),
    ).toBeVisible();
  });

  test("recognizes pull-request URLs on connected hosts and flags other hosts", async ({
    testPage,
    apiClient,
    seedData,
  }) => {
    await apiClient.mockGitHubReset();
    await apiClient.mockGitHubSetWorkspaceConnection(seedData.workspaceId, {
      source: "gh_cli",
      status: "active",
      login: "alice",
    });
    await apiClient.mockGitHubSetWorkspaceConnection(
      seedData.workspaceId,
      { source: "pat", status: "active", login: "octo-bot" },
      ENTERPRISE_HOST,
    );
    const task = await apiClient.createTask(seedData.workspaceId, "Enterprise PR link", {
      workflow_id: seedData.workflowId,
      workflow_step_id: seedData.startStepId,
      repository_ids: [seedData.repositoryId],
    });

    const kanban = new KanbanPage(testPage);
    await kanban.goto();
    await kanban.openTaskContextMenu(task.id);
    await testPage.getByTestId("task-context-link").focus();
    await testPage.keyboard.press("ArrowRight");
    await testPage.getByTestId("task-context-link-github-pull-request").click();

    const dialog = testPage.getByRole("dialog", { name: "Link GitHub pull request" });
    const input = dialog.getByTestId("task-github-pr-input");
    const save = dialog.getByTestId("task-github-pr-submit");

    await input.fill(`https://${UNCONNECTED_HOST}/org/api/pull/7`);
    const notice = dialog.getByTestId("github-host-not-connected");
    await expect(notice).toContainText(`${UNCONNECTED_HOST} is not connected.`);
    await expect(dialog.getByTestId("github-host-not-connected-settings")).toHaveAttribute(
      "href",
      `/settings/workspaces/${seedData.workspaceId}/integrations/github`,
    );
    await expect(save).toBeDisabled();

    await input.fill(`https://${ENTERPRISE_HOST}/org/api/pull/7`);
    await expect(notice).toBeHidden();
    await expect(save).toBeEnabled();
  });
});
