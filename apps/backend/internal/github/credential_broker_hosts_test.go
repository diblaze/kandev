package github

import (
	"context"
	"errors"
	"testing"
)

func newEnterprisePATCredentialBroker() *CredentialBroker {
	connection := &WorkspaceConnection{
		WorkspaceID: "workspace-1", Source: ConnectionSourcePAT, GitHubHost: enterpriseTestHost,
		Login: "hubot", Status: ConnectionStatusActive, CredentialGeneration: 3,
	}
	connections := &fakeConnectionReader{workspaces: map[string]*WorkspaceConnection{connection.WorkspaceID: connection}}
	resolver := NewCredentialResolver(connections, fakeAuthSecrets{
		WorkspacePATSecretKey(connection.WorkspaceID, enterpriseTestHost): "enterprise-secret",
	})
	return NewCredentialBroker(connections, resolver, &fakeBrokerAuthorizer{})
}

func TestCredentialBrokerHostsIssueAndResolveEnterpriseHost(t *testing.T) {
	broker := newEnterprisePATCredentialBroker()
	request := brokerLeaseRequest()
	request.Host = enterpriseTestHost

	lease, err := broker.Issue(context.Background(), request)
	if err != nil {
		t.Fatalf("Issue: %v", err)
	}
	credentialRequest := brokerCredentialRequest(lease.Token)
	credentialRequest.Host = enterpriseTestHost
	credential, err := broker.Resolve(context.Background(), credentialRequest)

	if err != nil || credential.Password != "enterprise-secret" {
		t.Fatalf("Resolve = %+v, err %v", credential, err)
	}
}

func TestCredentialBrokerHostsDenyHostWithoutConnection(t *testing.T) {
	broker := newEnterprisePATCredentialBroker()
	request := brokerLeaseRequest()
	request.Host = "ghe.example.com"

	_, err := broker.Issue(context.Background(), request)

	var notConnected *HostNotConnectedError
	if !errors.As(err, &notConnected) || notConnected.Host != "ghe.example.com" {
		t.Fatalf("Issue for unconnected host: err = %v", err)
	}
}
