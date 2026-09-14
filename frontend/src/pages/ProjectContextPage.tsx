import type { ProjectEntry } from "../api/projectRegistry";
import { DocumentStatusList } from "../components/DocumentStatusList";
import { ReadinessSummary } from "../components/ReadinessSummary";
import { StatusBadge } from "../components/StatusBadge";

interface ProjectContextPageProps {
  entry: ProjectEntry;
}

function RelationshipList({ title, values, empty }: { title: string; values: string[]; empty: string }) {
  return (
    <section className="relationship-block">
      <h3>{title}</h3>
      {values.length === 0 ? <p className="muted">{empty}</p> : <ul>{values.map((value) => <li key={value}>{value}</li>)}</ul>}
    </section>
  );
}

export function ProjectContextPage({ entry }: ProjectContextPageProps) {
  const { project } = entry;
  return (
    <main className="context-column" data-testid="project-context-page">
      <section className="hero-panel">
        <div>
          <p className="eyebrow">PROJECT CONTEXT</p>
          <h1>{project.name}</h1>
          <p className="project-purpose">{project.purpose ?? "No declared purpose."}</p>
          <p className="source-root"><span className="muted">Observed source root:</span> <code>{project.root}</code></p>
        </div>
        <div className="hero-status">
          <StatusBadge status={entry.context.status} label={`Context ${entry.context.status}`} />
          <span className="muted">Observed {entry.observed_at}</span>
        </div>
      </section>

      <section className="panel relationship-panel" aria-labelledby="relationships-heading">
        <div className="section-heading">
          <div>
            <p className="eyebrow">DECLARED RELATIONSHIPS</p>
            <h2 id="relationships-heading">Project map</h2>
          </div>
          <span className="muted">No discovery performed</span>
        </div>
        <div className="relationship-grid">
          <RelationshipList title="Repositories" values={(entry.repositories ?? []).map((value) => value.url ?? value.provider ?? "UNDECLARED")} empty="No repositories declared." />
          <RelationshipList title="Services" values={(entry.services ?? []).map((value) => `${value.id} · ${value.runtime ?? "UNDECLARED"}`)} empty="No services declared." />
          <RelationshipList title="Environments" values={(entry.environments ?? []).map((value) => `${value.sequence ?? "?"}. ${value.id} · ${value.type ?? "UNDECLARED"}`)} empty="No environments declared." />
          <RelationshipList title="Decisions" values={(entry.decisions ?? []).map((value) => `${value.decision_id ?? "UNDECLARED"} · ${value.status ?? "UNKNOWN"}`)} empty="No decisions observed." />
        </div>
      </section>

      <DocumentStatusList documents={entry.documents ?? []} />
      <ReadinessSummary readiness={entry.readiness} />
    </main>
  );
}
