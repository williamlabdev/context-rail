import { useState, type FormEvent } from "react";
import { currentVersion, type ChangeView, type MissingInput, type Option } from "../api/changes";
import type { ChangesController } from "../state/useChanges";
import type { TopologyEnvironment } from "../api/topology";
import { CandidateReview } from "./CandidateReview";
import { ChangeBrief, LineageStrip } from "./ChangeBrief";
import { StatusBadge } from "./StatusBadge";
import { useLocale } from "../i18n";

interface ChangesPanelProps {
  controller: ChangesController;
  environments: TopologyEnvironment[];
}

const lines = (value: string): string[] => value.split("\n").map((line) => line.trim()).filter(Boolean);
const criteriaFrom = (value: string) => lines(value).map((text) => ({ id: "", text }));
const constraintsFrom = (value: string): Record<string, string> => {
  const out: Record<string, string> = {};
  for (const line of lines(value)) {
    const index = line.indexOf("=");
    if (index > 0) out[line.slice(0, index).trim()] = line.slice(index + 1).trim();
  }
  return out;
};

// ---------------------------------------------------------------- create form

interface CreateFormProps {
  environments: TopologyEnvironment[];
  busy: boolean;
  onCancel: () => void;
  onSubmit: ChangesController["create"];
}

function CreateChangeForm({ environments, busy, onCancel, onSubmit }: CreateFormProps) {
  const { t } = useLocale();
  const [values, setValues] = useState({
    title: "", objective: "", owner_summary: "", scope_included: "", scope_excluded: "", acceptance_criteria: "", allowed_paths: "",
    forbidden_actions: "", target_environment_id: "", data_classification: "", expected_monthly_volume: "", extra_constraints: "", reason: "",
  });
  const update = (field: keyof typeof values) => (event: { target: { value: string } }) => setValues((previous) => ({ ...previous, [field]: event.target.value }));
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const constraints = constraintsFrom(values.extra_constraints);
    if (values.data_classification.trim()) constraints.data_classification = values.data_classification.trim();
    if (values.expected_monthly_volume.trim()) constraints.expected_monthly_volume = values.expected_monthly_volume.trim();
    const ok = await onSubmit({
      reason: values.reason.trim(),
      request: {
        title: values.title.trim(), objective: values.objective.trim(), owner_summary: values.owner_summary.trim(),
        scope_included: lines(values.scope_included), scope_excluded: lines(values.scope_excluded),
        acceptance_criteria: criteriaFrom(values.acceptance_criteria), allowed_paths: lines(values.allowed_paths),
        forbidden_actions: lines(values.forbidden_actions), target_environment_id: values.target_environment_id,
        business_constraints: constraints,
      },
    });
    if (ok) onCancel();
  };
  const selectable = environments.filter((environment) => environment.status !== "RETIRED");
  return (
    <form className="topology-form" data-testid="change-form-create" onSubmit={(event) => void submit(event)}>
      <p className="eyebrow">{t("NEW CHANGE · leave fields empty to see how the evaluation blocks on missing inputs")}</p>
      <div className="topology-form-grid">
        <label>{t("Title")}<input name="title" value={values.title} onChange={update("title")} required placeholder={t("Manual order review")} /></label>
        <label>{t("Target environment")}
          <select name="target_environment_id" value={values.target_environment_id} onChange={update("target_environment_id")}>
            <option value="">{t("— not declared —")}</option>
            {selectable.map((environment) => (
              <option key={environment.id} value={environment.id}>{environment.display_name} ({environment.type}{environment.protection ? `, ${t("blocked in P0")}` : ""})</option>
            ))}
          </select>
        </label>
        <label className="span-2">{t("Owner summary (plain language, for the business owner)")}<textarea name="owner_summary" rows={2} value={values.owner_summary} onChange={update("owner_summary")} placeholder={t("What changes for the people who use it, in one or two sentences")} /></label>
        <label className="span-2">{t("Objective")}<textarea name="objective" rows={2} value={values.objective} onChange={update("objective")} placeholder={t("What must be true for the requester afterwards")} /></label>
        <label>{t("Scope included (one per line)")}<textarea name="scope_included" rows={3} value={values.scope_included} onChange={update("scope_included")} /></label>
        <label>{t("Scope excluded (one per line)")}<textarea name="scope_excluded" rows={3} value={values.scope_excluded} onChange={update("scope_excluded")} /></label>
        <label>{t("Acceptance criteria (one per line)")}<textarea name="acceptance_criteria" rows={3} value={values.acceptance_criteria} onChange={update("acceptance_criteria")} /></label>
        <label>{t("Allowed paths (one per line)")}<textarea name="allowed_paths" rows={3} value={values.allowed_paths} onChange={update("allowed_paths")} placeholder="main.go&#10;web/index.html" /></label>
        <label>{t("Forbidden actions (one per line)")}<textarea name="forbidden_actions" rows={2} value={values.forbidden_actions} onChange={update("forbidden_actions")} /></label>
        <label>{t("Other constraints (key=value per line)")}<textarea name="extra_constraints" rows={2} value={values.extra_constraints} onChange={update("extra_constraints")} placeholder="retention_days=365" /></label>
        <label>{t("Data classification")}<input name="data_classification" value={values.data_classification} onChange={update("data_classification")} placeholder="internal" /></label>
        <label>{t("Expected monthly volume")}<input name="expected_monthly_volume" value={values.expected_monthly_volume} onChange={update("expected_monthly_volume")} placeholder={t("500 reviews")} /></label>
      </div>
      <label className="topology-reason-field">{t("Reason for opening this Change")}<input name="reason" value={values.reason} onChange={update("reason")} required /></label>
      <div className="topology-form-actions">
        <button type="submit" className="button-primary" disabled={busy}>{t("Open Change and evaluate")}</button>
        <button type="button" className="button-secondary" onClick={onCancel} disabled={busy}>{t("Cancel")}</button>
      </div>
    </form>
  );
}

