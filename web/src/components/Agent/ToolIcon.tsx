import {
  BookOpen,
  Bot,
  Eye,
  FileDiff,
  FileSliders,
  FileTerminal,
  GitPullRequestArrow,
  Link,
  MessageCircleQuestionMark,
  MessageCircleText,
  Paperclip,
  Pen,
  Play,
  Plus,
  Search,
  SquareDot,
  SquareTerminal,
  Unlink,
  Wrench,
  type LucideIcon,
} from "lucide-react";
import type { ToolIconId } from "./chat-thread-utils";

const ICONS: Record<ToolIconId, LucideIcon> = {
  read: Eye,
  edit: Pen,
  search: Search,
  shell: SquareTerminal,
  skill: BookOpen,
  add: Plus,
  comment: MessageCircleText,
  link: Link,
  unlink: Unlink,
  artifact: Paperclip,
  git_status: SquareDot,
  git_diff: FileDiff,
  spec: FileSliders,
  walkthrough: FileTerminal,
  start: Play,
  merge: GitPullRequestArrow,
  agent: Bot,
  question: MessageCircleQuestionMark,
  tool: Wrench,
};

/** A tool's icon in place of its verb. The verb stays as tooltip and accessible name. */
export default function ToolIcon({
  icon,
  label,
  className = "",
}: {
  icon: ToolIconId;
  label: string;
  className?: string;
}) {
  const Icon = ICONS[icon];
  return (
    <span
      role="img"
      aria-label={label}
      title={label}
      className={`inline-flex shrink-0 ${className}`}
    >
      <Icon className="w-3.5 h-3.5" aria-hidden />
    </span>
  );
}
