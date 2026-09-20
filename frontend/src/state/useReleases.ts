import { useCallback, useEffect, useState } from "react";
import {
  approveRelease,
  createRelease,
  fetchReleases,
  recordBuild,
  recordDeployment,
  ReleaseRequestError,
  type ApprovalInput,
  type BuildInput,
  type CreateReleaseInput,
  type DeploymentInput,
  type ReleaseView,
} from "../api/releases";

export type ReleasesStatus = "LOADING" | "READY" | "ERROR";

export interface ReleasesController {
  status: ReleasesStatus;
  releases: ReleaseView[];
  selectedID: string | null;
  selected: ReleaseView | null;
  error: { code: string; message: string } | null;
  busy: boolean;
  reload: () => void;
  select: (releaseID: string | null) => void;
  create: (input: CreateReleaseInput) => Promise<boolean>;
  build: (releaseID: string, input: BuildInput) => Promise<boolean>;
  approve: (releaseID: string, input: ApprovalInput) => Promise<boolean>;
  deployment: (releaseID: string, input: DeploymentInput) => Promise<boolean>;
}

function describe(error: unknown): { code: string; message: string } {
  if (error instanceof ReleaseRequestError) {
    return { code: error.code, message: error.message.replace(`${error.code}: `, "") };
  }
  return { code: "RELEASES_UNAVAILABLE", message: error instanceof Error ? error.message : "Release request failed." };
}

/** Owns the release ledger of one Project; reloads when refreshKey changes. */
export function useReleases(projectID: string | null, refreshKey: unknown = null): ReleasesController {
  const [status, setStatus] = useState<ReleasesStatus>("LOADING");
  const [releases, setReleases] = useState<ReleaseView[]>([]);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [error, setError] = useState<{ code: string; message: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const reload = useCallback(() => {
    if (!projectID) return;
    setStatus("LOADING");
    setError(null);
    fetchReleases(projectID)
      .then((list) => { setReleases(list.releases); setStatus("READY"); })
      .catch((cause: unknown) => { setReleases([]); setError(describe(cause)); setStatus("ERROR"); });
  }, [projectID]);

  useEffect(() => { setSelectedID(null); }, [projectID]);
  useEffect(() => { reload(); }, [reload, refreshKey]);

  const apply = useCallback(async (operation: () => Promise<ReleaseView>): Promise<boolean> => {
    setBusy(true);
    setError(null);
    try {
      const view = await operation();
      setReleases((previous) => {
        const index = previous.findIndex((entry) => entry.release.release_id === view.release.release_id);
        if (index < 0) return [...previous, view];
        const next = previous.slice();
        next[index] = view;
        return next;
      });
      setSelectedID(view.release.release_id);
      setStatus("READY");
      return true;
    } catch (cause: unknown) {
      setError(describe(cause));
      return false;
    } finally {
      setBusy(false);
    }
  }, []);

  const selected = releases.find((entry) => entry.release.release_id === selectedID) ?? null;
  return {
    status, releases, selectedID, selected, error, busy, reload,
    select: (releaseID) => { setSelectedID(releaseID); setError(null); },
    create: (input) => apply(() => createRelease(projectID ?? "", input)),
    build: (releaseID, input) => apply(() => recordBuild(projectID ?? "", releaseID, input)),
    approve: (releaseID, input) => apply(() => approveRelease(projectID ?? "", releaseID, input)),
    deployment: (releaseID, input) => apply(() => recordDeployment(projectID ?? "", releaseID, input)),
  };
}
