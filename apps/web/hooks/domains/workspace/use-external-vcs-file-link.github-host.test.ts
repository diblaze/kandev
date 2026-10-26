import { describe, expect, it } from "vitest";
import type { TaskPR } from "@/lib/types/github";
import type { Repository } from "@/lib/types/http";
import { githubPRMatches } from "./use-external-vcs-file-link";

const repository = {
  id: "repo-1",
  provider: "github",
  provider_host: "https://octo.ghe.com",
  provider_owner: "acme",
  provider_name: "web",
} as Repository;

function pr(overrides: Partial<TaskPR>): TaskPR {
  return { owner: "acme", repo: "web", ...overrides } as TaskPR;
}

describe("githubPRMatches hosts", () => {
  it("matches a pull request without a repository ID only on the repository host", () => {
    expect(githubPRMatches(pr({ github_host: "octo.ghe.com" }), repository)).toBe(true);
    expect(githubPRMatches(pr({ github_host: "github.com" }), repository)).toBe(false);
    expect(githubPRMatches(pr({}), repository)).toBe(false);
  });

  it("treats a pull request without a host as github.com", () => {
    expect(
      githubPRMatches(pr({}), { ...repository, provider_host: "https://github.com" } as Repository),
    ).toBe(true);
  });
});
