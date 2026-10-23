package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	taskmodels "github.com/kandev/kandev/internal/task/models"
)

const (
	taskMetaIssueURL      = "issue_url"
	taskMetaIssueNumber   = "issue_number"
	taskMetaIssueOwner    = "issue_owner"
	taskMetaIssueRepo     = "issue_repo"
	taskMetaIssueLinked   = "github_issue_linked"
	githubProviderName    = "github"
	githubIssuePathMarker = "/issues/"
	gitlabDotComHost      = "gitlab.com"
)

// TaskIssueStore is the task-facing dependency used by GitHub task links.
// It supplies task repositories for PR association validation and task
// metadata for issue links. The backendapp adapter routes metadata writes
// through task.Service so the normal task.updated event is published.
type TaskIssueStore interface {
	GetTask(ctx context.Context, taskID string) (*taskmodels.Task, error)
	ListTaskRepositories(ctx context.Context, taskID string) ([]*taskmodels.TaskRepository, error)
	GetRepository(ctx context.Context, repositoryID string) (*taskmodels.Repository, error)
	UpdateTaskGitHubIssue(ctx context.Context, taskID string, link *taskmodels.TaskGitHubIssueLink) (*taskmodels.Task, error)
}

type LinkTaskIssueRequest struct {
	Issue  string `json:"issue"`
	Host   string `json:"host,omitempty"`
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	Number int    `json:"number"`
}

type TaskIssueLinkResponse struct {
	TaskID      string `json:"task_id"`
	TaskTitle   string `json:"task_title"`
	Owner       string `json:"owner"`
	Repo        string `json:"repo"`
	IssueNumber int    `json:"issue_number"`
	IssueURL    string `json:"issue_url"`
	IssueTitle  string `json:"issue_title"`
}

func (s *Service) SetTaskIssueStore(store TaskIssueStore) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.taskIssueStore = store
}

func (s *Service) getTaskIssueStore() TaskIssueStore {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.taskIssueStore
}

func (s *Service) LinkTaskIssue(ctx context.Context, taskID string, req LinkTaskIssueRequest) (*TaskIssueLinkResponse, error) {
	if s.client == nil {
		return nil, ErrNoClient
	}
	return s.linkTaskIssueWithClient(ctx, s.client, taskID, req)
}

// LinkTaskIssueForUser derives workspace ownership from the task before
// resolving a personal-read credential. A task without ownership fails closed.
func (s *Service) LinkTaskIssueForUser(
	ctx context.Context, userID, taskID string, req LinkTaskIssueRequest,
) (*TaskIssueLinkResponse, error) {
	store := s.getTaskIssueStore()
	if store == nil {
		return nil, errStoreUnavailable
	}
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if task == nil || strings.TrimSpace(task.WorkspaceID) == "" {
		return nil, ErrGitHubWorkspaceRequired
	}
	ref, number, err := resolveIssueReference(req)
	if err != nil {
		return nil, err
	}
	if err := s.ensureRepositoryInWorkspaceScope(ctx, task.WorkspaceID, ref.host, ref.owner, ref.name); err != nil {
		return nil, err
	}
	resolved, err := s.resolvePersonalReadClient(ctx, task.WorkspaceID, userID, ref)
	if err != nil {
		return nil, err
	}
	return s.linkTaskIssueResolved(ctx, resolved.Client, store, task, taskID, ref, number)
}

func (s *Service) linkTaskIssueWithClient(
	ctx context.Context, client Client, taskID string, req LinkTaskIssueRequest,
) (*TaskIssueLinkResponse, error) {
	store := s.getTaskIssueStore()
	if store == nil {
		return nil, errStoreUnavailable
	}
	ref, number, err := resolveIssueReference(req)
	if err != nil {
		return nil, err
	}
	task, err := store.GetTask(ctx, taskID)
	if err != nil {
		return nil, err
	}
	return s.linkTaskIssueResolved(ctx, client, store, task, taskID, ref, number)
}

func (s *Service) linkTaskIssueResolved(
	ctx context.Context,
	client Client,
	store TaskIssueStore,
	task *taskmodels.Task,
	taskID string,
	ref repoRef,
	number int,
) (*TaskIssueLinkResponse, error) {
	issue, err := client.GetIssue(ctx, ref.owner, ref.name, number)
	if err != nil {
		return nil, err
	}
	if err := s.validateIssueTaskRepository(ctx, store, taskID, repoRef{ref.host, issue.RepoOwner, issue.RepoName}); err != nil {
		return nil, err
	}
	link := &taskmodels.TaskGitHubIssueLink{
		URL: issue.HTMLURL, Number: issue.Number, Owner: issue.RepoOwner, Repo: issue.RepoName,
	}
	if _, err := store.UpdateTaskGitHubIssue(context.WithoutCancel(ctx), taskID, link); err != nil {
		return nil, err
	}
	return taskIssueResponse(taskID, task.Title, issue), nil
}

// ListWorkspaceTaskIssues returns persisted GitHub issue links for one workspace.
func (s *Service) ListWorkspaceTaskIssues(ctx context.Context, workspaceID string) (map[string]TaskIssueLinkResponse, error) {
	if s.store == nil {
		return nil, errStoreUnavailable
	}
	rows, err := s.store.ListTaskIssueMetadataByWorkspaceID(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]TaskIssueLinkResponse)
	for _, row := range rows {
		link, ok := taskIssueLinkFromMetadata(row)
		if ok {
			result[row.TaskID] = link
		}
	}
	return result, nil
}

