import { test } from "../../fixtures/test-base";
import { checkMCPPendingChanges } from "../../helpers/mcp-pending-changes";

// @covers AC-AGENTS-WORKSPACE-MCP-CONFIGURATION-006.19
test("phone shows the same pending actions with reachable choices and stable search", async ({
  testPage,
  apiClient,
  seedData,
}) => {
  test.setTimeout(150_000);
  await checkMCPPendingChanges(testPage, apiClient, seedData, true);
});
