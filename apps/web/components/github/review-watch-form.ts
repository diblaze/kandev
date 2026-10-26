import { DEFAULT_GITHUB_HOST } from "@/lib/github/host";
import { DEFAULT_REVIEW_WATCH_PROMPT } from "@/components/github/review-watch-placeholders";
import type { CleanupPolicy, RepoFilter, ReviewWatch } from "@/lib/types/github";

export const QUERY_TEMPLATES = {
  meAndTeams: "type:pr state:open review-requested:@me -is:draft",
  me: "type:pr state:open user-review-requested:@me -is:draft",
} as const;

export type FormState = {
  workspaceId: string;
  githubHost: string;
  selectedRepos: RepoFilter[];
  allRepos: boolean;
  workflowId: string;
  workflowStepId: string;
  agentProfileId: string;
  executorProfileId: string;
  prompt: string;
  customQuery: string;
  enabled: boolean;
  pollInterval: number;
  cleanupPolicy: CleanupPolicy;
};

export function makeDefaultForm(workspaceId: string): FormState {
  return {
    workspaceId,
    githubHost: DEFAULT_GITHUB_HOST,
    selectedRepos: [],
    allRepos: false,
    workflowId: "",
    workflowStepId: "",
    agentProfileId: "",
    executorProfileId: "",
    prompt: DEFAULT_REVIEW_WATCH_PROMPT,
    customQuery: QUERY_TEMPLATES.meAndTeams,
    enabled: true,
    pollInterval: 300,
    cleanupPolicy: "auto",
  };
}

export function formStateFromWatch(watch: ReviewWatch): FormState {
  const hasRepos = watch.repos && watch.repos.length > 0;
  return {
    workspaceId: watch.workspace_id,
    githubHost: watch.github_host || DEFAULT_GITHUB_HOST,
    selectedRepos: hasRepos ? watch.repos : [],
    allRepos: !hasRepos,
    workflowId: watch.workflow_id,
    workflowStepId: watch.workflow_step_id,
    agentProfileId: watch.agent_profile_id,
    executorProfileId: watch.executor_profile_id,
    prompt: watch.prompt || DEFAULT_REVIEW_WATCH_PROMPT,
    customQuery: watch.custom_query || QUERY_TEMPLATES.meAndTeams,
    enabled: watch.enabled,
    pollInterval: watch.poll_interval_seconds,
    cleanupPolicy: watch.cleanup_policy ?? "auto",
  };
}
