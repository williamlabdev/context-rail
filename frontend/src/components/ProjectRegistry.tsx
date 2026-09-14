import type { ProjectEntry } from "../api/projectRegistry";
import { StatusBadge } from "./StatusBadge";

interface ProjectRegistryProps {
  projects: ProjectEntry[];
  selectedProjectID: string | null;
  onSelect: (projectID: string) => void;
}

export function ProjectRegistry({ projects, selectedProjectID, onSelect }: ProjectRegistryProps) {
  return (
    <section className="panel registry-panel" aria-labelledby="registry-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">PROJECT REGISTRY</p>
          <h2 id="registry-heading">Projects</h2>
        </div>
        <span className="count-label">{projects.length} configured</span>
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
                <StatusBadge status={entry.context.status} label={`Context ${entry.context.status}`} />
                <StatusBadge status={readiness} label={`Decision ${readiness}`} />
              </span>
            </button>
          );
        })}
      </div>
    </section>
  );
}
