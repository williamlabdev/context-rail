import type { DocumentStatus } from "../api/projectRegistry";
import { StatusBadge } from "./StatusBadge";

interface DocumentStatusListProps {
  documents: DocumentStatus[];
}

export function DocumentStatusList({ documents }: DocumentStatusListProps) {
  return (
    <section className="panel" aria-labelledby="documents-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">PROVENANCE</p>
          <h2 id="documents-heading">Documents / AI Context</h2>
        </div>
        <span className="muted">Read-only observation</span>
      </div>
      <div className="document-list">
        {documents.length === 0 ? (
          <p className="muted">No declared documents.</p>
        ) : (
          documents.map((document) => {
            const derived = document.source_of_truth === false;
            return (
              <article className="document-row" data-testid={`document-${document.path}`} key={document.path}>
                <div className="document-main">
                  <div className="document-title-line">
                    <span className="document-kind">{derived ? "DERIVED" : "SOURCE"}</span>
                    <code>{document.path}</code>
                  </div>
                  <span className="muted">{document.kind ?? "document"}</span>
                  {document.stale_sources && document.stale_sources.length > 0 ? (
                    <span className="reason">Stale source: {document.stale_sources.join(", ")}</span>
                  ) : null}
                </div>
                <StatusBadge status={document.status} />
              </article>
            );
          })
        )}
      </div>
    </section>
  );
}
