import { useState, type FormEvent } from "react";
import type { AgentWorkOrder, Candidate, ChangeView, CheckResult, ReviewEvidence } from "../api/changes";
import type { ChangesController } from "../state/useChanges";
import { StatusBadge } from "./StatusBadge";
import { useLocale } from "../i18n";

interface CandidateReviewProps {
  view: ChangeView;
  controller: ChangesController;
}

const lines = (value: string): string[] => value.split("\n").map((line) => line.trim()).filter(Boolean);

function parseChecks(value: string): CheckResult[] {
  return lines(value).map((line) => {
    const [name, status = "PASS", ref = ""] = line.split("=").map((part) => part.trim());
    return { name, status: status.toUpperCase(), evidence_ref: ref };
  });
}

// ---------------------------------------------------------------- submit form

function SubmitCandidateForm({ order, changeID, busy, onSubmit }: { order: AgentWorkOrder; changeID: string; busy: boolean; onSubmit: ChangesController["submitCandidate"] }) {
  const { t } = useLocale();
  const [values, setValues] = useState({
    run_id: "", started_by: "", agent_name: "claude-code", agent_model: "", branch: order.target_branch, base_branch: order.base_branch,
    head_commit: "", changed_paths: "", checks: "test=PASS\nbuild=PASS", reviewer: "", review_kind: "human", review_verdict: "APPROVED",
    ai_review: false, controls_declared: false, controls_ref: "", controls_reason: "", read_back: false, repository: order.repository, reason: "",
  });
  const update = (field: keyof typeof values) => (event: { target: { value: string } }) => setValues((previous) => ({ ...previous, [field]: event.target.value }));
  const toggle = (field: "ai_review" | "controls_declared" | "read_back") => () => setValues((previous) => ({ ...previous, [field]: !previous[field] }));
  const submit = async (event: FormEvent) => {
    event.preventDefault();
    const reviews: ReviewEvidence[] = [];
    if (values.reviewer.trim()) reviews.push({ reviewer: values.reviewer.trim(), kind: values.review_kind, verdict: values.review_verdict });
    if (values.ai_review) reviews.push({ reviewer: "ai-technical-review", kind: "ai", verdict: "APPROVED" });
    await onSubmit(changeID, {
      reason: values.reason.trim(),
      run: {
        run_id: values.run_id.trim(), work_order_id: order.work_order_id, work_order_hash: order.work_order_hash,
        agent: { name: values.agent_name.trim(), provider: "", model: values.agent_model.trim(), version: "" },
        started_by: values.started_by.trim(), adapter: values.read_back ? "github-read-back" : "manual-declaration",
      },
      observation: {
        source: "declared", repository: values.repository.trim(), branch: values.branch.trim(), base_branch: values.base_branch.trim(), head_commit: values.head_commit.trim(),
        changed_paths: lines(values.changed_paths), checks: parseChecks(values.checks), reviews,
        compensating_controls: values.controls_declared ? { declared: true, evidence_ref: values.controls_ref.trim(), reason: values.controls_reason.trim() } : null,
      },
      ...(values.read_back ? { read_back: { source: "github", repository: values.repository.trim(), base: values.base_branch.trim(), head: values.branch.trim() } } : {}),
    });
  };
  return (
    <form className="topology-form" data-testid="candidate-form" onSubmit={(event) => void submit(event)}>
      <p className="eyebrow">{t("SUBMIT CANDIDATE · run declaration + what was observed; the gate checks the observation against {order}, not the declaration", { order: order.work_order_id })}</p>
      <div className="topology-form-grid">
        <label>{t("Run id")}<input name="run_id" value={values.run_id} onChange={update("run_id")} placeholder="ARR-001" /></label>
        <label>{t("Started by (human who ran the agent)")}<input name="started_by" value={values.started_by} onChange={update("started_by")} required placeholder="dev-1" /></label>
        <label>{t("Agent")}<input name="agent_name" value={values.agent_name} onChange={update("agent_name")} /></label>
        <label>{t("Model")}<input name="agent_model" value={values.agent_model} onChange={update("agent_model")} placeholder="claude-fable-5-1" /></label>
        <label>{t("Branch")}<input name="branch" value={values.branch} onChange={update("branch")} /></label>
        <label>{t("Base branch")}<input name="base_branch" value={values.base_branch} onChange={update("base_branch")} /></label>
        <label>{t("Head commit")}<input name="head_commit" value={values.head_commit} onChange={update("head_commit")} placeholder="sha" /></label>
        <label>{t("Repository")}<input name="repository" value={values.repository} onChange={update("repository")} /></label>
        <label className="span-2">{t("Changed paths (one per line, as observed in the diff)")}<textarea name="changed_paths" rows={3} value={values.changed_paths} onChange={update("changed_paths")} /></label>
        <label className="span-2">{t("Checks (name=PASS|FAIL[=evidence ref] per line)")}<textarea name="checks" rows={3} value={values.checks} onChange={update("checks")} /></label>
        <label>{t("Reviewer")}<input name="reviewer" value={values.reviewer} onChange={update("reviewer")} placeholder={t("someone other than the run starter")} /></label>
        <label>{t("Review verdict")}
          <select name="review_verdict" value={values.review_verdict} onChange={update("review_verdict")}>
            <option value="APPROVED">APPROVED</option><option value="CHANGES_REQUESTED">CHANGES_REQUESTED</option>
          </select>
        </label>
        <label className="checkbox-line"><input type="checkbox" name="ai_review" checked={values.ai_review} onChange={toggle("ai_review")} /> {t("AI technical review APPROVED (evidence, not approval)")}</label>
        <label className="checkbox-line"><input type="checkbox" name="controls_declared" checked={values.controls_declared} onChange={toggle("controls_declared")} /> {t("Single-operator controls documented")}</label>
        {values.controls_declared && (
          <>
            <label>{t("Controls evidence ref")}<input name="controls_ref" value={values.controls_ref} onChange={update("controls_ref")} placeholder="evidence/single-operator-controls.md" /></label>
            <label>{t("Controls reason")}<input name="controls_reason" value={values.controls_reason} onChange={update("controls_reason")} /></label>
          </>
        )}
        <label className="checkbox-line span-2"><input type="checkbox" name="read_back" checked={values.read_back} onChange={toggle("read_back")} /> {t("Read back branch, diff, checks and reviews from GitHub (overrides the declaration; needs a configured adapter)")}</label>
      </div>
      <label className="topology-reason-field">{t("Reason")}<input name="reason" value={values.reason} onChange={update("reason")} required placeholder={t("agent run finished; submitting for gate")} /></label>
      <div className="topology-form-actions"><button type="submit" className="button-primary" disabled={busy}>{t("Submit candidate to gate")}</button></div>
    </form>
  );
}