// ---------------------------------------------------------------- inputs form

interface InputsFormProps {
  view: ChangeView;
  environments: TopologyEnvironment[];
  busy: boolean;
  onSubmit: ChangesController["inputs"];
}

function SupplyInputsForm({ view, environments, busy, onSubmit }: InputsFormProps) {
  const { t } = useLocale();
  const version = currentVersion(view.change);
  const missing = new Set(version.evaluation.missing_inputs.map((input) => input.field));
  const [values, setValues] = useState({
    objective: version.request.objective, owner_summary: version.request.owner_summary ?? "", acceptance_criteria: "", allowed_paths: version.request.allowed_paths.join("\n"),
    target_environment_id: version.request.target_environment_id, data_classification: version.request.business_constraints.data_classification ?? "",
    expected_monthly_volume: version.request.business_constraints.expected_monthly_volume ?? "", extra_constraints: "", reason: "",
  });
  const update = (field: keyof typeof values) => (event: { target: { value: string } }) => setValues((previous) => ({ ...previous, [field]: event.target.value }));
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const constraints = constraintsFrom(values.extra_constraints);
    if (values.data_classification.trim()) constraints.data_classification = values.data_classification.trim();
    if (values.expected_monthly_volume.trim()) constraints.expected_monthly_volume = values.expected_monthly_volume.trim();
    await onSubmit(view.change.change_id, {
      reason: values.reason.trim(),
      ...(values.objective.trim() !== version.request.objective ? { objective: values.objective.trim() } : {}),
      ...(values.owner_summary.trim() !== (version.request.owner_summary ?? "") ? { owner_summary: values.owner_summary.trim() } : {}),
      ...(values.acceptance_criteria.trim() ? { acceptance_criteria: criteriaFrom(values.acceptance_criteria) } : {}),
      ...(values.allowed_paths.trim() !== version.request.allowed_paths.join("\n") ? { allowed_paths: lines(values.allowed_paths) } : {}),
      ...(values.target_environment_id !== version.request.target_environment_id ? { target_environment_id: values.target_environment_id } : {}),
      ...(Object.keys(constraints).length > 0 ? { business_constraints: constraints } : {}),
    });
  };
  const selectable = environments.filter((environment) => environment.status !== "RETIRED");
  return (
    <form className="topology-form" data-testid="change-inputs-form" onSubmit={(event) => void submit(event)}>
      <p className="eyebrow">{t("SUPPLY INPUTS · creates change v{version} and re-evaluates; earlier versions stay readable", { version: version.version + 1 })}</p>
      <div className="topology-form-grid">
        {missing.has("owner_summary") && <label className="span-2">{t("Owner summary (plain language, for the business owner)")}<textarea name="owner_summary" rows={2} value={values.owner_summary} onChange={update("owner_summary")} /></label>}
        {missing.has("objective") && <label className="span-2">{t("Objective")}<textarea name="objective" rows={2} value={values.objective} onChange={update("objective")} /></label>}
        {missing.has("acceptance_criteria") && <label>{t("Acceptance criteria (one per line)")}<textarea name="acceptance_criteria" rows={3} value={values.acceptance_criteria} onChange={update("acceptance_criteria")} /></label>}
        {missing.has("allowed_paths") && <label>{t("Allowed paths (one per line)")}<textarea name="allowed_paths" rows={3} value={values.allowed_paths} onChange={update("allowed_paths")} /></label>}
        {missing.has("target_environment_id") && (
          <label>{t("Target environment")}
            <select name="target_environment_id" value={values.target_environment_id} onChange={update("target_environment_id")}>
              <option value="">{t("— not declared —")}</option>
              {selectable.map((environment) => <option key={environment.id} value={environment.id}>{environment.display_name} ({environment.type})</option>)}
            </select>
          </label>
        )}
        {missing.has("business_constraints.data_classification") && <label>{t("Data classification")}<input name="data_classification" value={values.data_classification} onChange={update("data_classification")} /></label>}
        {missing.has("business_constraints.expected_monthly_volume") && <label>{t("Expected monthly volume")}<input name="expected_monthly_volume" value={values.expected_monthly_volume} onChange={update("expected_monthly_volume")} /></label>}
        <label>{t("Other constraints (key=value per line)")}<textarea name="extra_constraints" rows={2} value={values.extra_constraints} onChange={update("extra_constraints")} /></label>
      </div>
      <label className="topology-reason-field">{t("Reason")}<input name="reason" value={values.reason} onChange={update("reason")} required placeholder={t("who supplied what")} /></label>
      <div className="topology-form-actions">
        <button type="submit" className="button-primary" disabled={busy}>{t("Re-evaluate with these inputs")}</button>
      </div>
    </form>
  );
}

