import type { ReactNode } from "react";
import {
  Bell, Clock, Columns3, FilePen, LayoutDashboard, Link, List, Milestone as MilestoneIcon, MousePointer2, RefreshCw, Settings, Tag,
  User as UserIcon,
} from "lucide-react";
import type { ViewId } from "../../views";

/** Each view's icon, as the sidebar and the command palette show it. */
export const VIEW_ICONS: Record<ViewId, ReactNode> = {
  dashboard: <LayoutDashboard size={16} />,
  backlog: <List size={16} />,
  board: <Columns3 size={16} />,
  cycles: <RefreshCw size={16} />,
  milestones: <MilestoneIcon size={16} />,
  timeline: <Clock size={16} />,
  labels: <Tag size={16} />,
  dependencies: <Link size={16} />,
  "my-issues": <UserIcon size={16} />,
  inbox: <Bell size={16} />,
  drafts: <FilePen size={16} />,
  agent: <MousePointer2 size={16} />,
  preferences: <Settings size={16} />,
};
