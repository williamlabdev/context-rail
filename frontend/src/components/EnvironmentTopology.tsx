import { useState, type FormEvent } from "react";
import { currentVersion, standardTypes, type FieldChange, type TopologyEnvironment, type TopologyVersion } from "../api/topology";
import type { TopologyController } from "../state/useTopology";
import { StatusBadge } from "./StatusBadge";
import { useLocale } from "../i18n";

interface EnvironmentTopologyProps {
  controller: TopologyController;
}

function formatValue(value: unknown): string {
  if (Array.isArray(value)) return value.length === 0 ? "∅" : value.join(", ");
  if (value === null || value === undefined || value === "") return "∅";
  return String(value);
}

function ChangeSummary({ version }: { version: TopologyVersion }) {
  const { t } = useLocale();
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
              {field.material ? <span className="reason"> {t("material")}</span> : <span className="muted"> {t("display-only")}</span>}
            </li>
          ))}
        </ul>
      )}
      <p className="topology-reason"><span className="muted">{t("Reason:")}</span> {change.reason}</p>
      <p className="topology-impact"><span className="muted">{t("Impact:")}</span> {change.impact}</p>
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
  const { t } = useLocale();
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
        <label>{t("Environment id")}
          <input name="id" value={values.id} onChange={update("id")} disabled={mode === "edit"} placeholder="uat" required={mode === "add"} />
        </label>
        <label>{t("Display name")}
          <input name="display_name" value={values.display_name} onChange={update("display_name")} placeholder="UAT" />
        </label>
        <label>{t("Standard type")}
          <select name="type" value={values.type} onChange={update("type")} disabled={isProduction}>
            {standardTypes.map((type) => <option key={type} value={type}>{type}</option>)}
          </select>
        </label>
        <label>{t("Sequence")}
          <input name="sequence" type="number" min={1} value={values.sequence} onChange={update("sequence")} placeholder={t("append")} />
        </label>
        <label>{t("Target ref")}
          <input name="target_ref" value={values.target_ref} onChange={update("target_ref")} placeholder="cloud-run/service-uat" />
        </label>
        <label>{t("Owner")}
          <input name="owner" value={values.owner} onChange={update("owner")} placeholder="qa-lead" />
        </label>
        <label>{t("Required evidence (comma separated)")}
          <input name="required_evidence" value={values.required_evidence} onChange={update("required_evidence")} placeholder="test, build, smoke" />
        </label>
        <label>{t("Approver policy")}
          <input name="approver_policy" value={values.approver_policy} onChange={update("approver_policy")} placeholder={t("distinct human approver")} />
        </label>
      </div>
      <label className="topology-reason-field">{t("Reason for this version")}
        <input name="reason" value={values.reason} onChange={update("reason")} placeholder={t("why the topology changes")} required />
      </label>
      {isProduction && <p className="muted">{t("Production: display name and owner may change; standard type is protected in P0.")}</p>}
      <div className="topology-form-actions">
        <button type="submit" className="button-primary" disabled={busy}>{mode === "add" ? t("Create version with new environment") : t("Save as new version")}</button>
        <button type="button" className="button-secondary" onClick={onCancel} disabled={busy}>{t("Cancel")}</button>
      </div>
    </form>
  );
}

