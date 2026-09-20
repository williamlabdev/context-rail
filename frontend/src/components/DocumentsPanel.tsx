import { useState, type FormEvent } from "react";
import { documentStages, type ContextPack, type DocumentReadiness, type DocumentSource } from "../api/documents";
import type { DocumentsController } from "../state/useDocuments";
import { StatusBadge } from "./StatusBadge";

interface DocumentsPanelProps {
  controller: DocumentsController;
}

const short = (hash?: string): string => (hash ? hash.replace(/^sha256:/, "").slice(0, 12) : "∅");

function ReadinessRow({ readiness, testid }: { readiness: Record<string, DocumentReadiness>; testid: string }) {
  return (
    <div className="readiness-grid" data-testid={testid}>
      {documentStages.map((stage) => {
        const entry = readiness[stage] ?? { status: "UNKNOWN", reasons: [] };
        return (
          <div className="readiness-card" key={stage} data-testid={`${testid}-${stage}`}>
            <div className="readiness-card-heading"><span>{stage}</span><StatusBadge status={entry.status} /></div>
            {entry.reasons.length > 0 && <ul className="reason-list">{entry.reasons.map((reason) => <li key={reason} className="reason">{reason}</li>)}</ul>}
          </div>
        );
      })}
    </div>
  );
}

