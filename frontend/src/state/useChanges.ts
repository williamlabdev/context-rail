import { useCallback, useEffect, useState } from "react";
import {
  ChangeRequestError,
  compileWorkOrder,
  createChange,
  decideCandidate,
  decideChange,
  fetchChanges,
  submitCandidate,
  supplyInputs,
  type CandidateDecisionInput,
  type ChangeView,
  type CreateChangeInput,
  type DecideInput,
  type InputsPatch,
  type SubmitCandidateInput,
  type WorkOrderInput,
} from "../api/changes";

export type ChangesStatus = "LOADING" | "READY" | "ERROR";

export interface ChangesController {
  status: ChangesStatus;
  changes: ChangeView[];
  selectedID: string | null;
  selected: ChangeView | null;
  error: { code: string; message: string } | null;
  busy: boolean;
  reload: () => void;
  select: (changeID: string | null) => void;
  create: (input: CreateChangeInput) => Promise<boolean>;
  inputs: (changeID: string, input: InputsPatch) => Promise<boolean>;
  decide: (changeID: string, input: DecideInput) => Promise<boolean>;
  workOrder: (changeID: string, input: WorkOrderInput) => Promise<boolean>;
  submitCandidate: (changeID: string, input: SubmitCandidateInput) => Promise<boolean>;
  decideCandidate: (changeID: string, candidateID: string, input: CandidateDecisionInput) => Promise<boolean>;
}

function describe(error: unknown): { code: string; message: string } {
  if (error instanceof ChangeRequestError) {
    return { code: error.code, message: error.message.replace(`${error.code}: `, "") };
  }
  return { code: "CHANGES_UNAVAILABLE", message: error instanceof Error ? error.message : "Change request failed." };
}

/**
 * Owns the change ledger of one Project. `refreshKey` lets the caller force
 * a reload when something the ledger depends on (the topology) changed, so
 * STALE verdicts show up without a manual refresh.
 */
export function useChanges(projectID: string | null, refreshKey: unknown = null): ChangesController {
  const [status, setStatus] = useState<ChangesStatus>("LOADING");
  const [changes, setChanges] = useState<ChangeView[]>([]);
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [error, setError] = useState<{ code: string; message: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const reload = useCallback(() => {
    if (!projectID) return;
    setStatus("LOADING");
    setError(null);
    fetchChanges(projectID)
      .then((list) => { setChanges(list.changes); setStatus("READY"); })
      .catch((cause: unknown) => { setChanges([]); setError(describe(cause)); setStatus("ERROR"); });
  }, [projectID]);

  useEffect(() => { setSelectedID(null); }, [projectID]);
  useEffect(() => { reload(); }, [reload, refreshKey]);

  const apply = useCallback(async (operation: () => Promise<ChangeView>): Promise<boolean> => {
    setBusy(true);
    setError(null);
    try {
      const view = await operation();
      setChanges((previous) => {
        const index = previous.findIndex((entry) => entry.change.change_id === view.change.change_id);
        if (index < 0) return [...previous, view];
        const next = previous.slice();
        next[index] = view;
        return next;
      });
      setSelectedID(view.change.change_id);
      setStatus("READY");
      return true;
    } catch (cause: unknown) {
      setError(describe(cause));
      return false;
    } finally {
      setBusy(false);
    }
  }, []);

  const selected = changes.find((entry) => entry.change.change_id === selectedID) ?? null;
  return {
    status, changes, selectedID, selected, error, busy, reload,
    select: (changeID) => { setSelectedID(changeID); setError(null); },
    create: (input) => apply(() => createChange(projectID ?? "", input)),
    inputs: (changeID, input) => apply(() => supplyInputs(projectID ?? "", changeID, input)),
    decide: (changeID, input) => apply(() => decideChange(projectID ?? "", changeID, input)),
    workOrder: (changeID, input) => apply(() => compileWorkOrder(projectID ?? "", changeID, input)),
    submitCandidate: (changeID, input) => apply(() => submitCandidate(projectID ?? "", changeID, input)),
    decideCandidate: (changeID, candidateID, input) => apply(() => decideCandidate(projectID ?? "", changeID, candidateID, input)),
  };
}
