import { useCallback, useEffect, useState } from "react";
import {
  declareDocument,
  DocumentRequestError,
  fetchDocuments,
  rebuildContextPack,
  withdrawDocument,
  type DeclareInput,
  type DocumentsView,
  type RebuildInput,
  type WithdrawInput,
} from "../api/documents";

export type DocumentsStatus = "LOADING" | "READY" | "ERROR";

export interface DocumentsController {
  status: DocumentsStatus;
  view: DocumentsView | null;
  error: { code: string; message: string } | null;
  busy: boolean;
  reload: () => void;
  declare: (input: Omit<DeclareInput, "expected_version">) => Promise<boolean>;
  withdraw: (input: Omit<WithdrawInput, "expected_version">) => Promise<boolean>;
  rebuild: (input: RebuildInput) => Promise<boolean>;
}

function describe(error: unknown): { code: string; message: string } {
  if (error instanceof DocumentRequestError) {
    return { code: error.code, message: error.message.replace(`${error.code}: `, "") };
  }
  return { code: "DOCUMENTS_UNAVAILABLE", message: error instanceof Error ? error.message : "Document request failed." };
}

/** Owns the document baseline and Context Pack of one Project. */
export function useDocuments(projectID: string | null): DocumentsController {
  const [status, setStatus] = useState<DocumentsStatus>("LOADING");
  const [view, setView] = useState<DocumentsView | null>(null);
  const [error, setError] = useState<{ code: string; message: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const reload = useCallback(() => {
    if (!projectID) return;
    setStatus("LOADING");
    setError(null);
    fetchDocuments(projectID)
      .then((next) => { setView(next); setStatus("READY"); })
      .catch((cause: unknown) => { setView(null); setError(describe(cause)); setStatus("ERROR"); });
  }, [projectID]);

  useEffect(() => { reload(); }, [reload]);

  const apply = useCallback(async (operation: (version: number) => Promise<DocumentsView>): Promise<boolean> => {
    if (!view) return false;
    setBusy(true);
    setError(null);
    try {
      const next = await operation(view.current_version);
      setView(next);
      setStatus("READY");
      return true;
    } catch (cause: unknown) {
      setError(describe(cause));
      return false;
    } finally {
      setBusy(false);
    }
  }, [view]);

  return {
    status, view, error, busy, reload,
    declare: (input) => apply((version) => declareDocument(projectID ?? "", { ...input, expected_version: version })),
    withdraw: (input) => apply((version) => withdrawDocument(projectID ?? "", { ...input, expected_version: version })),
    rebuild: (input) => apply(() => rebuildContextPack(projectID ?? "", input)),
  };
}
