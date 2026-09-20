import { useState, type FormEvent } from "react";
import { currentVersion, standardTypes, type FieldChange, type TopologyEnvironment, type TopologyVersion } from "../api/topology";
import type { TopologyController } from "../state/useTopology";
import { StatusBadge } from "./StatusBadge";

interface EnvironmentTopologyProps {
  controller: TopologyController;
}

function formatValue(value: unknown): string {
  if (Array.isArray(value)) return value.length === 0 ? "∅" : value.join(", ");
  if (value === null || value === undefined || value === "") return "∅";
  return String(value);
}

function ChangeSummary({ version }: { version: TopologyVersion }) {
  const { change } = version;
  const fields = change.fields_changed ?? [];
  return (
    <div className="topology-change" data-testid="topology-last-change">
      <div className="topology-change-heading">
        <span className="topology-version">v{version.version}</span>
        <span className="topology-op">{change.operation}{change.environment_id ? ` · ${change.environment_id}` : ""}</span>
        <StatusBadge status={change.material ? "MATERIAL" : "INFORMATIONAL"} />
        <span className="muted">{version.origin} · {version.actor} · {version.created_at}</span>
      </div>
      {fields.length > 0 && (
        <ul className="topology-diff">
          {fields.map((field: FieldChange, index) => (
            <li key={`${field.field}-${index}`}>
              <code>{field.field}</code>: {formatValue(field.from)} → {formatValue(field.to)}
              {field.material ? <span className="reason"> material</span> : <span className="muted"> display-only</span>}
            </li>
          ))}
        </ul>
      )}
      <p className="topology-reason"><span className="muted">Reason:</span> {change.reason}</p>
      <p className="topology-impact"><span className="muted">Impact:</span> {change.impact}</p>
    </div>
  );
}

interface EnvironmentFormValues {
  id: string;
  display_name: string;
  type: string;
  sequence: string;
  target_ref: string;
  owner: string;
  required_evidence: string;
  approver_policy: string;
  reason: string;
}

const emptyForm: EnvironmentFormValues = {
  id: "", display_name: "", type: "other", sequence: "", target_ref: "", owner: "", required_evidence: "", approver_policy: "", reason: "",
};

function fromEnvironment(environment: TopologyEnvironment): EnvironmentFormValues {
  return {
    id: environment.id,
    display_name: environment.display_name,
    type: environment.type,
    sequence: String(environment.sequence),
    target_ref: environment.target_ref,
    owner: environment.owner,
    required_evidence: environment.required_evidence.join(", "),
    approver_policy: environment.approver_policy,
    reason: "",
  };
}

function splitEvidence(value: string): string[] {
  return value.split(",").map((item) => item.trim()).filter(Boolean);
}

interface EnvironmentFormProps {
  mode: "add" | "edit";
  initial: EnvironmentFormValues;
  busy: boolean;
  onCancel: () => void;
  onSubmit: (values: EnvironmentFormValues) => void;
}

function EnvironmentForm({ mode, initial, busy, onCancel, onSubmit }: EnvironmentFormProps) {
  const [values, setValues] = useState<EnvironmentFormValues>(initial);
  const update = (field: keyof EnvironmentFormValues) => (event: { target: { value: string } }) =>
    setValues((previous) => ({ ...previous, [field]: event.target.value }));
  const submit = (event: FormEvent) => {
    event.preventDefault();
    onSubmit(values);
  };
  const isProduction = mode === "edit" && initial.type === "production";
  return (
    <form className="topology-form" data-testid={`environment-form-${mode}`} onSubmit={submit}>
      <div className="topology-form-grid">
        <label>Environment id
          <input name="id" value={values.id} onChange={update("id")} disabled={mode === "edit"} placeholder="uat" required={mode === "add"} />
        </label>
        <label>Display name
          <input name="display_name" value={values.display_name} onChange={update("display_name")} placeholder="UAT" />
        </label>
        <label>Standard type
          <select name="type" value={values.type} onChange={update("type")} disabled={isProduction}>
            {standardTypes.map((type) => <option key={type} value={type}>{type}</option>)}
          </select>
        </label>
        <label>Sequence
          <input name="sequence" type="number" min={1} value={values.sequence} onChange={update("sequence")} placeholder="append" />
        </label>
        <label>Target ref
          <input name="target_ref" value={values.target_ref} onChange={update("target_ref")} placeholder="cloud-run/service-uat" />
        </label>
        <label>Owner
          <input name="owner" value={values.owner} onChange={update("owner")} placeholder="qa-lead" />
        </label>
        <label>Required evidence (comma separated)
          <input name="required_evidence" value={values.required_evidence} onChange={update("required_evidence")} placeholder="test, build, smoke" />
        </label>
        <label>Approver policy
          <input name="approver_policy" value={values.approver_policy} onChange={update("approver_policy")} placeholder="distinct human approver" />
        </label>
      </div>
      <label className="topology-reason-field">Reason for this version
        <input name="reason" value={values.reason} onChange={update("reason")} placeholder="why the topology changes" required />
      </label>
      {isProduction && <p className="muted">Production: display name and owner may change; standard type is protected in P0.</p>}
      <div className="topology-form-actions">
        <button type="submit" className="button-primary" disabled={busy}>{mode === "add" ? "Create version with new environment" : "Save as new version"}</button>
        <button type="button" className="button-secondary" onClick={onCancel} disabled={busy}>Cancel</button>
      </div>
    </form>
  );
}

