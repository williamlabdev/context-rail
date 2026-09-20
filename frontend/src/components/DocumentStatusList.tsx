import type { DocumentStatus } from "../api/projectRegistry";
import { StatusBadge } from "./StatusBadge";
import { useLocale } from "../i18n";

interface DocumentStatusListProps {
  documents: DocumentStatus[];
}

export function DocumentStatusList({ documents }: DocumentStatusListProps) {
  const { t } = useLocale();
  return (
    <section className="panel" aria-labelledby="documents-heading">
      <div className="section-heading">
        <div>
          <p className="eyebrow">{t("PROVENANCE")}</p>
          <h2 id="documents-heading">{t("Documents / AI Context")}</h2>
        </div>
        <span className="muted">{t("Read-only observation")}</span>
      </div>
      <div className="document-list">
        {documents.length === 0 ? (
          <p className="muted">{t("No declared documents.")}</p>
        ) : (
          documents.map((document) => {
            const derived = document.source_of_truth === false;
            return (
              <article className="document-row" data-testid={`document-${document.path}`} key={document.path}>
                <div className="document-main">
                  <div className="document-title-line">
                    <span className="document-kind">{derived ? t("DERIVED") : t("SOURCE")}</span>
                    <code>{document.path}</code>
                  </div>
                  <span className="muted">{document.kind ?? "document"}</span>
                  {document.stale_sources && document.stale_sources.length > 0 ? (
                    <span className="reason">{t("Stale source: {sources}", { sources: document.stale_sources.join(", ") })}</span>
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
