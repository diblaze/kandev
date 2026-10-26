"use client";

import { Label } from "@kandev/ui/label";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@kandev/ui/select";
import { useTranslation } from "react-i18next";
import { useGitHubHosts } from "@/hooks/domains/github/use-github-hosts";

/** GitHub host a watch polls; shown only when the workspace has two or more hosts. */
export function GitHubWatchHostField({
  workspaceId,
  value,
  onChange,
}: {
  workspaceId: string;
  value: string;
  onChange: (host: string) => void;
}) {
  const { t } = useTranslation();
  const hosts = useGitHubHosts(workspaceId);
  if (hosts.length < 2 && hosts.includes(value)) return null;
  const options = hosts.includes(value) ? hosts : [...hosts, value];
  return (
    <div className="space-y-1.5">
      <Label htmlFor="github-watch-host">{t("github:githubHost")}</Label>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger
          id="github-watch-host"
          className="w-full sm:w-[240px]"
          data-testid="github-watch-host-select"
        >
          <SelectValue />
        </SelectTrigger>
        <SelectContent>
          {options.map((host) => (
            <SelectItem key={host} value={host}>
              {host}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
}
