import { describe, expect, it } from "vitest";
import { issuePayload } from "./task-github-issue-dialog";

describe("issuePayload", () => {
  it("names the enterprise host of the inferred repository for a bare issue number", () => {
    expect(issuePayload("#8", { host: "octo.ghe.com", owner: "org", repo: "api" })).toEqual({
      issue: "#8",
      owner: "org",
      repo: "api",
      number: 8,
      host: "octo.ghe.com",
    });
  });

  it("sends a github.com repository without a host", () => {
    expect(issuePayload("8", { owner: "org", repo: "api" })).toEqual({
      issue: "8",
      owner: "org",
      repo: "api",
      number: 8,
    });
  });

  it("sends a full URL as is so the backend reads its host", () => {
    expect(issuePayload(" https://octo.ghe.com/org/api/issues/8 ", null)).toEqual({
      issue: "https://octo.ghe.com/org/api/issues/8",
    });
  });
});