export function EnvironmentTopology({ controller }: EnvironmentTopologyProps) {
  const { status, state, error, busy } = controller;
  const [adding, setAdding] = useState(false);
  const [editingID, setEditingID] = useState<string | null>(null);
  const [quickReason, setQuickReason] = useState("");
  const [quickReasonMissing, setQuickReasonMissing] = useState(false);
  const [showHistory, setShowHistory] = useState(false);

  const requireQuickReason = (): string | null => {
    if (quickReason.trim() === "") {
      setQuickReasonMissing(true);
      return null;
    }
    setQuickReasonMissing(false);
    return quickReason.trim();
  };

  const current = state ? currentVersion(state) : undefined;
  const environments = [...(current?.environments ?? [])].sort((a, b) => a.sequence - b.sequence);

  const move = async (environmentID: string, direction: -1 | 1) => {
    const reason = requireQuickReason();
    if (!reason) return;
    const order = environments.map((environment) => environment.id);
    const index = order.indexOf(environmentID);
    const target = index + direction;
    if (index < 0 || target < 0 || target >= order.length) return;
    [order[index], order[target]] = [order[target], order[index]];
    if (await controller.reorder(order, reason)) setQuickReason("");
  };

  const retire = async (environmentID: string) => {
    const reason = requireQuickReason();
    if (!reason) return;
    if (await controller.retire(environmentID, reason)) setQuickReason("");
  };

  const restore = async (environmentID: string) => {
    const reason = requireQuickReason();
    if (!reason) return;
    if (await controller.restore(environmentID, reason)) setQuickReason("");
  };

  const submitAdd = async (values: EnvironmentFormValues) => {
    const ok = await controller.add({
      id: values.id.trim(),
      display_name: values.display_name.trim() || undefined,
      type: values.type,
      sequence: values.sequence ? Number(values.sequence) : undefined,
      target_ref: values.target_ref.trim() || undefined,
      owner: values.owner.trim() || undefined,
      required_evidence: splitEvidence(values.required_evidence),
      approver_policy: values.approver_policy.trim() || undefined,
      reason: values.reason.trim(),
    });
    if (ok) setAdding(false);
  };

  const submitEdit = async (environmentID: string, values: EnvironmentFormValues) => {
    const ok = await controller.edit(environmentID, {
      display_name: values.display_name.trim(),
      type: values.type,
      sequence: values.sequence ? Number(values.sequence) : undefined,
      target_ref: values.target_ref.trim(),
      owner: values.owner.trim(),
      required_evidence: splitEvidence(values.required_evidence),
      approver_policy: values.approver_policy.trim(),
      reason: values.reason.trim(),
    });
    if (ok) setEditingID(null);
  };

  return (
    <section className="panel topology-panel" aria-labelledby="topology-heading" data-testid="environment-topology">
      <div className="section-heading">
        <div>
          <p className="eyebrow">ENVIRONMENT TOPOLOGY</p>
          <h2 id="topology-heading">Promotion path</h2>
        </div>
        <div className="topology-heading-meta">
          {current && (
            <>
              <StatusBadge status="VERSIONED" label={`Topology v${current.version}`} />
              <span className="muted" title={current.config_hash}>config {current.config_hash.slice(7, 19)}</span>
            </>
          )}
          <span className="muted">Every change creates an immutable version; nothing is hard-deleted</span>
        </div>
      </div>

      {status === "LOADING" && <p className="muted" data-testid="topology-loading"><span className="spinner" aria-hidden="true" />Loading topology…</p>}
      {status === "ERROR" && error && (
        <div className="workspace-state error-state" data-testid="topology-error">
          <strong>{error.code}</strong> — {error.message}
          <div><button type="button" className="button-secondary" onClick={controller.reload}>Reload topology</button></div>
        </div>
      )}

      {state && current && (
        <>
          {status === "READY" && error && (
            <div className="topology-inline-error" data-testid="topology-error" role="alert">
              <strong>{error.code}</strong> — {error.message}
              {error.code === "TOPOLOGY_VERSION_CONFLICT" && (
                <button type="button" className="button-secondary" onClick={controller.reload}>Reload topology</button>
              )}
            </div>
          )}

          <ChangeSummary version={current} />

          <div className="topology-quick-reason">
            <label>Reason for move / retire / restore
              <input
                data-testid="topology-quick-reason"
                value={quickReason}
                onChange={(event) => { setQuickReason(event.target.value); if (event.target.value.trim()) setQuickReasonMissing(false); }}
                placeholder="required before a quick action"
              />
            </label>
            {quickReasonMissing && <span className="reason" data-testid="topology-reason-missing">A reason is required for every topology change.</span>}
          </div>

          <table className="topology-table">
            <thead>
              <tr>
                <th>#</th><th>Environment</th><th>Type</th><th>Target</th><th>Owner</th><th>Required evidence</th><th>Status</th><th>Actions</th>
              </tr>
            </thead>
            <tbody>
              {environments.map((environment, index) => {
                const retired = environment.status === "RETIRED";
                const production = environment.type === "production";
                return (
                  <tr key={environment.id} data-testid={`environment-row-${environment.id}`} className={retired ? "topology-row-retired" : ""}>
                    <td>{environment.sequence}</td>
                    <td>
                      <div className="topology-env-name">{environment.display_name}</div>
                      <code className="topology-env-id">{environment.id}</code>
                    </td>
                    <td>{environment.type}</td>
                    <td><code>{environment.target_ref || "∅"}</code></td>
                    <td>{environment.owner || <span className="muted">UNDECLARED</span>}</td>
                    <td>{environment.required_evidence.length === 0 ? <span className="muted">none</span> : environment.required_evidence.join(", ")}</td>
                    <td>
                      <div className="topology-status">
                        <StatusBadge status={environment.status} />
                        {environment.protection && <StatusBadge status="BLOCKED" label="READ_ONLY · BLOCKED IN P0" />}
                        {environment.action && <span className="muted">{environment.action}</span>}
                      </div>
                    </td>
                    <td>
                      <div className="topology-actions">
                        <button type="button" className="button-icon" aria-label={`Move ${environment.id} up`} disabled={busy || index === 0} onClick={() => void move(environment.id, -1)}>↑</button>
                        <button type="button" className="button-icon" aria-label={`Move ${environment.id} down`} disabled={busy || index === environments.length - 1} onClick={() => void move(environment.id, 1)}>↓</button>
                        <button type="button" className="button-secondary" disabled={busy || retired} onClick={() => { setAdding(false); setEditingID(environment.id); }}>Edit</button>
                        {retired ? (
                          <button type="button" className="button-secondary" disabled={busy} onClick={() => void restore(environment.id)}>Restore</button>
                        ) : (
                          <button
                            type="button"
                            className="button-secondary"
                            disabled={busy || production}
                            title={production ? "Production cannot be retired in P0" : undefined}
                            onClick={() => void retire(environment.id)}
                          >Retire</button>
                        )}
                      </div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>

          {editingID && environments.find((environment) => environment.id === editingID) && (
            <EnvironmentForm
              key={`edit-${editingID}-${current.version}`}
              mode="edit"
              initial={fromEnvironment(environments.find((environment) => environment.id === editingID)!)}
              busy={busy}
              onCancel={() => setEditingID(null)}
              onSubmit={(values) => void submitEdit(editingID, values)}
            />
          )}

          {adding ? (
            <EnvironmentForm mode="add" initial={emptyForm} busy={busy} onCancel={() => setAdding(false)} onSubmit={(values) => void submitAdd(values)} />
          ) : (
            <div className="topology-form-actions">
              <button type="button" className="button-primary" disabled={busy} onClick={() => { setEditingID(null); setAdding(true); }}>Add environment</button>
            </div>
          )}

          <div className="topology-invalidations" data-testid="topology-invalidations">
            <h3>Decisions marked STALE by topology changes</h3>
            {state.invalidations.length === 0 ? (
              <p className="muted">No decision has been invalidated by a topology change.</p>
            ) : (
              <ul>
                {state.invalidations.slice().reverse().slice(0, 8).map((invalidation, index) => (
                  <li key={`${invalidation.decision_id}-${invalidation.topology_version}-${index}`}>
                    <StatusBadge status={invalidation.status} /> <strong>{invalidation.decision_id}</strong> <span className="muted">{invalidation.reason}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>

          <div className="topology-history">
            <button type="button" className="button-link" onClick={() => setShowHistory((value) => !value)}>
              {showHistory ? "Hide" : "Show"} version history ({state.versions.length})
            </button>
            {showHistory && (
              <ol className="topology-history-list" data-testid="topology-history">
                {state.versions.slice().reverse().map((version) => (
                  <li key={version.version}>
                    <span className="topology-version">v{version.version}</span> {version.change.operation}
                    {version.change.environment_id ? ` ${version.change.environment_id}` : ""} · {version.change.material ? "material" : "informational"} · {version.actor} · <span className="muted">{version.change.reason}</span>
                  </li>
                ))}
              </ol>
            )}
          </div>
        </>
      )}
    </section>
  );
}
