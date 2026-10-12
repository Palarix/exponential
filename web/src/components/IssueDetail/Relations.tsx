import { useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode, type RefObject } from "react";
import { createPortal } from "react-dom";
import {
  ArrowDownLeft,
  ArrowUpRight,
  Ban,
  CircleSlash,
  Copy,
  CornerDownRight,
  Ellipsis,
  Flag,
  FlagOff,
  Link,
  Waypoints,
  OctagonMinus,
  SquarePlus,
  X,
} from "lucide-react";
import type { Issue } from "../../api/client";
import type { IssuePatch } from "../../api/mutations";
import { Popover, StatusIcon } from "../ui";
import Tooltip from "../ui/Tooltip";
import { Menu, MenuDivider, MenuFilter, MenuItem } from "../ui/Menu";
import { SubMenu } from "../ui/SubMenu";
import { IssuePreview } from "../IssueRef/IssuePreview";
import { createHoverIntent } from "./hover-intent";
import {
  LINK_KIND_MENU_LABEL,
  MORE_LINK_KINDS,
  PRIMARY_LINK_KINDS,
  addLink,
  changeLinkKind,
  changeToKinds,
  groupRelations,
  linkCandidates,
  parentOfCandidates,
  removeLink,
  subIssueOfCandidates,
  type LinkKind,
  type RelationSection,
} from "./relation-utils";

type SaveUpdate = (patch: Partial<Issue>, alsoUpdate?: IssuePatch[]) => void;

const PREVIEW_DELAY = 250;

const KIND_ICON: Record<LinkKind, ReactNode> = {
  relates_to: <Flag size={16} />,
  blocked_by: <OctagonMinus size={16} />,
  blocks: <Ban size={16} />,
  depends_on: <ArrowUpRight size={16} />,
  dependency_of: <ArrowDownLeft size={16} />,
  duplicates: <CircleSlash size={16} />,
  duplicated_by: <Copy size={16} />,
};

type AddKind = LinkKind | "parent_of" | "sub_issue_of";

const ADD_LABEL: Record<AddKind, string> = {
  ...LINK_KIND_MENU_LABEL,
  parent_of: "Parent of…",
  sub_issue_of: "Sub-issue of…",
};

function issueHash(id: string) {
  return `#/issues/${id}`;
}

// ─── Add popover ───────────────────────────────────────

export function RelationsAddPopover({
  issue,
  issues,
  anchorRef,
  onClose,
  saveUpdate,
}: {
  issue: Issue;
  issues: Issue[];
  anchorRef: RefObject<HTMLElement | null>;
  onClose: () => void;
  saveUpdate: SaveUpdate;
}) {
  const [kind, setKind] = useState<AddKind | null>(null);
  const [search, setSearch] = useState("");
  // Menu calls onClose after every item click; choosing a kind moves to
  // the issue step instead of closing the popover.
  const choosing = useRef(false);
  useEffect(() => { choosing.current = false; }, [kind]);

  const candidates = useMemo(() => {
    if (!kind) return [];
    if (kind === "sub_issue_of") return subIssueOfCandidates(issue, issues, search);
    if (kind === "parent_of") return parentOfCandidates(issue, issues, search);
    return linkCandidates(issue, issues, search);
  }, [kind, issue, issues, search]);

  const pick = (targetId: string) => {
    if (kind === "sub_issue_of") saveUpdate({ parent_id: targetId });
    else if (kind === "parent_of") saveUpdate({}, [{ issueId: targetId, patch: { parent_id: issue.id } }]);
    else if (kind) saveUpdate({ dependencies: addLink(issue.id, issue.dependencies || [], targetId, kind) });
    onClose();
  };

  // Menu calls onClose after the click, which closes the popover.
  const removeParent = () => saveUpdate({ parent_id: "" });

  const chooseKind = (k: AddKind) => {
    choosing.current = true;
    setSearch("");
    setKind(k);
  };

  const closeKindMenu = () => {
    if (choosing.current) choosing.current = false;
    else onClose();
  };

  return (
    <Popover anchorRef={anchorRef} onClose={onClose}>
      {kind === null ? (
        // Keyed so each step remounts and registers its Escape after the popover's.
        <Menu key="kinds" onClose={closeKindMenu} aria-label="Add relation">
          {issue.parent_id && <MenuItem label="Remove parent" icon={<X size={16} />} onClick={removeParent} />}
          {issue.parent_id && <MenuDivider />}
          <MenuItem label={ADD_LABEL.parent_of} icon={<SquarePlus size={16} />} onClick={() => chooseKind("parent_of")} />
          <MenuItem label={ADD_LABEL.sub_issue_of} icon={<CornerDownRight size={16} />} onClick={() => chooseKind("sub_issue_of")} />
          {PRIMARY_LINK_KINDS.map(k => (
            <MenuItem key={k} label={ADD_LABEL[k]} icon={KIND_ICON[k]} onClick={() => chooseKind(k)} />
          ))}
          <MenuDivider />
          <SubMenu label="More" icon={<Ellipsis size={16} />}>
            {MORE_LINK_KINDS.map(k => (
              <MenuItem key={k} label={ADD_LABEL[k]} icon={KIND_ICON[k]} onClick={() => chooseKind(k)} />
            ))}
          </SubMenu>
        </Menu>
      ) : (
        <Menu key={kind} onClose={() => setKind(null)} aria-label={ADD_LABEL[kind]} maxHeight="20rem" className="w-96">
          <MenuFilter value={search} onChange={setSearch} placeholder={`${ADD_LABEL[kind].replace("…", "")} issue…`} />
          {candidates.slice(0, 30).map(c => (
            <MenuItem
              key={c.id}
              label={c.title}
              truncate
              icon={
                <span className="flex items-center gap-2">
                  <StatusIcon status={c.status} size={12} isInferred={c.is_inferred} />
                  <span className="text-xs text-[var(--color-text-muted)] font-mono whitespace-nowrap">{c.id}</span>
                </span>
              }
              onClick={() => pick(c.id)}
            />
          ))}
          {candidates.length === 0 && (
            <MenuItem label="No matching issues" disabled />
          )}
        </Menu>
      )}
    </Popover>
  );
}

