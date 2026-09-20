import { useLocale } from "../i18n";

interface StatusBadgeProps {
  /** Machine status code (or an identifier such as RR-001). Never translated. */
  status: string;
  /** Optional static prefix, already translated by the caller (e.g. "Context"). */
  prefix?: string;
  /** Legacy: a fully composed label; rendered as-is next to the code. */
  label?: string;
}

/**
 * UI-18: the main screen shows a human label for a machine status code while
 * the code itself stays visible (small, monospace) so the technical detail
 * can always be read back. Unknown codes and identifiers render verbatim.
 */
export function StatusBadge({ status, prefix, label }: StatusBadgeProps) {
  const { statusLabel } = useLocale();
  const className = `status-badge status-${status.toLowerCase().replaceAll("_", "-")}`;
  const human = label ?? statusLabel(status);
  if (!human) {
    return <span className={className} data-code={status}>{prefix ? `${prefix} ${status}` : status}</span>;
  }
  return (
    <span className={className} data-code={status} title={status}>
      <span className="status-label">{prefix ? `${prefix} ${human}` : human}</span>
      <code className="status-code">{status}</code>
    </span>
  );
}
