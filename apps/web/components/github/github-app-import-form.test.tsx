import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "@/components/toast-provider";
import type { PrepareGitHubAppImportResponse } from "@/lib/types/github";
import { GitHubAppImportForm } from "./github-app-import-form";

afterEach(() => cleanup());

const preparation: PrepareGitHubAppImportResponse = {
  registration_id: "registration-1",
  public_base_url: "https://kandev.example",
  manifest_callback_url: "https://kandev.example/manifest/callback",
  install_callback_url: "https://kandev.example/install/callback",
  personal_callback_url: "https://kandev.example/personal/callback",
  webhook_url: "https://kandev.example/webhook",
  setup_url: "",
  permissions: { contents: "write" },
  events: ["installation"],
  expires_at: "2026-07-22T00:00:00Z",
};

describe("GitHubAppImportForm", () => {
  it("clears imported secrets when setup starts over", async () => {
    const registrations = {
      mutating: false,
      prepareImport: vi.fn().mockResolvedValue(preparation),
      importRegistration: vi.fn(),
    };
    render(
      <ToastProvider>
        <GitHubAppImportForm
          workspaceId="workspace-1"
          registrations={registrations as never}
          onImported={vi.fn()}
        />
      </ToastProvider>,
    );

    fireEvent.change(screen.getByLabelText("Public Kandev URL"), {
      target: { value: "https://kandev.example" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Generate setup instructions" }));
    const clientSecret = await screen.findByLabelText("Client secret");
    fireEvent.change(clientSecret, { target: { value: "client-secret" } });
    fireEvent.change(screen.getByLabelText("Webhook secret"), {
      target: { value: "webhook-secret" },
    });
    fireEvent.change(screen.getByLabelText("Private key (.pem)"), {
      target: { value: "private-key" },
    });

    fireEvent.click(screen.getByRole("button", { name: "Start over" }));
    fireEvent.click(screen.getByRole("button", { name: "Generate setup instructions" }));

    await waitFor(() => {
      expect((screen.getByLabelText("Client secret") as HTMLInputElement).value).toBe("");
      expect((screen.getByLabelText("Webhook secret") as HTMLInputElement).value).toBe("");
      expect((screen.getByLabelText("Private key (.pem)") as HTMLTextAreaElement).value).toBe("");
    });
  });
});

describe("GitHubAppImportForm hosts", () => {
  it("verifies the App against the named GitHub host", async () => {
    const registrations = {
      mutating: false,
      prepareImport: vi.fn().mockResolvedValue(preparation),
      importRegistration: vi.fn().mockResolvedValue({ id: "registration-1" }),
    };
    render(
      <ToastProvider>
        <GitHubAppImportForm
          workspaceId="workspace-1"
          registrations={registrations as never}
          onImported={vi.fn()}
        />
      </ToastProvider>,
    );
    fireEvent.change(screen.getByLabelText("Public Kandev URL"), {
      target: { value: "https://kandev.example" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Generate setup instructions" }));
    const fields: Record<string, string> = {
      "GitHub host": "OCTO.ghe.com",
      "Name in Kandev": "Tenant automation",
      "GitHub App slug": "tenant-app",
      "GitHub owner login": "octo-org",
      "App ID": "42",
      "Client ID": "Iv1.tenant",
      "Client secret": "client-secret",
      "Webhook secret": "webhook-secret",
      "Private key (.pem)": "private-key",
    };
    await screen.findByLabelText("Client secret");
    for (const [label, value] of Object.entries(fields)) {
      fireEvent.change(screen.getByLabelText(label), { target: { value } });
    }
    fireEvent.click(screen.getByRole("button", { name: "Verify and import App" }));

    await waitFor(() =>
      expect(registrations.importRegistration).toHaveBeenCalledWith(
        expect.objectContaining({ github_host: "octo.ghe.com", slug: "tenant-app" }),
      ),
    );
  });
});
