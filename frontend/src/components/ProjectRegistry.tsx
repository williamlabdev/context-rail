import type { ProjectEntry } from "../api/projectRegistry";
import { StatusBadge } from "./StatusBadge";
import { useLocale } from "../i18n";

interface ProjectRegistryProps {
  projects: ProjectEntry[];
  selectedProjectID: string | null;
  onSelect: (projectID: string) => void;
}

export function ProjectRegistry({ projects, selectedProjectID, onSelect }: ProjectRegistryProps) {
  const { t } = useLocale();
  return (
    <section className="panel registry-panel" aria-labelledby="registry-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{t("PROJECT REGISTRY")}</p>
          <h2 id="registry-heading">{t("Projects")}</h2>
        </div>
        <span className="count-label">{t("{count} configured", { count: projects.length })}</span>
      </div>
      <div className="project-list" role="list">
        {projects.map((entry) => {
          const projectID = entry.project.id;
          const selected = projectID === selectedProjectID;
          const readiness = entry.readiness.ready_for_decision?.status ?? "UNKNOWN";
          return (
            <button
              className={`project-card${selected ? " selected" : ""}`}
              type="button"
              role="listitem"
              aria-pressed={selected}
              data-testid={`project-card-${projectID}`}
              key={projectID}
              onClick={() => onSelect(projectID)}
            >
              <span className="project-card-title">{entry.project.name}</span>
              <span className="project-card-id">{projectID}</span>
              <span className="project-card-meta">
                <StatusBadge status={entry.context.status} prefix={t("Context")} />
                <StatusBadge status={readiness} prefix={t("Decision")} />
              </span>
            </button>
          );
        })}
      </div>
    </section>
  );
}
