package handlers

import (
	"fmt"
	"testing"

	"github.com/kandev/kandev/internal/github"
	ws "github.com/kandev/kandev/pkg/websocket"
)

func TestClassifyAddBranchErrorTreatsUnconnectedGitHubHostAsValidation(t *testing.T) {
	err := fmt.Errorf("resolve repository: %w", &github.HostNotConnectedError{Host: "ghe.example.com"})

	if got := classifyAddBranchError(err); got != ws.ErrorCodeValidation {
		t.Fatalf("classifyAddBranchError() = %q, want %q", got, ws.ErrorCodeValidation)
	}
}
