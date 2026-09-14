import type { ReadinessSet } from "../api/projectRegistry";
import { StatusBadge } from "./StatusBadge";

const readinessLabels: Array<[keyof ReadinessSet, string]> = [
  ["ready_for_decision", "Decision inputs"],
  ["ready_for_local_development", "Local development"],
  ["ready_for_cloud_testing", "Cloud testing"],
  ["ready_for_staging", "Staging inputs"],
  ["staging_verified", "Staging verification"],
  ["production", "Production threshold"],
];

interface ReadinessSummaryProps {
  readiness: ReadinessSet;
}

export function ReadinessSummary({ readiness }: ReadinessSummaryProps) {
  return (
    <section className="panel" aria-labelledby="readiness-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">GOVERNANCE SIGNALS</p>
          <h2 id="readiness-heading">Readiness</h2>
        </div>
        <span className="muted">Informational; not authorization</span>
      </div>
      <div className="readiness-grid">
        {readinessLabels.map(([key, label]) => {
          const value = readiness[key];
          if (!value) return null;
          return (
            <article className="readiness-card" data-testid={`readiness-${key}`} key={key}>
              <div className="readiness-card-heading">
                <span>{label}</span>
                <StatusBadge status={value.status} />
              </div>
              {value.reasons.length > 0 ? (
                <ul className="reason-list">
                  {value.reasons.map((reason) => <li key={reason}>{reason}</li>)}
                </ul>
              ) : (
                <p className="muted">No blocking reason observed.</p>
              )}
            </article>
          );
        })}
      </div>
    </section>
  );
}
