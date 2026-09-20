import { useCallback, useEffect, useState } from "react";
import {
  addEnvironment,
  editEnvironment,
  fetchTopology,
  reorderEnvironments,
  restoreEnvironment,
  retireEnvironment,
  TopologyRequestError,
  type AddEnvironmentInput,
  type EditEnvironmentInput,
  type TopologyState,
} from "../api/topology";

export type TopologyStatus = "LOADING" | "READY" | "ERROR";

export interface TopologyController {
  status: TopologyStatus;
  state: TopologyState | null;
  error: { code: string; message: string } | null;
  busy: boolean;
  reload: () => void;
  add: (input: Omit<AddEnvironmentInput, "expected_version">) => Promise<boolean>;
  edit: (environmentID: string, input: Omit<EditEnvironmentInput, "expected_version">) => Promise<boolean>;
  reorder: (order: string[], reason: string) => Promise<boolean>;
  retire: (environmentID: string, reason: string) => Promise<boolean>;
  restore: (environmentID: string, reason: string) => Promise<boolean>;
}

function describe(error: unknown): { code: string; message: string } {
  if (error instanceof TopologyRequestError) {
    return { code: error.code, message: error.message.replace(`${error.code}: `, "") };
  }
  return { code: "TOPOLOGY_UNAVAILABLE", message: error instanceof Error ? error.message : "Topology request failed." };
}

/**
 * Owns the versioned topology of one Project. Each mutation sends the
 * current version as expected_version; a 409 surfaces as an error and the
 * caller reloads rather than retrying blindly.
 */
export function useTopology(projectID: string | null): TopologyController {
  const [status, setStatus] = useState<TopologyStatus>("LOADING");
  const [state, setState] = useState<TopologyState | null>(null);
  const [error, setError] = useState<{ code: string; message: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const reload = useCallback(() => {
    if (!projectID) return;
    setStatus("LOADING");
    setError(null);
    fetchTopology(projectID)
      .then((next) => { setState(next); setStatus("READY"); })
      .catch((cause: unknown) => { setState(null); setError(describe(cause)); setStatus("ERROR"); });
  }, [projectID]);

  useEffect(() => { reload(); }, [reload]);

  const run = useCallback(async (operation: (version: number) => Promise<TopologyState>): Promise<boolean> => {
    if (!state) return false;
    setBusy(true);
    setError(null);
    try {
      const next = await operation(state.current_version);
      setState(next);
      setStatus("READY");
      return true;
    } catch (cause: unknown) {
      setError(describe(cause));
      return false;
    } finally {
      setBusy(false);
    }
  }, [state]);

  return {
    status, state, error, busy, reload,
    add: (input) => run((version) => addEnvironment(projectID ?? "", { ...input, expected_version: version })),
    edit: (environmentID, input) => run((version) => editEnvironment(projectID ?? "", environmentID, { ...input, expected_version: version })),
    reorder: (order, reason) => run((version) => reorderEnvironments(projectID ?? "", { order, reason, expected_version: version })),
    retire: (environmentID, reason) => run((version) => retireEnvironment(projectID ?? "", environmentID, { reason, expected_version: version })),
    restore: (environmentID, reason) => run((version) => restoreEnvironment(projectID ?? "", environmentID, { reason, expected_version: version })),
  };
}
