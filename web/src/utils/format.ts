export function shortName(fullName: string): string {
  return fullName.split(" <")[0];
}

export interface ActorDisplay {
  principal: string;
  via?: string;
}

export function displayActor(createdBy: string, onBehalfOf?: string): ActorDisplay {
  if (onBehalfOf) {
    return {
      principal: shortName(onBehalfOf),
      via: shortName(createdBy),
    };
  }
  return { principal: shortName(createdBy) };
}

export function formatRelativeTime(dateStr: string): string {
  const now = Date.now();
  const then = new Date(dateStr).getTime();
  const diff = now - then;
  const minutes = Math.floor(diff / 60000);
  if (minutes < 1) return "just now";
  if (minutes < 60) return `${minutes}m ago`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 30) return `${days}d ago`;
  const months = Math.floor(days / 30);
  return `${months}mo ago`;
}

const SHORT_MONTHS = [
  "Jan", "Feb", "Mar", "Apr", "May", "Jun",
  "Jul", "Aug", "Sep", "Oct", "Nov", "Dec",
];

export function formatShortDate(dateStr: string): string {
  const parts = dateStr.slice(0, 10).split("-");
  if (parts.length === 3) {
    return `${SHORT_MONTHS[parseInt(parts[1], 10) - 1]} ${parseInt(parts[2], 10)}`;
  }
  const d = new Date(dateStr);
  return `${SHORT_MONTHS[d.getMonth()]} ${d.getDate()}`;
}

export function formatTriage(mins: number): string {
  if (mins <= 0) return "—";
  if (mins < 60) return `${mins}m`;
  const hours = mins / 60;
  if (hours < 24) return `${Math.round(hours)}h`;
  const days = hours / 24;
  if (days < 10) return `${days.toFixed(1)}d`;
  return `${Math.round(days)}d`;
}

export function stripMarkdown(text: string): string {
  return text
    .replace(/```[\s\S]*?```/g, " ")
    .replace(/`([^`]+)`/g, "$1")
    .replace(/^#{1,6}\s+/gm, "")
    .replace(/\*\*([^*]+)\*\*/g, "$1")
    .replace(/\*([^*]+)\*/g, "$1")
    .replace(/__([^_]+)__/g, "$1")
    .replace(/_([^_]+)_/g, "$1")
    .replace(/~~([^~]+)~~/g, "$1")
    .replace(/!\[[^\]]*\]\([^)]+\)/g, "")
    .replace(/\[([^\]]+)\]\([^)]+\)/g, "$1")
    .replace(/^>\s?/gm, "")
    .replace(/^[-*+]\s+/gm, "")
    .replace(/^\d+\.\s+/gm, "")
    .replace(/\s+/g, " ")
    .trim();
}

export function linkifyIssueIds(text: string, prefix: string): string {
  const escaped = prefix.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  return text.replace(
    new RegExp(`\\b(${escaped}[a-f0-9]{6})\\b`, "g"),
    "[$1](#/issues/$1)",
  );
}

export function formatDuration(hours: number): string {
  if (hours < 1) return `${Math.round(hours * 60)}m`;
  if (hours < 24) return `${Math.round(hours * 10) / 10}h`;
  const days = hours / 24;
  if (days < 7) return `${Math.round(days * 10) / 10}d`;
  const weeks = days / 7;
  if (weeks < 5) return `${Math.round(weeks * 10) / 10}w`;
  const months = days / 30;
  return `${Math.round(months * 10) / 10}mo`;
}