// ---------------------------------------------------------------- one candidate

function CandidateCard({ candidate, changeID, controller, decisionStale }: { candidate: Candidate; changeID: string; controller: ChangesController; decisionStale: boolean }) {
  const { t } = useLocale();
  const [actor, setActor] = useState("");
  const [rationale, setRationale] = useState("");
  const [open, setOpen] = useState(candidate.human_decision === null);
  const decidable = candidate.human_decision === null && candidate.status !== "STALE" && !decisionStale;
  const decide = async (decision: "ACCEPT" | "REJECT") => {
    await controller.decideCandidate(changeID, candidate.candidate_id, { actor: actor.trim(), role: "solution_architect", decision, rationale: rationale.trim() });
  };
  return (
    <div className="decision-card candidate-card" data-testid={`candidate-${candidate.candidate_id}`}>
      <div className="topology-change-heading">
        <code>{candidate.candidate_id}</code>
        <StatusBadge status={candidate.verdict} />
        <StatusBadge status={candidate.status} />
        <span className="muted">{candidate.run.run_id} {t("by")} {candidate.run.started_by} · {candidate.run.agent.name}{candidate.run.agent.model ? ` (${candidate.run.agent.model})` : ""} · {t("observed via")} {candidate.observation.source} · {candidate.created_at}</span>
        <button type="button" className="button-link" onClick={() => setOpen((value) => !value)}>{open ? t("Hide gates") : t("Show gates")}</button>
      </div>
      <p className="topology-reason"><span className="muted">{candidate.observation.branch}</span> @ <code>{candidate.observation.head_commit || "∅"}</code> · {t("{count} path(s)", { count: candidate.observation.changed_paths.length })} · <span className="muted">{candidate.recommended_action}</span></p>
      {candidate.violations.length > 0 && (
        <ul className="reason-list violation-list" data-testid={`candidate-violations-${candidate.candidate_id}`}>
          {candidate.violations.map((violation, index) => (
            <li key={`${violation.rule}-${index}`}><StatusBadge status="BLOCKED" label={violation.rule} /> {violation.path && <code>{violation.path}</code>} {violation.detail}</li>
          ))}
        </ul>
      )}
      {open && (
        <table className="topology-table gate-table" data-testid={`candidate-gates-${candidate.candidate_id}`}>
          <thead><tr><th>{t("Gate")}</th><th>{t("Result")}</th><th>{t("Detail")}</th></tr></thead>
          <tbody>
            {candidate.gates.map((gate) => (
              <tr key={gate.gate} data-testid={`gate-${candidate.candidate_id}-${gate.gate}`}>
                <td><code>{gate.gate}</code></td>
                <td><StatusBadge status={gate.status} /></td>
                <td>
                  {gate.detail}
                  {gate.paths && gate.paths.length > 0 && <div className="muted">{t("paths:")} {gate.paths.map((path) => <code key={path}>{path} </code>)}</div>}
                  {gate.missing && gate.missing.length > 0 && <div className="muted">{t("missing:")} {gate.missing.join("; ")}</div>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
      {candidate.human_decision && (
        <p className="topology-reason" data-testid={`candidate-human-${candidate.candidate_id}`}>
          <strong>{candidate.human_decision.decision}</strong> {t("by")} {candidate.human_decision.actor} ({candidate.human_decision.role}) {t("at")} {candidate.human_decision.at} — {candidate.human_decision.reason}
        </p>
      )}
      {decidable && (
        <form className="topology-form" data-testid={`candidate-decision-${candidate.candidate_id}`} onSubmit={(event) => { event.preventDefault(); void decide("ACCEPT"); }}>
          <div className="topology-form-grid">
            <label>{t("Deciding actor")}<input name="actor" value={actor} onChange={(event) => setActor(event.target.value)} required placeholder={t("must not be the run starter unless controls are waived")} /></label>
            <label>{t("Rationale")}<input name="rationale" value={rationale} onChange={(event) => setRationale(event.target.value)} required /></label>
          </div>
          <div className="topology-form-actions">
            <button type="submit" className="button-primary" disabled={controller.busy || candidate.verdict !== "CANDIDATE_ACCEPTABLE"} title={candidate.verdict !== "CANDIDATE_ACCEPTABLE" ? t("Only CANDIDATE_ACCEPTABLE can be accepted") : undefined}>{t("Accept candidate for promotion")}</button>
            <button type="button" className="button-secondary" disabled={controller.busy} onClick={() => void decide("REJECT")}>{t("Reject candidate")}</button>
          </div>
        </form>
      )}
    </div>
  );
}

// ---------------------------------------------------------------- section

export function CandidateReview({ view, controller }: CandidateReviewProps) {
  const { t } = useLocale();
  const order = view.work_order;
  const candidates = view.change.candidates ?? [];
  const [showForm, setShowForm] = useState(candidates.length === 0);
  if (!order) return null;
  const orderUsable = order.status === "ISSUED" && !view.staleness.stale;
  return (
    <div className="candidate-section" data-testid="candidate-review">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{t("CANDIDATE REVIEW · second human decision")}</p>
          <h3>{t("Agent candidates under {order}", { order: order.work_order_id })}</h3>
        </div>
        <span className="muted">{t("The gate blocks; it never accepts. Tests cannot override a policy violation.")}</span>
      </div>
      {candidates.slice().reverse().map((candidate) => (
        <CandidateCard key={candidate.candidate_id} candidate={candidate} changeID={view.change.change_id} controller={controller} decisionStale={view.staleness.stale} />
      ))}
      {orderUsable && !showForm && (
        <div className="topology-form-actions"><button type="button" className="button-primary" disabled={controller.busy} onClick={() => setShowForm(true)}>{t("Submit another candidate")}</button></div>
      )}
      {orderUsable && showForm && (
        <SubmitCandidateForm key={`submit-${candidates.length}`} order={order} changeID={view.change.change_id} busy={controller.busy} onSubmit={async (changeID, input) => { const ok = await controller.submitCandidate(changeID, input); if (ok) setShowForm(false); return ok; }} />
      )}
      {!orderUsable && <p className="reason">{t("The Work Order is {status}; no new candidate can be submitted under it.", { status: order.status })}</p>}
    </div>
  );
}
