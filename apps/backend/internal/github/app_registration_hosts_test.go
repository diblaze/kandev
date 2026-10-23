package github

import (
	"context"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

type acceptingAppRegistrationRuntime struct{}

func (acceptingAppRegistrationRuntime) ValidateAppRegistrationRuntime(ResolvedDeploymentAppConfig) error {
	return nil
}
func (acceptingAppRegistrationRuntime) ApplyAppRegistrationRuntime(ResolvedDeploymentAppConfig) error {
	return nil
}
func (acceptingAppRegistrationRuntime) InvalidateAppRegistrationRuntime(string, int64) {}

type hostRecordingManifestConverter struct {
	conversion ManifestConversionResult
	hosts      []string
}

func (c *hostRecordingManifestConverter) Convert(_ context.Context, host, _ string) (ManifestConversionResult, error) {
	c.hosts = append(c.hosts, host)
	return c.conversion, nil
}

func newHostManifestLifecycle(t *testing.T) (*AppRegistrationLifecycleService, *Store, *hostRecordingManifestConverter) {
	t.Helper()
	store := newTestStore(t)
	seedConnectionWorkspaces(t, store, "workspace-1")
	converter := &hostRecordingManifestConverter{conversion: testManifestConversion(t, "acme", "Organization")}
	lifecycle := NewAppRegistrationLifecycleService(AppRegistrationLifecycleConfig{
		Repository: NewAppRegistrationRepository(store, newFakeConnectionSecrets()), Store: store,
		Runtime: acceptingAppRegistrationRuntime{}, Converter: converter,
		Resolver: manifestResolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("203.0.114.10")}, nil
		}),
		Random: strings.NewReader(strings.Repeat("m", oauthRandomBytes*3)),
	})
	return lifecycle, store, converter
}

func enterpriseManifestRequest(host string) AppRegistrationManifestStartRequest {
	return AppRegistrationManifestStartRequest{
		WorkspaceID: "workspace-1", DisplayName: "Enterprise App", GitHubHost: host,
		OwnerType: ManifestOwnerOrganization, OwnerLogin: "acme",
		Visibility: AppRegistrationVisibilityPrivate, PublicBaseURL: "https://kandev.example",
	}
}

func TestAppRegistrationManifestUsesRequestedHost(t *testing.T) {
	lifecycle, store, converter := newHostManifestLifecycle(t)
	ctx := context.Background()

	started, err := lifecycle.StartManifest(ctx, DefaultUserID, enterpriseManifestRequest("Octo.GHE.com"))
	if err != nil {
		t.Fatalf("StartManifest: %v", err)
	}
	registrationURL, err := url.Parse(started.RegistrationURL)
	if err != nil || registrationURL.Host != enterpriseTestHost ||
		registrationURL.Path != "/organizations/acme/settings/apps/new" {
		t.Fatalf("registration URL = %q", started.RegistrationURL)
	}
	if _, err := lifecycle.CompleteManifest(ctx, started.RegistrationID, AppRegistrationManifestCallback{
		State: started.State, Code: "one-time-code",
	}); err != nil {
		t.Fatalf("CompleteManifest: %v", err)
	}

	registration, err := store.GetAppRegistration(ctx, started.RegistrationID)
	if err != nil || registration == nil || registration.GitHubHost != enterpriseTestHost {
		t.Fatalf("stored registration = %+v, err %v", registration, err)
	}
	if len(converter.hosts) != 1 || converter.hosts[0] != enterpriseTestHost {
		t.Fatalf("conversion hosts = %v", converter.hosts)
	}
}

func TestAppRegistrationManifestRejectsInvalidHost(t *testing.T) {
	lifecycle, _, _ := newHostManifestLifecycle(t)

	if _, err := lifecycle.StartManifest(context.Background(), DefaultUserID,
		enterpriseManifestRequest("octo.ghe.com/evil")); err == nil {
		t.Fatal("StartManifest accepted an invalid host")
	}
}

func TestStoreAppRegistrationFlowHostUpgradeDefaultsToGitHubCom(t *testing.T) {
	store := newTestStore(t)
	seedConnectionWorkspaces(t, store, "workspace-1")
	legacyFlows := strings.Replace(appRegistrationTablesSQL[strings.Index(appRegistrationTablesSQL,
		"\tCREATE TABLE IF NOT EXISTS github_app_registration_flows"):strings.Index(appRegistrationTablesSQL,
		"\tCREATE TABLE IF NOT EXISTS github_app_import_preparations")],
		"\t\tgithub_host TEXT NOT NULL DEFAULT 'github.com',\n", "", 1)
	for _, statement := range []string{`DROP TABLE github_app_registration_flows`, legacyFlows} {
		if _, err := store.db.Exec(statement); err != nil {
			t.Fatalf("install legacy registration flows: %v", err)
		}
	}
	if _, err := store.db.Exec(`INSERT INTO github_app_registration_flows (
		state_hash, registration_id, workspace_id, user_id, owner_type, owner_login, display_name,
		visibility, public_base_url, manifest_revision, expires_at, created_at
	) VALUES ('state', 'registration-1', 'workspace-1', 'user-1', 'Organization', 'acme', 'App',
		'private', 'https://kandev.example', 1, ?, ?)`, legacyConnectionFixtureTime, legacyConnectionFixtureTime); err != nil {
		t.Fatalf("seed legacy registration flow: %v", err)
	}

	for range 2 {
		if _, err := NewStore(store.db, store.db); err != nil {
			t.Fatalf("upgrade registration flows: %v", err)
		}
	}

	flow, err := store.GetDeploymentAppRegistrationFlow(context.Background(), "state")
	if err != nil || flow == nil || flow.GitHubHost != defaultGitHubHost {
		t.Fatalf("upgraded flow = %+v, err %v", flow, err)
	}
}

func TestAppRegistrationImportVerifiesAgainstNamedHost(t *testing.T) {
	store := newTestStore(t)
	repository := NewAppRegistrationRepository(store, newFakeConnectionSecrets())
	request := validAppRegistrationImportRequest(t)
	request.GitHubHost = "Octo.GHE.com"
	var verifiedHost string
	importer := NewAppRegistrationImporter(repository, importPublicResolver(),
		func(host string, _ int64, _ []byte) (AppRegistrationVerifier, error) {
			verifiedHost = host
			return appRegistrationVerifierFunc(func(context.Context) (AuthenticatedApp, error) {
				app := authenticatedAppForImport(t, request)
				app.Events = []string{"push", "check_run"}
				return app, nil
			}), nil
		})

	registration, err := importer.Import(context.Background(), request)

	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if verifiedHost != enterpriseTestHost || registration.GitHubHost != enterpriseTestHost {
		t.Fatalf("verified host %q, registration host %q", verifiedHost, registration.GitHubHost)
	}
}

func TestAppRegistrationImportRejectsInvalidHost(t *testing.T) {
	store := newTestStore(t)
	request := validAppRegistrationImportRequest(t)
	request.GitHubHost = "octo.ghe.com:8443"
	importer := NewAppRegistrationImporter(NewAppRegistrationRepository(store, newFakeConnectionSecrets()),
		importPublicResolver(), func(string, int64, []byte) (AppRegistrationVerifier, error) {
			t.Fatal("invalid host reached verification")
			return nil, nil
		})

	if _, err := importer.Import(context.Background(), request); err == nil {
		t.Fatal("Import() accepted a host with a port")
	}
}
