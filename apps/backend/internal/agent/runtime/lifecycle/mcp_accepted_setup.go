package lifecycle

import (
	"context"

	"github.com/kandev/kandev/internal/agent/mcpconfig"
	"go.uber.org/zap"
)

func (e *AgentExecution) rememberMCPSetup(result *mcpconfig.EffectiveMCPResolution) {
	e.mcpSetupMu.Lock()
	defer e.mcpSetupMu.Unlock()
	e.mcpSetup = result.Setup
	e.mcpSetupRevision = result.DesiredRevision
	e.mcpSetupCaptured = result.Setup != nil || result.DesiredRevision != 0
}

func (e *AgentExecution) capturedMCPSetup() (*mcpconfig.MCPSetupSnapshot, int64, bool) {
	e.mcpSetupMu.Lock()
	defer e.mcpSetupMu.Unlock()
	return e.mcpSetup, e.mcpSetupRevision, e.mcpSetupCaptured
}

func (m *Manager) recordAcceptedMCPSetupOperation(ctx context.Context, execution *AgentExecution, setup *mcpconfig.MCPSetupSnapshot, revision int64) {
	attempt := ""
	client, release := execution.AcquireAgentCtlClient()
	if client != nil {
		attempt = client.GetLastAttachmentAttemptID()
	}
	release()
	if err := m.saveAppliedSessionMCP(ctx, execution.SessionID, revision, attempt, setup); err != nil {
		m.logger.Warn("failed to record accepted MCP setup", zap.String("session_id", execution.SessionID), zap.Error(err))
	}
}
