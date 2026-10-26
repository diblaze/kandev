import { cleanup, fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "@/components/toast-provider";
import type { GitHubStatus } from "@/lib/types/github";
import { GitHubAutomationSettings, GitHubPersonalSettings } from "./github-status";

const WORKSPACE_ID = "workspace-1";

const mocks = vi.hoisted(() => ({
  status: null as GitHubStatus | null,
  loading: false,
  mobile: false,
  disconnect: vi.fn(),
  hostStatus: null as GitHubStatus | null,
  hostLoading: false,
  hostRefresh: vi.fn(),
  hostStatusCalls: [] as Array<[string, string]>,
}));
vi.mock("@/hooks/use-responsive-breakpoint", () => ({
  useResponsiveBreakpoint: () => ({ isMobile: mocks.mobile }),
}));
vi.mock("@/hooks/domains/github/use-github-app-registrations", () => ({
  useGitHubAppRegistrations: () => ({ registrations: [] }),
}));
vi.mock("@/hooks/domains/github/use-task-git-credentials", () => ({
  useTaskGitCredentials: () => ({ mode: "managed", loading: false, error: false, save: vi.fn() }),
}));
vi.mock("@/lib/api/domains/github-api", async (importOriginal) => ({
  ...(await importOriginal<typeof import("@/lib/api/domains/github-api")>()),
  disconnectGitHubWorkspace: mocks.disconnect,
}));
vi.mock("@/hooks/domains/github/use-github-status", () => ({
  useGitHubStatus: () => ({
    status: mocks.status,
    loaded: true,
    loading: mocks.loading,
    refresh: vi.fn(),
  }),
  useGitHubHostStatus: (workspaceId: string, host: string) => {
    mocks.hostStatusCalls.push([workspaceId, host]);
    return { status: mocks.hostStatus, loading: mocks.hostLoading, refresh: mocks.hostRefresh };
  },
}));

afterEach(() => {
  cleanup();
  mocks.status = null;
  mocks.loading = false;
  mocks.mobile = false;
  mocks.disconnect.mockReset();
  mocks.hostStatus = null;
  mocks.hostLoading = false;
  mocks.hostRefresh.mockReset();
  mocks.hostStatusCalls = [];
});

function renderPersonal() {
  return render(
    <ToastProvider>
      <GitHubPersonalSettings workspaceId={WORKSPACE_ID} />
    </ToastProvider>,
  );
}

function status(source: "pat" | "gh_cli" | "github_app_installation"): GitHubStatus {
  return {
    workspace_id: WORKSPACE_ID,
    automation: {
      workspace_id: WORKSPACE_ID,
      source,
      github_host: "github.com",
      login: "octocat",
      status: "active",
      credential_generation: 1,
    },
    personal: null,
    app_available: source === "github_app_installation",
    effective_personal_actor: {
      kind: "human",
      source,
      login: "octocat",
      workspace_id: WORKSPACE_ID,
    },
    authenticated: true,
    username: "octocat",
    auth_method: source,
    token_configured: source === "pat",
    required_scopes: [],
  };
}

describe("GitHubPersonalSettings", () => {
  it.each(["pat", "gh_cli"] as const)("does not separate the shared identity for %s", (source) => {
    mocks.status = status(source);
    renderPersonal();
    expect(screen.queryByText("My GitHub identity")).toBeNull();
    expect(screen.queryByTestId("github-personal-identity")).toBeNull();
    expect(screen.queryByRole("button", { name: /Connect identity/ })).toBeNull();
  });

  it("offers a separate identity only for App automation", () => {
    mocks.status = status("github_app_installation");
    renderPersonal();
    expect(screen.getByRole("button", { name: /Connect identity/ })).toBeTruthy();
    expect(screen.getByText(/never given to managed agents/)).toBeTruthy();
  });

  it("keeps the current identity visible while refreshing", () => {
    mocks.status = status("github_app_installation");
    mocks.loading = true;
    renderPersonal();

    expect(screen.getByTestId("github-personal-identity")).toBeTruthy();
    expect(screen.getByText("octocat", { exact: true })).toBeTruthy();
  });
});

describe("GitHubAutomationSettings hosts", () => {
  const ENTERPRISE = "octo.ghe.com";

  function twoHosts(): GitHubStatus {
    return {
      ...status("gh_cli"),
      hosts: [
        { github_host: "github.com", source: "gh_cli", login: "octocat", status: "active" },
        { github_host: ENTERPRISE, source: "pat", login: "octo-bot", status: "active" },
      ],
    };
  }

  function enterpriseStatus(): GitHubStatus {
    const base = status("pat");
    return {
      ...twoHosts(),
      automation: {
        ...base.automation!,
        github_host: ENTERPRISE,
        login: "octo-bot",
        actor: { kind: "human", source: "pat", login: "octo-bot", workspace_id: WORKSPACE_ID },
      },
      personal: {
        workspace_id: WORKSPACE_ID,
        user_id: "user-1",
        github_host: ENTERPRISE,
        app_registration_id: "registration-1",
        github_user_id: 7,
        login: "bob",
        status: "active",
        access_expires_at: "2026-12-01T00:00:00Z",
        credential_generation: 1,
      },
    };
  }

  function renderAutomation() {
    return render(
      <ToastProvider>
        <GitHubAutomationSettings workspaceId={WORKSPACE_ID} />
      </ToastProvider>,
    );
  }

  function enterpriseCard() {
    return within(screen.getAllByTestId("github-host-card")[1]!);
  }

  it("shows one card per host connection and an action to connect another host", () => {
    mocks.status = twoHosts();
    mocks.hostStatus = enterpriseStatus();
    renderAutomation();

    const cards = screen.getAllByTestId("github-host-card");
    expect(cards.map((card) => card.getAttribute("data-host"))).toEqual(["github.com", ENTERPRISE]);
    expect(mocks.hostStatusCalls).toContainEqual([WORKSPACE_ID, ENTERPRISE]);
    const enterprise = enterpriseCard();
    expect(enterprise.getByText("octo-bot", { exact: true })).toBeTruthy();
    expect(enterprise.getByText(/Kandev-managed operations act as/)).toBeTruthy();
    expect(enterprise.getByText("Personal: bob")).toBeTruthy();
    expect(enterprise.getByRole("button", { name: "Change connection" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "Connect another GitHub host" })).toBeTruthy();
  });

  it("shows the workspace-wide task access summary once", () => {
    mocks.status = twoHosts();
    mocks.hostStatus = enterpriseStatus();
    renderAutomation();

    expect(screen.getAllByTestId("github-task-access-summary")).toHaveLength(1);
    expect(enterpriseCard().queryByTestId("github-task-access-summary")).toBeNull();
  });

  it("refreshes the enterprise host status from its card", () => {
    mocks.status = twoHosts();
    mocks.hostStatus = enterpriseStatus();
    renderAutomation();

    fireEvent.click(enterpriseCard().getByRole("button", { name: "Refresh GitHub connection" }));
    expect(mocks.hostRefresh).toHaveBeenCalledTimes(1);
  });

  it("disconnects only the enterprise host from its card", async () => {
    mocks.status = twoHosts();
    mocks.hostStatus = enterpriseStatus();
    mocks.disconnect.mockResolvedValue({ disconnected: true });
    renderAutomation();

    fireEvent.click(enterpriseCard().getByRole("button", { name: "Disconnect" }));

    await waitFor(() => expect(mocks.disconnect).toHaveBeenCalledWith(WORKSPACE_ID, ENTERPRISE));
  });

  it("offers a retry when the enterprise host status could not be loaded", () => {
    mocks.status = twoHosts();
    renderAutomation();

    const enterprise = enterpriseCard();
    expect(
      enterprise.getByText(`Could not load the connection status for ${ENTERPRISE}.`),
    ).toBeTruthy();
    fireEvent.click(enterprise.getByRole("button", { name: "Refresh GitHub connection" }));
    expect(mocks.hostRefresh).toHaveBeenCalledTimes(1);
  });

  it("keeps enterprise card actions inline on phones like github.com", () => {
    mocks.mobile = true;
    mocks.status = twoHosts();
    mocks.hostStatus = enterpriseStatus();
    renderAutomation();

    expect(enterpriseCard().getByRole("button", { name: "Disconnect" })).toBeTruthy();
  });
});
