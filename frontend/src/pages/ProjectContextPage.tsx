import type { ProjectEntry } from "../api/projectRegistry";
import { currentVersion as currentTopologyVersion, staleDecisions } from "../api/topology";
import { ChangesPanel } from "../components/ChangesPanel";
import { ReleasesPanel } from "../components/ReleasesPanel";
import { DocumentStatusList } from "../components/DocumentStatusList";
import { DocumentsPanel } from "../components/DocumentsPanel";
import { EnvironmentTopology } from "../components/EnvironmentTopology";
import { ReadinessSummary } from "../components/ReadinessSummary";
import { StatusBadge } from "../components/StatusBadge";
import type { TopologyController } from "../state/useTopology";
import type { ChangesController } from "../state/useChanges";
import type { ReleasesController } from "../state/useReleases";
import type { DocumentsController } from "../state/useDocuments";
import { useLocale } from "../i18n";

interface ProjectContextPageProps {
  entry: ProjectEntry;
  /** Versioned topology controller; omitted in read-only/test renders. */
  topology?: TopologyController;
  /** Change ledger controller; omitted in read-only/test renders. */
  changes?: ChangesController;
  /** Release ledger controller; omitted in read-only/test renders. */
  releases?: ReleasesController;
  /** Document baseline / Context Pack controller; omitted in read-only/test renders. */
  documents?: DocumentsController;
}

function RelationshipList({ title, values, empty }: { title: string; values: string[]; empty: string }) {
  return (
    <section className="relationship-block">
      <h3>{title}</h3>
      {values.length === 0 ? <p className="muted">{empty}</p> : <ul>{values.map((value) => <li key={value}>{value}</li>)}</ul>}
    </section>
  );
}

export function ProjectContextPage({ entry, topology, changes, releases, documents }: ProjectContextPageProps) {
  const { t } = useLocale();
  const { project } = entry;
  const topologyEnvironments = topology?.state ? currentTopologyVersion(topology.state)?.environments ?? [] : [];
  const stale = staleDecisions(topology?.state ?? null);
  const decisionLabels = (entry.decisions ?? []).map((value) => {
    const id = value.decision_id ?? "UNDECLARED";
    const invalidation = stale.get(id);
    return invalidation
      ? `${id} · ${value.status ?? "UNKNOWN"} → STALE (${t("topology v{version}", { version: invalidation.topology_version })})`
      : `${id} · ${value.status ?? "UNKNOWN"}`;
  });
  return (
    <main className="context-column" data-testid="project-context-page">
      <section className="hero-panel">
        <div>
          <p className="eyebrow">{t("PROJECT CONTEXT")}</p>
          <h1>{project.name}</h1>
          <p className="project-purpose">{project.purpose ?? t("No declared purpose.")}</p>
          <p className="source-root"><span className="muted">{t("Observed source root:")}</span> <code>{project.root}</code></p>
        </div>
        <div className="hero-status">
          <StatusBadge status={entry.context.status} prefix={t("Context")} />
          <span className="muted">{t("Observed {at}", { at: entry.observed_at })}</span>
        </div>
      </section>

      <section className="panel relationship-panel" aria-labelledby="relationships-heading">
        <div className="section-heading">
          <div>
            <p className="eyebrow">{t("DECLARED RELATIONSHIPS")}</p>
            <h2 id="relationships-heading">{t("Project map")}</h2>
          </div>
          <span className="muted">{t("No discovery performed")}</span>
        </div>
        <div className="relationship-grid">
          <RelationshipList title={t("Repositories")} values={(entry.repositories ?? []).map((value) => value.url ?? value.provider ?? "UNDECLARED")} empty={t("No repositories declared.")} />
          <RelationshipList title={t("Services")} values={(entry.services ?? []).map((value) => `${value.id} · ${value.runtime ?? "UNDECLARED"}`)} empty={t("No services declared.")} />
          <RelationshipList title={t("Environments")} values={(entry.environments ?? []).map((value) => `${value.sequence ?? "?"}. ${value.id} · ${value.type ?? "UNDECLARED"}`)} empty={t("No environments declared.")} />
          <RelationshipList title={t("Decisions")} values={decisionLabels} empty={t("No decisions observed.")} />
        </div>
      </section>

      {changes && <ChangesPanel controller={changes} environments={topologyEnvironments} />}
      {releases && changes && <ReleasesPanel controller={releases} changes={changes.changes} environments={topologyEnvironments} />}
      {topology && <EnvironmentTopology controller={topology} />}
      {documents ? <DocumentsPanel controller={documents} /> : <DocumentStatusList documents={entry.documents ?? []} />}
      <ReadinessSummary readiness={entry.readiness} />
    </main>
  );
}