func (s *Service) UnlinkTaskIssue(ctx context.Context, taskID string) error {
	store := s.getTaskIssueStore()
	if store == nil {
		return errStoreUnavailable
	}
	_, err := store.GetTask(ctx, taskID)
	if err != nil {
		return err
	}
	_, err = store.UpdateTaskGitHubIssue(context.WithoutCancel(ctx), taskID, nil)
	return err
}

// resolveIssueReference returns the issue repository and number; an issue URL
// names its own host, any other reference is on the request host.
func resolveIssueReference(req LinkTaskIssueRequest) (repoRef, int, error) {
	host, owner, repo, number := "", req.Owner, req.Repo, req.Number
	if req.Issue != "" {
		var err error
		if host, owner, repo, number, err = parseIssueReference(req.Issue, req.Owner, req.Repo); err != nil {
			return repoRef{}, 0, err
		}
	} else if owner == "" || repo == "" || number <= 0 {
		return repoRef{}, 0, ErrInvalidIssueReference
	}
	if host == "" {
		normalized, err := normalizeGitHubHost(req.Host)
		if err != nil {
			return repoRef{}, 0, fmt.Errorf("%w: %w", ErrInvalidIssueReference, err)
		}
		host = normalized
	}
	return repoRef{host, owner, repo}, number, nil
}

// parseIssueReference returns the issue of a bare number on the default
// repository (host empty) or of an https://<host>/<owner>/<repo>/issues/<n>
// URL on a GitHub host.
func parseIssueReference(input, defaultOwner, defaultRepo string) (string, string, string, int, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return "", "", "", 0, ErrInvalidIssueReference
	}
	if n, err := strconv.Atoi(strings.TrimPrefix(trimmed, "#")); err == nil && n > 0 {
		if defaultOwner == "" || defaultRepo == "" {
			return "", "", "", 0, fmt.Errorf("%w: owner and repo are required for issue numbers", ErrInvalidIssueReference)
		}
		return "", defaultOwner, defaultRepo, n, nil
	}
	host, err := WebHostFromURL(trimmed)
	if err != nil || host == gitlabDotComHost {
		return "", "", "", 0, fmt.Errorf("%w: expected a GitHub issue URL", ErrInvalidIssueReference)
	}
	u, err := url.Parse(trimmed)
	if err != nil || !strings.Contains(u.Path, githubIssuePathMarker) {
		return "", "", "", 0, fmt.Errorf("%w: expected a GitHub issue URL", ErrInvalidIssueReference)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "issues" {
		return "", "", "", 0, fmt.Errorf("%w: expected /owner/repo/issues/number", ErrInvalidIssueReference)
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return "", "", "", 0, fmt.Errorf("%w: invalid issue number", ErrInvalidIssueReference)
	}
	return host, parts[0], parts[1], number, nil
}

func (s *Service) validateIssueTaskRepository(ctx context.Context, store TaskIssueStore, taskID string, ref repoRef) error {
	taskRepos, err := store.ListTaskRepositories(ctx, taskID)
	if err != nil {
		return err
	}
	if len(taskRepos) == 0 {
		return nil
	}
	for _, taskRepo := range taskRepos {
		entity, err := store.GetRepository(ctx, taskRepo.RepositoryID)
		if err != nil {
			return err
		}
		if entity == nil {
			continue
		}
		if !strings.EqualFold(entity.Provider, githubProviderName) {
			continue
		}
		host, hostErr := RepositoryGitHubHost(entity.ProviderHost)
		if hostErr == nil && host == ref.host &&
			strings.EqualFold(entity.ProviderOwner, ref.owner) &&
			strings.EqualFold(entity.ProviderName, ref.name) {
			return nil
		}
	}
	return ErrIssueRepositoryMismatch
}

func taskIssueLinkFromMetadata(row taskIssueMetadataRow) (TaskIssueLinkResponse, bool) {
	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(row.Metadata), &metadata); err != nil {
		return TaskIssueLinkResponse{}, false
	}
	issueURL, ok := metadata[taskMetaIssueURL].(string)
	if !ok || issueURL == "" {
		return TaskIssueLinkResponse{}, false
	}
	metadataNumber, ok := positiveMetadataInt(metadata[taskMetaIssueNumber])
	if !ok {
		return TaskIssueLinkResponse{}, false
	}
	_, owner, repo, issueNumber, err := parseIssueReference(issueURL, "", "")
	if err != nil || issueNumber != metadataNumber {
		return TaskIssueLinkResponse{}, false
	}
	return TaskIssueLinkResponse{
		TaskID:      row.TaskID,
		TaskTitle:   row.TaskTitle,
		Owner:       owner,
		Repo:        repo,
		IssueNumber: issueNumber,
		IssueURL:    issueURL,
	}, true
}

func positiveMetadataInt(value interface{}) (int, bool) {
	switch number := value.(type) {
	case int:
		return number, number > 0
	case float64:
		integer := int(number)
		return integer, integer > 0 && float64(integer) == number
	default:
		return 0, false
	}
}

func taskIssueResponse(taskID, taskTitle string, issue *Issue) *TaskIssueLinkResponse {
	return &TaskIssueLinkResponse{
		TaskID:      taskID,
		TaskTitle:   taskTitle,
		Owner:       issue.RepoOwner,
		Repo:        issue.RepoName,
		IssueNumber: issue.Number,
		IssueURL:    issue.HTMLURL,
		IssueTitle:  issue.Title,
	}
}
