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
}));

afterEach(() => {
  cleanup();
  mocks.status = null;
  mocks.loading = false;
  mocks.mobile = false;
  mocks.disconnect.mockReset();
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
  function twoHosts(): GitHubStatus {
    return {
      ...status("gh_cli"),
      hosts: [
        { github_host: "github.com", source: "gh_cli", login: "octocat", status: "active" },
        {
          github_host: "octo.ghe.com",
          source: "github_app_installation",
          installation_account_login: "octo-org",
          status: "active",
          personal_login: "bob",
        },
      ],
    };
  }

  function renderAutomation() {
    return render(
      <ToastProvider>
        <GitHubAutomationSettings workspaceId={WORKSPACE_ID} />
      </ToastProvider>,
    );
  }

  it("shows one card per host connection and an action to connect another host", () => {
    mocks.status = twoHosts();
    renderAutomation();

    const cards = screen.getAllByTestId("github-host-card");
    expect(cards.map((card) => card.getAttribute("data-host"))).toEqual([
      "github.com",
      "octo.ghe.com",
    ]);
    const enterprise = within(cards[1]!);
    expect(enterprise.getByText("octo-org")).toBeTruthy();
    expect(enterprise.getByText(/bob/)).toBeTruthy();
    expect(screen.getByRole("button", { name: "Connect another GitHub host" })).toBeTruthy();
  });

  it("disconnects only the enterprise host from its card", async () => {
    mocks.status = twoHosts();
    mocks.disconnect.mockResolvedValue({ disconnected: true });
    renderAutomation();

    const enterprise = within(screen.getAllByTestId("github-host-card")[1]!);
    fireEvent.click(enterprise.getByRole("button", { name: "Disconnect" }));

    await waitFor(() =>
      expect(mocks.disconnect).toHaveBeenCalledWith(WORKSPACE_ID, "octo.ghe.com"),
    );
  });

  it("moves enterprise card actions into a row menu on phones", () => {
    mocks.mobile = true;
    mocks.status = twoHosts();
    renderAutomation();

    const enterprise = within(screen.getAllByTestId("github-host-card")[1]!);
    expect(enterprise.queryByRole("button", { name: "Disconnect" })).toBeNull();
    expect(enterprise.getByRole("button", { name: "Actions for octo.ghe.com" })).toBeTruthy();
  });
});
