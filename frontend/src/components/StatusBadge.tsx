interface StatusBadgeProps {
  status: string;
  label?: string;
}

export function StatusBadge({ status, label }: StatusBadgeProps) {
  const className = `status-badge status-${status.toLowerCase().replaceAll("_", "-")}`;
  return <span className={className}>{label ?? status}</span>;
}