function SourceTable({ sources, testid, onWithdraw, busy }: { sources: DocumentSource[]; testid: string; onWithdraw?: (path: string) => void; busy?: boolean }) {
  return (
    <table className="topology-table" data-testid={testid}>
      <thead><tr><th>Source</th><th>Kind</th><th>Required for</th><th>Version</th><th>Content hash</th><th>Status</th>{onWithdraw && <th />}</tr></thead>
      <tbody>
        {sources.map((source) => (
          <tr key={source.path} data-testid={`${testid}-${source.path}`}>
            <td><code>{source.path}</code> {source.origin === "operator" && <span className="document-kind">DECLARED</span>}{!source.source_of_truth && <span className="document-kind">DERIVED</span>}</td>
            <td>{source.kind}</td>
            <td>{source.required_for.join(", ") || <span className="muted">—</span>}</td>
            <td><code>{source.version}</code></td>
            <td>{source.content_hash ? <code title={source.content_hash}>{short(source.content_hash)}</code> : <span className="reason">no content — not synthesized</span>}</td>
            <td><StatusBadge status={source.status} />{source.stale_sources && source.stale_sources.length > 0 && <span className="reason"> stale: {source.stale_sources.join(", ")}</span>}</td>
            {onWithdraw && <td>{source.origin === "operator" && <button type="button" className="button-secondary" disabled={busy} onClick={() => onWithdraw(source.path)}>Withdraw</button>}</td>}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function DeclareForm({ busy, onCancel, onSubmit }: { busy: boolean; onCancel: () => void; onSubmit: DocumentsController["declare"] }) {
  const [values, setValues] = useState({ path: "", kind: "", reason: "" });
  const [stages, setStages] = useState<string[]>(["decision"]);
  const toggle = (stage: string) => setStages((previous) => (previous.includes(stage) ? previous.filter((entry) => entry !== stage) : [...previous, stage]));
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const ok = await onSubmit({ path: values.path.trim(), kind: values.kind.trim(), required_for: stages, reason: values.reason.trim() });
    if (ok) onCancel();
  };
  return (
    <form className="topology-form" data-testid="document-declare-form" onSubmit={(event) => void submit(event)}>
      <p className="eyebrow">DECLARE REQUIRED DOCUMENT · the file is verified, never created; a missing file blocks the stages it is required for</p>
      <div className="topology-form-grid">
        <label>Repository path<input name="path" value={values.path} onChange={(event) => setValues({ ...values, path: event.target.value })} required placeholder="docs/operations/runbook.md" /></label>
        <label>Kind<input name="kind" value={values.kind} onChange={(event) => setValues({ ...values, kind: event.target.value })} placeholder="runbook" /></label>
        <label className="span-2">Reason<input name="reason" value={values.reason} onChange={(event) => setValues({ ...values, reason: event.target.value })} required /></label>
      </div>
      <fieldset className="option-list">
        <legend>Required for</legend>
        {documentStages.map((stage) => (
          <label key={stage} className="checkbox-line">
            <input type="checkbox" name={`stage-${stage}`} checked={stages.includes(stage)} onChange={() => toggle(stage)} /> {stage}
          </label>
        ))}
      </fieldset>
      <div className="topology-form-actions">
        <button type="submit" className="button-primary" disabled={busy || stages.length === 0}>Declare as new baseline version</button>
        <button type="button" className="button-secondary" onClick={onCancel} disabled={busy}>Cancel</button>
      </div>
    </form>
  );
}

function PackCard({ pack, drift }: { pack: ContextPack; drift: { stale: boolean; reason?: string } }) {
  return (
    <div className="decision-card receipt-card" data-testid="context-pack">
      <div className="topology-change-heading">
        <strong>Context Pack</strong>
        <code>{pack.pack_id}</code>
        <StatusBadge status={pack.status} />
        {drift.stale && <StatusBadge status="STALE" />}
        <span className="muted">generated {pack.generated_at} by {pack.generated_by} · baseline v{pack.baseline_version} · topology v{pack.topology_version ?? "∅"} · snapshot <code title={pack.source_snapshot_hash}>{short(pack.source_snapshot_hash)}</code> · pack hash <code title={pack.pack_hash}>{short(pack.pack_hash)}</code></span>
      </div>
      {drift.stale && <div className="topology-inline-error" data-testid="context-pack-drift" role="alert"><strong>STALE</strong> — {drift.reason}</div>}
      <p className="topology-reason">{pack.reason}</p>
      {pack.missing.length > 0 && (
        <div data-testid="context-pack-missing">
          <p className="eyebrow">MISSING · listed, not filled in</p>
          <ul className="reason-list">
            {pack.missing.map((entry) => <li key={entry.path} className="reason" data-testid={`context-pack-missing-${entry.path}`}><code>{entry.path}</code> ({entry.kind}) · required for {entry.required_for.join(", ")} → blocks {entry.readiness_impact.join(", ") || "nothing"}</li>)}
          </ul>
        </div>
      )}
      <p className="eyebrow">SOURCES · path / version / content hash as observed at generation</p>
      <SourceTable sources={pack.sources} testid="context-pack-sources" />
      <p className="eyebrow">DOCUMENT READINESS · as recorded in the pack</p>
      <ReadinessRow readiness={pack.readiness} testid="context-pack-readiness" />
      <p className="topology-reason"><span className="muted">decision refs:</span> {pack.decision_refs.length > 0 ? pack.decision_refs.join(", ") : "none"}</p>
      <ul className="reason-list" data-testid="context-pack-limitations">
        {pack.limitations.map((limitation) => <li key={limitation} className="muted">{limitation}</li>)}
      </ul>
    </div>
  );
}

export function DocumentsPanel({ controller }: DocumentsPanelProps) {
  const { status, view, error, busy } = controller;
  const [declaring, setDeclaring] = useState(false);
  const [rebuildReason, setRebuildReason] = useState("");
  const [withdrawReason, setWithdrawReason] = useState("");
  const [withdrawMissing, setWithdrawMissing] = useState(false);
  const [showAudit, setShowAudit] = useState(false);
  return (
    <section className="panel topology-panel" aria-labelledby="documents-heading" data-testid="documents-panel">
      <div className="section-heading">
        <div>
          <p className="eyebrow">PROVENANCE · DOCUMENT BASELINE</p>
          <h2 id="documents-heading">Documents / AI Context</h2>
        </div>
        {view && (
          <span className="muted topology-heading-meta" data-testid="documents-meta">
            Baseline v{view.current_version} · declared context <StatusBadge status={view.declared_context.status} /> · live context <StatusBadge status={view.live_context_status} /> · consumer read-only
          </span>
        )}
      </div>
      {status === "LOADING" && <p className="muted" data-testid="documents-loading"><span className="spinner" aria-hidden="true" />Loading document baseline…</p>}
      {error && (
        <div className="topology-inline-error" data-testid="documents-error" role="alert">
          <strong>{error.code}</strong> — {error.message}
          {status === "ERROR" && <button type="button" className="button-secondary" onClick={controller.reload}>Reload</button>}
        </div>
      )}
      {view && (
        <>
          <p className="eyebrow">DOCUMENT READINESS · live — a missing required document is NEEDS_INPUT for its stages, never filled in by the AI</p>
          <ReadinessRow readiness={view.readiness} testid="document-readiness" />
          <p className="eyebrow">DECLARED DOCUMENTS · manifest + operator declarations, verified against the repository now</p>
          <SourceTable sources={view.documents} testid="documents" busy={busy} onWithdraw={(path) => {
            const reason = withdrawReason.trim();
            if (!reason) { setWithdrawMissing(true); return; }
            setWithdrawMissing(false);
            void controller.withdraw({ path, reason }).then((ok) => { if (ok) setWithdrawReason(""); });
          }} />
          <div className="topology-form-actions">
            {!declaring && <button type="button" className="button-primary" disabled={busy} onClick={() => setDeclaring(true)}>Declare required document</button>}
            <label className="topology-reason-field">Reason for withdraw<input name="withdraw_reason" data-testid="withdraw-reason" value={withdrawReason} onChange={(event) => { setWithdrawReason(event.target.value); setWithdrawMissing(false); }} placeholder="required before pressing Withdraw" /></label>
          </div>
          {withdrawMissing && <p className="reason" data-testid="withdraw-reason-missing">A reason is required to withdraw a declaration.</p>}
          {declaring && <DeclareForm busy={busy} onCancel={() => setDeclaring(false)} onSubmit={controller.declare} />}

          <form className="topology-form" data-testid="context-pack-rebuild-form" onSubmit={(event) => { event.preventDefault(); void controller.rebuild({ reason: rebuildReason.trim() }).then((ok) => { if (ok) setRebuildReason(""); }); }}>
            <p className="eyebrow">CONTEXT PACK · rebuild from the sources as they are now; missing sources make it PARTIAL</p>
            <div className="topology-form-grid">
              <label className="span-2">Reason<input name="rebuild_reason" value={rebuildReason} onChange={(event) => setRebuildReason(event.target.value)} required /></label>
            </div>
            <div className="topology-form-actions"><button type="submit" className="button-primary" disabled={busy}>Rebuild Context Pack</button></div>
          </form>
          {view.latest_pack ? <PackCard pack={view.latest_pack} drift={view.pack_drift} /> : <p className="muted" data-testid="context-pack-none">No rebuilt Context Pack yet; the consumer's own <code>{view.declared_context.path}</code> is {view.declared_context.status}.</p>}

          <button type="button" className="button-secondary" onClick={() => setShowAudit((previous) => !previous)}>{showAudit ? "Hide audit" : `Show audit (${view.audit.length})`}</button>
          {showAudit && (
            <ul className="reason-list" data-testid="documents-audit">
              {view.audit.slice().reverse().map((event) => <li key={event.sequence}><code>{event.action}</code> {event.object} · {event.actor} · {event.at} · {event.reason}</li>)}
            </ul>
          )}
        </>
      )}
    </section>
  );
}
