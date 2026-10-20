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

/** A tool's icon, before its verb on the row. Decorative: the verb is the row's text. */
export default function ToolIcon({ icon, className = "" }: { icon: ToolIconId; className?: string }) {
  const Icon = ICONS[icon];
  return (
    <span aria-hidden className={`inline-flex shrink-0 ${className}`}>
      <Icon className="w-3.5 h-3.5" />
    </span>
  );
}