// ---------------------------------------------------------------- decision form

interface DecisionFormProps {
  view: ChangeView;
  busy: boolean;
  onSubmit: ChangesController["decide"];
}

function DecisionForm({ view, busy, onSubmit }: DecisionFormProps) {
  const { t } = useLocale();
  const version = currentVersion(view.change);
  const recommended = version.options.find((option) => option.recommended)?.id ?? version.options[0]?.id ?? "";
  const [values, setValues] = useState({ actor: "", role: "solution_architect", selected_option: recommended, rationale: "", risk_level: "low" });
  const update = (field: keyof typeof values) => (event: { target: { value: string } }) => setValues((previous) => ({ ...previous, [field]: event.target.value }));
  const decide = async (decision: "ACCEPT" | "REJECT") => {
    await onSubmit(view.change.change_id, {
      actor: values.actor.trim(), role: values.role.trim(), decision, selected_option: values.selected_option, rationale: values.rationale.trim(), risk_level: values.risk_level,
    });
  };
  return (
    <form className="topology-form" data-testid="change-decision-form" onSubmit={(event) => { event.preventDefault(); void decide("ACCEPT"); }}>
      <p className="eyebrow">{t("HUMAN DECISION · version {version} is DECISION_READY; nothing below is chosen by the advisor", { version: version.version })}</p>
      <fieldset className="option-list" data-testid="change-options">
        <legend>{t("Select one candidate")}</legend>
        {version.options.map((option: Option) => (
          <label key={option.id} className={`option-card${values.selected_option === option.id ? " option-selected" : ""}`}>
            <input type="radio" name="selected_option" value={option.id} checked={values.selected_option === option.id} onChange={update("selected_option")} />
            <span className="option-title">{option.title}{option.recommended && <StatusBadge status="RECOMMENDED" />} <span className="muted">{option.advisor_source}</span></span>
            <span className="option-summary">{option.summary}</span>
            <span className="muted">{t("cost:")} {option.cost_drivers.join("; ")} · {t("risks:")} {option.risks.join("; ")}</span>
          </label>
        ))}
      </fieldset>
      <div className="topology-form-grid">
        <label>{t("Deciding actor")}<input name="actor" value={values.actor} onChange={update("actor")} required placeholder="founder-001" /></label>
        <label>{t("Role")}<input name="role" value={values.role} onChange={update("role")} /></label>
        <label>{t("Risk level")}
          <select name="risk_level" value={values.risk_level} onChange={update("risk_level")}>
            <option value="low">{t("low")}</option><option value="medium">{t("medium")}</option><option value="high">{t("high")}</option>
          </select>
        </label>
        <label className="span-2">{t("Rationale")}<textarea name="rationale" rows={2} value={values.rationale} onChange={update("rationale")} required /></label>
      </div>
      <div className="topology-form-actions">
        <button type="submit" className="button-primary" disabled={busy}>{t("Accept decision")}</button>
        <button type="button" className="button-secondary" disabled={busy} onClick={() => void decide("REJECT")}>{t("Reject change")}</button>
      </div>
    </form>
  );
}

