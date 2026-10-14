import { test } from "../../fixtures/test-base";
import { checkMCPPendingChanges } from "../../helpers/mcp-pending-changes";

// @covers AC-AGENTS-WORKSPACE-MCP-CONFIGURATION-006.15, AC-AGENTS-WORKSPACE-MCP-CONFIGURATION-006.16, AC-AGENTS-WORKSPACE-MCP-CONFIGURATION-006.17, AC-AGENTS-WORKSPACE-MCP-CONFIGURATION-006.18
test("names saved additions, deleted removals and updates across application states", async ({
  testPage,
  apiClient,
  seedData,
}) => {
  test.setTimeout(150_000);
  await checkMCPPendingChanges(testPage, apiClient, seedData, false);
});
