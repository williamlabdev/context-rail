// Change Decision Brief (change-decision-brief/v2) for people.
//
// The API sends structured values copied from the DecisionRecord; this view
// only chooses order, labels and language. Sections follow the questions a
// business owner asks — what was decided, what is and is not included, what
// risk was accepted, what must happen next, who decided — and the machine
// identity (ids, hashes, policy) sits in a collapsed traceability block.
// Data values are never passed through t().
import type { ChangeDecisionBrief, Lineage } from "../api/changes";
import { useLocale } from "../i18n";

export function LineageStrip({ lineage }: { lineage: Pick<Lineage, "decision_id" | "decision_version" | "source_snapshot_hash" | "topology_version" | "policy_version"> }) {
  const { t } = useLocale();
  return (
    <p className="lineage-strip" data-testid="lineage-strip">
      <code>{lineage.decision_id} v{lineage.decision_version}</code> · {t("snapshot")} <code title={lineage.source_snapshot_hash}>{lineage.source_snapshot_hash.slice(7, 19)}</code> · {t("topology v{version}", { version: lineage.topology_version })} · {t("policy")} {lineage.policy_version}
    </p>
  );
}

// Plain labels for the evidence codes a topology declares; unknown codes are
// shown as the code itself.
const evidenceLabels: Record<string, string> = {
  build: "The build succeeded",
  test: "Automated tests passed",
  smoke: "A smoke check passed in the environment",
  review: "The change was reviewed",
  "code-review": "The code was reviewed",
  "decision-record": "This decision is on record",
  "single-operator-controls": "Single-operator safeguards were applied",
  "release-approval": "A release approver signed off",
  "staging-receipt": "A staging release receipt exists",
};

const riskLabels: Record<string, string> = { low: "Low", medium: "Medium", high: "High" };

function formatMoment(value: string, locale: string): string {
  const parsed = new Date(value);
  if (Number.isNaN(parsed.getTime())) return value;
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium", timeStyle: "short" }).format(parsed);
}

export function ChangeBrief({ brief }: { brief: ChangeDecisionBrief }) {
  const { t, locale } = useLocale();
  const stale = brief.state === "STALE";
  const route = brief.route;
  const risk = riskLabels[brief.risk_level] ? t(riskLabels[brief.risk_level]) : brief.risk_level;

  return (
    <section className="dual-pane brief" data-testid="change-brief">
      <p className="eyebrow">{t("CHANGE DECISION BRIEF · for people")}</p>

      {stale && (
        <div className="topology-inline-error" data-testid="brief-stale" role="alert">
          <strong>{t("This decision no longer applies and must be made again.")}</strong> {brief.stale_reason}
        </div>
      )}

      <p className="brief-kicker">{stale ? t("Was approved for development") : t("Approved for development")}</p>
      {brief.owner_summary_missing ? (
        <div className="brief-missing" data-testid="brief-summary-missing">
          <strong>{t("No plain-language summary was supplied for this decision.")}</strong>{" "}
          {t("It was decided before the owner summary was required; the requester's technical objective is shown instead, unedited:")}
          <p className="brief-objective">{brief.objective}</p>
        </div>
      ) : (
        <h3 className="brief-headline" data-testid="brief-summary">{brief.owner_summary}</h3>
      )}
      <p className="brief-route" data-testid="brief-route">
        {t("Approach:")} <strong>{brief.selected.title}</strong>
        {" · "}{t("Next stop:")} <strong>{route.target_environment_id}</strong>
        {route.source_environment_id && <span className="muted"> ({route.source_environment_id} → {route.target_environment_id})</span>}
        {route.production_action === "forbidden" && <>{" · "}<span className="brief-no-prod">{t("Production is not authorized by this decision")}</span></>}
      </p>

      <div className={brief.unknowns.length > 0 ? "brief-warning" : "brief-section"} data-testid="brief-risk">
        <h4>{brief.unknowns.length > 0 ? t("Risk accepted knowingly") : t("Risk")}</h4>
        <p className="brief-line">{t("Risk level: {level}", { level: risk })}{brief.unknowns.length === 0 && <> · {t("no open unknowns were recorded")}</>}</p>
        {brief.unknowns.length > 0 && (
          <>
            <p className="brief-line">{t("The decision was made with these open questions:")}</p>
            <ul>{brief.unknowns.map((line) => <li key={line}>{line}</li>)}</ul>
          </>
        )}
      </div>

      <div className="brief-scope">
        <div className="brief-section" data-testid="brief-in-scope">
          <h4>{t("Included")}</h4>
          {brief.in_scope.length > 0 ? <ul>{brief.in_scope.map((line) => <li key={line}>{line}</li>)}</ul> : <p className="muted">{t("No explicit list; the objective and acceptance criteria bound the work.")}</p>}
        </div>
        <div className="brief-section" data-testid="brief-out-of-scope">
          <h4>{t("Not included")}</h4>
          {brief.out_of_scope.length > 0 ? <ul>{brief.out_of_scope.map((line) => <li key={line}>{line}</li>)}</ul> : <p className="muted">{t("No explicit list; the forbidden actions in the agent pack still apply.")}</p>}
        </div>
      </div>

      <div className="brief-section" data-testid="brief-next-gate">
        <h4>{t("Before it can reach {environment}", { environment: route.target_environment_id })}</h4>
        <ul>
          {brief.required_evidence.map((code) => (
            <li key={code}>{evidenceLabels[code] ? t(evidenceLabels[code]) : code} <code className="muted">{code}</code></li>
          ))}
          <li>{t("Someone other than the agent reviewed its change, and it only touched the files this decision allows.")}</li>
        </ul>
      </div>

      <div className="brief-section" data-testid="brief-decided-by">
        <h4>{t("Who decided")}</h4>
        <p className="brief-line">
          <strong>{brief.decided_by.actor}</strong> <span className="muted">({brief.decided_by.role})</span> · <time dateTime={brief.decided_by.at} title={brief.decided_by.at}>{formatMoment(brief.decided_by.at, locale)}</time>
        </p>
        {brief.decided_by.rationale && <blockquote className="brief-quote">{brief.decided_by.rationale}</blockquote>}
      </div>

      {brief.alternatives.length > 0 && (
        <div className="brief-section" data-testid="brief-alternatives">
          <h4>{t("Considered, not chosen")}</h4>
          <ul>{brief.alternatives.map((option) => <li key={option.id}><strong>{option.title}</strong> — {option.summary}</li>)}</ul>
        </div>
      )}

      <details className="brief-trace" data-testid="brief-trace">
        <summary>{t("Traceability (for audit)")}</summary>
        <dl>
          <dt>{t("Decision")}</dt><dd><code>{brief.lineage.decision_id} v{brief.lineage.decision_version}</code> · <code>{brief.lineage.change_id}</code> · <code>{brief.selected.id}</code></dd>
          {!brief.owner_summary_missing && <><dt>{t("Technical objective")}</dt><dd>{brief.objective}</dd></>}
          <dt>{t("Transition")}</dt><dd><code>{route.transition}</code> → <code>{route.target_ref}</code></dd>
          <dt>{t("Source snapshot")}</dt><dd><code>{brief.lineage.source_snapshot_hash}</code></dd>
          <dt>{t("Environment topology")}</dt><dd>v{brief.lineage.topology_version} · <code>{brief.lineage.topology_config_hash}</code></dd>
          <dt>{t("Policy")}</dt><dd><code>{brief.lineage.policy_version}</code></dd>
        </dl>
      </details>
      <LineageStrip lineage={brief.lineage} />
    </section>
  );
}