// ---------------------------------------------------------------- detail

function MissingInputs({ inputs }: { inputs: MissingInput[] }) {
  const { t } = useLocale();
  return (
    <div className="workspace-state error-state change-missing" data-testid="change-missing-inputs">
      <strong>NEEDS_INPUT — {inputs.length === 1 ? t("1 input must be supplied by a person; the system will not invent it.") : t("{count} inputs must be supplied by a person; the system will not invent them.", { count: inputs.length })}</strong>
      <table className="topology-table">
        <thead><tr><th>{t("Field")}</th><th>{t("Owner")}</th><th>{t("Why")}</th></tr></thead>
        <tbody>
          {inputs.map((input) => (
            <tr key={input.field}><td><code>{input.field}</code></td><td>{input.owner_role}</td><td>{input.reason}</td></tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ChangeDetail({ view, controller, environments }: { view: ChangeView; controller: ChangesController; environments: TopologyEnvironment[] }) {
  const { t } = useLocale();
  const { change, decision, brief, agent_context_pack: pack, work_order: order, staleness } = view;
  const version = currentVersion(change);
  const evaluation = version.evaluation;
  const [showPack, setShowPack] = useState(false);
  const [woReason, setWoReason] = useState("");
  const [woIssuer, setWoIssuer] = useState("");
  const decidedThisVersion = decision !== null && decision.change_version === change.current_version && decision.status !== "REJECTED";
  const canDecide = evaluation.status === "DECISION_READY" && !decidedThisVersion && change.status !== "REJECTED";
  const canIssue = decision?.status === "ACCEPTED_FOR_DEVELOPMENT" && !staleness.stale && !decidedIsIssued(view);

  return (
    <div className="change-detail" data-testid="change-detail">
      <div className="topology-change-heading">
        <code>{change.change_id}</code>
        <strong>{change.title}</strong>
        <StatusBadge status={change.status} />
        <span className="muted">{t("change v{version}", { version: change.current_version })} · {t("evaluated")} {evaluation.evaluated_at}</span>
      </div>
      {staleness.stale && (
        <div className="topology-inline-error" data-testid="change-stale" role="alert">
          <strong>STALE</strong> — {staleness.reason}
        </div>
      )}

      <div className="change-evaluation">
        <p className="muted">
          {t("Target")} {evaluation.target_environment?.id ?? "∅"} · {t("transition")} {evaluation.allowed_transition || "∅"} · {t("topology v{version}", { version: evaluation.topology_version })} · {t("snapshot")} <code title={evaluation.source_snapshot_hash}>{evaluation.source_snapshot_hash.slice(7, 19)}</code> · {t("context")} {evaluation.project_context_status}
        </p>
        {evaluation.observations.length > 0 && <ul className="reason-list">{evaluation.observations.map((line) => <li key={line}>{line}</li>)}</ul>}
        {version.unknowns.length > 0 && (
          <div data-testid="change-unknowns"><span className="muted">{t("Unknowns:")}</span><ul className="reason-list">{version.unknowns.map((line) => <li key={line}>{line}</li>)}</ul></div>
        )}
      </div>

      {evaluation.status === "NEEDS_INPUT" && change.status !== "REJECTED" && (
        <>
          <MissingInputs inputs={evaluation.missing_inputs} />
          <SupplyInputsForm key={`inputs-${change.current_version}`} view={view} environments={environments} busy={controller.busy} onSubmit={controller.inputs} />
        </>
      )}

      {canDecide && <DecisionForm key={`decide-${change.current_version}`} view={view} busy={controller.busy} onSubmit={controller.decide} />}
      {staleness.stale && evaluation.status === "DECISION_READY" && !canDecide && (
        <p className="reason">{t("The current decision is stale; supply inputs (creating a new version) or re-decide after the topology settles.")}</p>
      )}

      {decision && (
        <div className="decision-card" data-testid="change-decision">
          <div className="topology-change-heading">
            <code>{decision.decision_id} v{decision.version}</code>
            <StatusBadge status={staleness.stale && decision.status !== "REJECTED" ? "STALE" : decision.status} />
            <span className="muted">{decision.human_decision.actor} ({decision.human_decision.role}) {decision.human_decision.decision} {t("at")} {decision.human_decision.at}</span>
          </div>
          <p className="topology-reason"><span className="muted">{t("Selected:")}</span> {decision.alternatives.find((option) => option.id === decision.selected_option)?.title ?? decision.selected_option} <code className="muted">{decision.selected_option}</code> · <span className="muted">{t("rationale:")}</span> {decision.rationale}</p>
        </div>
      )}

      {brief && pack && (
        <div className="dual-view">
          <ChangeBrief brief={brief} />
          <section className="dual-pane" data-testid="change-pack">
            <p className="eyebrow">{t("AGENT CONTEXT PACK · for the coding agent")}</p>
            <LineageStrip lineage={pack.lineage} />
            <p className="topology-reason"><StatusBadge status={pack.status} /> {pack.objective}</p>
            <div className="brief-section"><h4>{t("Allowed paths")}</h4><ul>{pack.allowed_paths.map((path) => <li key={path}><code>{path}</code></li>)}</ul></div>
            <div className="brief-section"><h4>{t("Forbidden actions")}</h4><ul>{pack.forbidden_actions.map((action) => <li key={action}><code>{action}</code></li>)}</ul></div>
            <div className="brief-section"><h4>{t("Acceptance tests")}</h4><ul>{pack.acceptance_tests.map((test) => <li key={test.id}><code>{test.id}</code> {test.expected}</li>)}</ul></div>
            <div className="brief-section"><h4>{t("Required checks")}</h4><ul>{pack.required_checks.map((check) => <li key={check}>{check}</li>)}</ul></div>
            <button type="button" className="button-link" onClick={() => setShowPack((value) => !value)}>{showPack ? t("Hide pack JSON") : t("Show pack JSON")}</button>
            {showPack && <pre className="json-view" data-testid="change-pack-json">{JSON.stringify(pack, null, 2)}</pre>}
          </section>
        </div>
      )}

      {canIssue && (
        <form className="topology-form" data-testid="change-work-order-form" onSubmit={(event) => { event.preventDefault(); void controller.workOrder(change.change_id, { reason: woReason.trim(), issuer: woIssuer.trim() || undefined }); }}>
          <p className="eyebrow">{t("AGENT WORK ORDER · compiled from {decision}; issuing does not run any agent", { decision: decision?.decision_id ?? "∅" })}</p>
          <div className="topology-form-grid">
            <label>{t("Issuer")}<input name="issuer" value={woIssuer} onChange={(event) => setWoIssuer(event.target.value)} placeholder="founder-001" /></label>
            <label>{t("Reason")}<input name="reason" value={woReason} onChange={(event) => setWoReason(event.target.value)} required /></label>
          </div>
          <div className="topology-form-actions"><button type="submit" className="button-primary" disabled={controller.busy}>{t("Compile Work Order")}</button></div>
        </form>
      )}

      {order && (
        <div className="decision-card" data-testid="change-work-order">
          <div className="topology-change-heading">
            <code>{order.work_order_id}</code>
            <StatusBadge status={order.status} />
            <span className="muted">{t("issued by")} {order.issuer} {t("at")} {order.issued_at} · {t("expires")} {order.expires_at}</span>
          </div>
          <LineageStrip lineage={order.lineage} />
          <p className="topology-reason"><span className="muted">{t("hash")}</span> <code>{order.work_order_hash}</code></p>
          <p className="topology-reason"><span className="muted">{t("repo")}</span> <code>{order.repository}</code> · {order.base_branch} → {order.target_branch} · {t("target")} {order.target_environment_id} ({order.allowed_transition})</p>
          <p className="topology-reason"><span className="muted">{t("checks")}</span> {order.required_checks.join(", ")} · <span className="muted">{t("acceptance")}</span> {order.acceptance_ids.join(", ")}</p>
        </div>
      )}

      {order && <CandidateReview view={view} controller={controller} />}

      {change.versions.length > 1 && (
        <details className="topology-history">
          <summary>{t("Version history ({count})", { count: change.versions.length })}</summary>
          <ol className="topology-history-list" data-testid="change-history">
            {change.versions.slice().reverse().map((entry) => (
              <li key={entry.version}><span className="topology-version">v{entry.version}</span> {entry.evaluation.status} · {entry.actor} · <span className="muted">{entry.reason}</span></li>
            ))}
          </ol>
        </details>
      )}
    </div>
  );
}

function decidedIsIssued(view: ChangeView): boolean {
  return Boolean(view.work_order && view.decision && view.work_order.lineage.decision_id === view.decision.decision_id && view.work_order.lineage.decision_version === view.decision.version);
}

// ---------------------------------------------------------------- panel

export function ChangesPanel({ controller, environments }: ChangesPanelProps) {
  const { t } = useLocale();
  const { status, changes, selected, error, busy } = controller;
  const [creating, setCreating] = useState(false);
  return (
    <section className="panel topology-panel" aria-labelledby="changes-heading" data-testid="changes-panel">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{t("CHANGE DECISION PACK")}</p>
          <h2 id="changes-heading">{t("Changes")}</h2>
        </div>
        <span className="muted">{t("Request → evaluation → human decision → Brief + Agent Context Pack → Work Order")}</span>
      </div>

      {status === "LOADING" && <p className="muted" data-testid="changes-loading"><span className="spinner" aria-hidden="true" />{t("Loading changes…")}</p>}
      {error && (
        <div className="topology-inline-error" data-testid="changes-error" role="alert">
          <strong>{error.code}</strong> — {error.message}
          {status === "ERROR" && <button type="button" className="button-secondary" onClick={controller.reload}>{t("Reload changes")}</button>}
        </div>
      )}

      {status !== "LOADING" && (
        <div className="changes-layout">
          <div className="changes-list">
            {changes.length === 0 && <p className="muted" data-testid="changes-empty">{t("No Change opened yet for this Project.")}</p>}
            {changes.map((view) => (
              <button
                type="button"
                key={view.change.change_id}
                className={`project-card${controller.selectedID === view.change.change_id ? " project-card-selected" : ""}`}
                data-testid={`change-card-${view.change.change_id}`}
                onClick={() => controller.select(view.change.change_id)}
              >
                <span className="project-card-title">{view.change.title}</span>
                <span className="project-card-id">{view.change.change_id} · v{view.change.current_version}</span>
                <span className="project-card-meta"><StatusBadge status={view.change.status} />{view.decision && <StatusBadge status={view.decision.decision_id} />}{view.work_order && <StatusBadge status={view.work_order.work_order_id} />}</span>
              </button>
            ))}
            {creating ? null : (
              <button type="button" className="button-primary" disabled={busy} onClick={() => { controller.select(null); setCreating(true); }}>{t("New Change")}</button>
            )}
          </div>
          <div className="changes-detail">
            {creating && <CreateChangeForm environments={environments} busy={busy} onCancel={() => setCreating(false)} onSubmit={controller.create} />}
            {!creating && selected && <ChangeDetail key={selected.change.change_id} view={selected} controller={controller} environments={environments} />}
            {!creating && !selected && changes.length > 0 && <p className="muted">{t("Select a Change to see its evaluation, decision and handoff artifacts.")}</p>}
          </div>
        </div>
      )}
    </section>
  );
}