// ─── Relations card ────────────────────────────────────

export function RelationsCard({
  issue,
  issues,
  saveUpdate,
}: {
  issue: Issue;
  issues: Issue[];
  saveUpdate: SaveUpdate;
}) {
  const sections = useMemo(() => groupRelations(issue, issues), [issue, issues]);
  if (sections.length === 0) return null;

  const deps = issue.dependencies || [];
  const remove = (section: RelationSection, targetId: string) => {
    saveUpdate({ dependencies: removeLink(deps, targetId, section.key) });
  };
  const change = (targetId: string, kind: LinkKind) => {
    saveUpdate({ dependencies: changeLinkKind(issue.id, deps, targetId, kind) });
  };

  return (
    <div className="relative rounded-[var(--radius-md)] bg-[var(--color-surface-2)] border border-[var(--color-border-default)] px-4 py-3 space-y-3">
      <Tooltip content="Show in dependency graph">
        <a
          href={`#/dependencies/${issue.id}`}
          aria-label="Show in dependency graph"
          className="absolute top-2 right-2 p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface)] transition-colors"
        >
          <Waypoints size={14} />
        </a>
      </Tooltip>
      {sections.map(section => (
        <div key={section.key}>
          <div className="text-xs font-medium text-[var(--color-text-muted)] mb-1.5">{section.label}</div>
          <div className="flex flex-col items-start gap-1">
            {section.items.map(item => (
              <RelationChip
                key={item.targetId}
                targetId={item.targetId}
                target={item.target}
                currentKind={section.key}
                onRemove={() => remove(section, item.targetId)}
                onChange={kind => change(item.targetId, kind)}
              />
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

// ─── Chip ──────────────────────────────────────────────

function RelationChip({
  targetId,
  target,
  currentKind,
  onRemove,
  onChange,
}: {
  targetId: string;
  target?: Issue;
  currentKind: LinkKind;
  onRemove: () => void;
  onChange: (kind: LinkKind) => void;
}) {
  const chipRef = useRef<HTMLDivElement>(null);
  const [preview, setPreview] = useState(false);
  const [menuAt, setMenuAt] = useState<{ x: number; y: number } | null>(null);
  const [intent] = useState(() => createHoverIntent(PREVIEW_DELAY, setPreview));
  useEffect(() => () => intent.cancel(), [intent]);

  const openMenu = (e: React.MouseEvent) => {
    e.preventDefault();
    intent.leave();
    setMenuAt({ x: e.clientX, y: e.clientY });
  };

  // Preview and menu are siblings of the chip: React events bubble through
  // portals, so as children they would retrigger the chip's hover handlers.
  return (
    <>
      <div
        ref={chipRef}
        className="group flex items-center gap-1 max-w-full"
        onMouseEnter={() => target && !menuAt && intent.enter()}
        onMouseLeave={() => intent.leave()}
        onContextMenu={openMenu}
      >
        <a
          href={target ? issueHash(target.id) : undefined}
          onFocus={() => target && intent.enter()}
          onBlur={() => intent.leave()}
          onClick={() => intent.leave()}
          className="flex items-center gap-2 min-w-0 text-sm rounded-full px-2.5 py-1 group-hover:bg-[var(--color-hover-surface)] transition-colors outline-none focus-visible:ring-1 focus-visible:ring-[var(--color-border-focus)]"
        >
          {target ? (
            <>
              <StatusIcon status={target.status} size={12} isInferred={target.is_inferred} />
              <span className="text-[var(--color-text-primary)] truncate">{target.title}</span>
            </>
          ) : (
            <span className="text-[var(--color-text-muted)] italic truncate">{targetId} (deleted)</span>
          )}
        </a>
        <button
          type="button"
          aria-label="Remove relation"
          onClick={() => { intent.leave(); onRemove(); }}
          className="shrink-0 p-1 rounded-full text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-2)] opacity-0 group-hover:opacity-100 focus-visible:opacity-100 transition-opacity"
        >
          <X size={12} />
        </button>
      </div>
      {preview && target && <IssuePreview anchorRef={chipRef} issue={target} />}
      {menuAt && (
        <RelationContextMenu
          x={menuAt.x}
          y={menuAt.y}
          targetId={target?.id ?? targetId}
          currentKind={currentKind}
          onClose={() => setMenuAt(null)}
          onRemove={onRemove}
          onChange={onChange}
        />
      )}
    </>
  );
}

// ─── Context menu ──────────────────────────────────────

function RelationContextMenu({
  x,
  y,
  targetId,
  currentKind,
  onClose,
  onRemove,
  onChange,
}: {
  x: number;
  y: number;
  targetId: string;
  currentKind: LinkKind;
  onClose: () => void;
  onRemove: () => void;
  onChange: (kind: LinkKind) => void;
}) {
  const menuRef = useRef<HTMLDivElement>(null);

  useLayoutEffect(() => {
    const el = menuRef.current;
    if (!el) return;
    const rect = el.getBoundingClientRect();
    const pad = 8;
    el.style.top = `${Math.max(pad, Math.min(y, window.innerHeight - rect.height - pad))}px`;
    el.style.left = `${Math.max(pad, Math.min(x, window.innerWidth - rect.width - pad))}px`;
  }, [x, y]);

  useEffect(() => {
    let armed = false;
    const armTimer = requestAnimationFrame(() => { armed = true; });
    const handleDown = (e: MouseEvent) => {
      if (!armed) return;
      const t = e.target as Node;
      if (menuRef.current?.contains(t)) return;
      if (t instanceof Element && t.closest("[data-submenu-flyout]")) return;
      onClose();
    };
    document.addEventListener("mousedown", handleDown);
    return () => { cancelAnimationFrame(armTimer); document.removeEventListener("mousedown", handleDown); };
  }, [onClose]);

  const kinds = changeToKinds(currentKind);
  const copyLink = () => {
    const { origin, pathname } = window.location;
    navigator.clipboard.writeText(`${origin}${pathname}${issueHash(targetId)}`);
  };

  return createPortal(
    <div ref={menuRef} style={{ position: "fixed", top: y, left: x }} className="z-[100]">
      <Menu onClose={onClose} aria-label="Relation actions">
        <SubMenu label="Change to" icon={<Flag size={16} />}>
          {kinds.primary.map(k => (
            <MenuItem key={k} label={LINK_KIND_MENU_LABEL[k].replace("…", "")} icon={KIND_ICON[k]} onClick={() => onChange(k)} />
          ))}
          <MenuDivider />
          <SubMenu label="More" icon={<Ellipsis size={16} />}>
            {kinds.more.map(k => (
              <MenuItem key={k} label={LINK_KIND_MENU_LABEL[k].replace("…", "")} icon={KIND_ICON[k]} onClick={() => onChange(k)} />
            ))}
          </SubMenu>
        </SubMenu>
        <MenuItem label="Remove relation" icon={<FlagOff size={16} />} onClick={onRemove} />
        <MenuDivider />
        <MenuItem label="Copy link" icon={<Link size={16} />} onClick={copyLink} />
      </Menu>
    </div>,
    document.body,
  );
}
