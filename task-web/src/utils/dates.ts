export function todayISO(): string {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, '0');
  const day = String(d.getDate()).padStart(2, '0');
  return `${y}-${m}-${day}`;
}

export function daysUntil(dueDate?: string | null): number | null {
  if (!dueDate) return null;
  const parts = dueDate.split('-');
  if (parts.length !== 3) return null;
  const [y, m, d] = parts.map((v) => parseInt(v, 10));
  if (!y || !m || !d) return null;
  const target = new Date(y, m - 1, d);
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());
  const diffMs = target.getTime() - today.getTime();
  return Math.round(diffMs / (1000 * 60 * 60 * 24));
}

export function formatDue(dueDate?: string | null): string {
  if (!dueDate) return '—';
  return dueDate;
}

export function humanDaysRemaining(dueDate?: string | null): string {
  const days = daysUntil(dueDate);
  if (days === null) return '';
  if (days === 0) return 'Due today';
  if (days > 0) return `In ${days} day${days === 1 ? '' : 's'}`;
  const overdue = Math.abs(days);
  return `${overdue} day${overdue === 1 ? '' : 's'} overdue`;
}

export function isOverdue(task: { dueDate?: string | null; completed?: boolean }): boolean {
  if (task.completed) return false;
  const d = daysUntil(task.dueDate);
  return d !== null && d < 0;
}