export function EnvironmentTopology({ controller }: EnvironmentTopologyProps) {
  const { t } = useLocale();
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
          <p className="eyebrow">{t("ENVIRONMENT TOPOLOGY")}</p>
          <h2 id="topology-heading">{t("Promotion path")}</h2>
        </div>
        <div className="topology-heading-meta">
          {current && (
            <>
              <StatusBadge status="VERSIONED" label={t("Topology v{version}", { version: current.version })} />
              <span className="muted" title={current.config_hash}>{t("config")} {current.config_hash.slice(7, 19)}</span>
            </>
          )}
          <span className="muted">{t("Every change creates an immutable version; nothing is hard-deleted")}</span>
        </div>
      </div>

      {status === "LOADING" && <p className="muted" data-testid="topology-loading"><span className="spinner" aria-hidden="true" />{t("Loading topology…")}</p>}
      {status === "ERROR" && error && (
        <div className="workspace-state error-state" data-testid="topology-error">
          <strong>{error.code}</strong> — {error.message}
          <div><button type="button" className="button-secondary" onClick={controller.reload}>{t("Reload topology")}</button></div>
        </div>
      )}

      {state && current && (
        <>
          {status === "READY" && error && (
            <div className="topology-inline-error" data-testid="topology-error" role="alert">
              <strong>{error.code}</strong> — {error.message}
              {error.code === "TOPOLOGY_VERSION_CONFLICT" && (
                <button type="button" className="button-secondary" onClick={controller.reload}>{t("Reload topology")}</button>
              )}
            </div>
          )}

          <ChangeSummary version={current} />

          <div className="topology-quick-reason">
            <label>{t("Reason for move / retire / restore")}
              <input
                data-testid="topology-quick-reason"
                value={quickReason}
                onChange={(event) => { setQuickReason(event.target.value); if (event.target.value.trim()) setQuickReasonMissing(false); }}
                placeholder={t("required before a quick action")}
              />
            </label>
            {quickReasonMissing && <span className="reason" data-testid="topology-reason-missing">{t("A reason is required for every topology change.")}</span>}
          </div>

          <table className="topology-table">
            <thead>
              <tr>
                <th>#</th><th>{t("Environment")}</th><th>{t("Type")}</th><th>{t("Target")}</th><th>{t("Owner")}</th><th>{t("Required evidence")}</th><th>{t("Status")}</th><th>{t("Actions")}</th>
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
                    <td>{environment.owner || <span className="muted">{t("UNDECLARED")}</span>}</td>
                    <td>{environment.required_evidence.length === 0 ? <span className="muted">{t("none")}</span> : environment.required_evidence.join(", ")}</td>
                    <td>
                      <div className="topology-status">
                        <StatusBadge status={environment.status} />
                        {environment.protection && <StatusBadge status="BLOCKED" label={t("READ_ONLY · BLOCKED IN P0")} />}
                        {environment.action && <span className="muted">{environment.action}</span>}
                      </div>
                    </td>
                    <td>
                      <div className="topology-actions">
                        <button type="button" className="button-icon" aria-label={t("Move {id} up", { id: environment.id })} disabled={busy || index === 0} onClick={() => void move(environment.id, -1)}>↑</button>
                        <button type="button" className="button-icon" aria-label={t("Move {id} down", { id: environment.id })} disabled={busy || index === environments.length - 1} onClick={() => void move(environment.id, 1)}>↓</button>
                        <button type="button" className="button-secondary" disabled={busy || retired} onClick={() => { setAdding(false); setEditingID(environment.id); }}>{t("Edit")}</button>
                        {retired ? (
                          <button type="button" className="button-secondary" disabled={busy} onClick={() => void restore(environment.id)}>{t("Restore")}</button>
                        ) : (
                          <button
                            type="button"
                            className="button-secondary"
                            disabled={busy || production}
                            title={production ? t("Production cannot be retired in P0") : undefined}
                            onClick={() => void retire(environment.id)}
                          >{t("Retire")}</button>
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
              <button type="button" className="button-primary" disabled={busy} onClick={() => { setEditingID(null); setAdding(true); }}>{t("Add environment")}</button>
            </div>
          )}

          <div className="topology-invalidations" data-testid="topology-invalidations">
            <h3>{t("Decisions marked STALE by topology changes")}</h3>
            {state.invalidations.length === 0 ? (
              <p className="muted">{t("No decision has been invalidated by a topology change.")}</p>
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
              {showHistory ? t("Hide version history ({count})", { count: state.versions.length }) : t("Show version history ({count})", { count: state.versions.length })}
            </button>
            {showHistory && (
              <ol className="topology-history-list" data-testid="topology-history">
                {state.versions.slice().reverse().map((version) => (
                  <li key={version.version}>
                    <span className="topology-version">v{version.version}</span> {version.change.operation}
                    {version.change.environment_id ? ` ${version.change.environment_id}` : ""} · {version.change.material ? t("material") : t("informational")} · {version.actor} · <span className="muted">{version.change.reason}</span>
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
