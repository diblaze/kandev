import { cleanup, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ToastProvider } from "@/components/toast-provider";
import { GitHubAppCreateForm } from "./github-app-create-form";

afterEach(() => cleanup());

function renderForm() {
  const registrations = {
    mutating: false,
    startManifest: vi.fn().mockResolvedValue(null),
  };
  render(
    <ToastProvider>
      <GitHubAppCreateForm workspaceId="workspace-1" registrations={registrations as never} />
    </ToastProvider>,
  );
  fireEvent.change(screen.getByLabelText("Name in Kandev"), { target: { value: "Automation" } });
  fireEvent.change(screen.getByLabelText("GitHub owner login"), { target: { value: "octo-org" } });
  fireEvent.change(screen.getByLabelText("Public Kandev URL"), {
    target: { value: "https://kandev.example.com" },
  });
  return registrations;
}

describe("GitHubAppCreateForm hosts", () => {
  it("starts the App manifest on the named GitHub host", async () => {
    const registrations = renderForm();
    fireEvent.change(screen.getByLabelText("GitHub host"), { target: { value: "octo.ghe.com" } });
    fireEvent.click(screen.getByRole("button", { name: /Prepare App on GitHub/ }));

    await waitFor(() =>
      expect(registrations.startManifest).toHaveBeenCalledWith(
        expect.objectContaining({ github_host: "octo.ghe.com", owner_login: "octo-org" }),
      ),
    );
  });

  it("defaults the host to github.com and rejects one that is not a bare hostname", async () => {
    const registrations = renderForm();
    const host = screen.getByLabelText("GitHub host") as HTMLInputElement;
    expect(host.value).toBe("github.com");

    fireEvent.change(host, { target: { value: "https://octo.ghe.com" } });
    fireEvent.click(screen.getByRole("button", { name: /Prepare App on GitHub/ }));

    expect(await screen.findByText(/bare hostname/)).toBeTruthy();
    expect(registrations.startManifest).not.toHaveBeenCalled();
  });
});
