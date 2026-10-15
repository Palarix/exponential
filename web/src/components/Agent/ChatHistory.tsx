import { useMemo, useRef, useState, type ReactNode } from "react";
import { useQueryClient } from "@tanstack/react-query";
import * as TooltipPrimitive from "@radix-ui/react-tooltip";
import { Archive, ArchiveRestore, MoreHorizontal, Pencil, SquarePen, Trash2, X } from "lucide-react";
import { renameChat, type ChatMeta } from "../../api/chat";
import { useAppNav } from "../../app/hooks";
import { IssueRef } from "../IssueRef/IssueRef";
import { cn } from "../../utils/cn";
import { formatRelativeTime } from "../../utils/format";
import { Button, ConfirmDialog, InlineDropdown, Menu, MenuItem, MenuLabel, Popover, SearchInput, Tabs, TopBar, useToast } from "../ui";
import { agentQueryKeys, useAgentChat } from "./agent-context";
import { archiveBlocker, filterHistory, formatCost, formatTokens, summarize, type HistoryPeriod } from "./chat-history-utils";
import ProviderIcon from "./ProviderIcon";
import { chatStateIcon, idleChatIcon } from "./recent-chats";
import { useArchiveChat } from "./useArchiveChat";

const SHOW_TABS = { active: { label: "Active" }, archived: { label: "Archived" } } as const;

const PERIODS = [
  { value: "7d", label: "Last 7 days" },
  { value: "30d", label: "Last 30 days" },
  { value: "all", label: "All time" },
];

const cell = "px-3 py-2 whitespace-nowrap";

// The idle state: a muted ring in the icon slot, quieter than the state dots, so titles line up.
const IDLE_DOT = (
  <span title="Idle" aria-label="Idle" role="img" className="shrink-0 inline-flex items-center justify-center w-3.5 h-3.5">
    <span className="w-1.5 h-1.5 rounded-full border border-[var(--color-text-muted)] opacity-60" />
  </span>
);

// The checkbox fills the state icon slot it stacks on.
const checkbox = "absolute inset-0 m-0 w-3.5 h-3.5 accent-[var(--color-accent-primary)]";

/**
 * Every chat in a table, with search, filters, token and cost totals, and
 * archive, unarchive, rename and delete for one chat or a selection.
 */
export default function ChatHistory({ issueId }: { issueId: string | null }) {
  const { allChats, seen, status } = useAgentChat();
  const { openChatView, openChatHistory } = useAppNav();
  const { archive, unarchive, remove } = useArchiveChat();
  const showToast = useToast();
  const [query, setQuery] = useState("");
  const [show, setShow] = useState<keyof typeof SHOW_TABS>("active");
  const [agentFilter, setAgentFilter] = useState("");
  const [period, setPeriod] = useState<HistoryPeriod>("all");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [confirmDelete, setConfirmDelete] = useState<ChatMeta[] | null>(null);
  const [renaming, setRenaming] = useState<{ id: string; draft: string } | null>(null);
  const qc = useQueryClient();
  // The time the period filter counts back from; set once per mount.
  const [now] = useState(() => Date.now());

  const rows = useMemo(
    () => filterHistory(allChats, { query, archived: show === "archived", agent: agentFilter, issueId: issueId ?? "", period }, now),
    [allChats, query, show, agentFilter, issueId, period, now],
  );
  const summary = summarize(rows);
  // The selection only counts rows still on screen.
  const picked = rows.filter((c) => selected.has(c.id));
  const allPicked = rows.length > 0 && picked.length === rows.length;
  const selecting = picked.length > 0;

  const agentOptions = [{ value: "", label: "All agents" }, ...(status?.providers ?? []).map((p) => ({ value: p.id, label: p.name }))];

  const toggle = (id: string) => setSelected((s) => {
    const next = new Set(s);
    if (!next.delete(id)) next.add(id);
    return next;
  });

  const saveTitle = async (c: ChatMeta) => {
    if (!renaming) return;
    const next = renaming.draft.trim();
    setRenaming(null);
    if (next === "" || next === c.title) return;
    try {
      await renameChat(c.id, next);
      qc.invalidateQueries({ queryKey: agentQueryKeys.chats });
    } catch {
      showToast("Could not rename the chat", { variant: "error" });
    }
  };

  const archivePicked = async () => {
    const ok = picked.filter((c) => !archiveBlocker(c));
    const skipped = picked.length - ok.length;
    if (ok.length > 0) await archive(ok.map((c) => c.id));
    if (skipped > 0) showToast(`Skipped ${skipped} running or Work Mode ${skipped === 1 ? "chat" : "chats"}`);
    setSelected(new Set());
  };

  return (
    <TooltipPrimitive.Provider delayDuration={150}>
    <TopBar
      className="px-6 border-[var(--color-border-ai-subtle)]"
      left={<h1 className="text-sm font-medium text-[var(--color-text-primary)]">All chats</h1>}
      center={<SearchInput value={query} onChange={setQuery} placeholder="Search chats…" />}
      right={<Button variant="ghost" size="sm" icon={<SquarePen size={14} />} onClick={() => openChatView(null)}>New chat</Button>}
    />
    <div className="flex-1 overflow-y-auto scroll-stable">
      <div className="px-6 py-6 flex flex-col gap-4">
        <div className="flex flex-wrap items-center gap-3">
          <InlineDropdown borderless placeholder="All agents" options={agentOptions} value={agentFilter} onChange={setAgentFilter} />
          <InlineDropdown borderless placeholder="All time" options={PERIODS} value={period} onChange={(v) => setPeriod(v as HistoryPeriod)} />
          {issueId && (
            <span className="flex items-center gap-1 h-7 pl-2 pr-1 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-2)] text-xs">
              <span className="text-[var(--color-text-muted)]">Issue</span>
              <span className="font-mono text-[var(--color-text-primary)]">{issueId}</span>
              <button type="button" aria-label="Clear issue filter" onClick={() => openChatHistory(null)} className="p-0.5 rounded text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)]">
                <X className="w-3 h-3" />
              </button>
            </span>
          )}
          <div className="h-8 ml-auto">
            <Tabs items={SHOW_TABS} activeId={show} onChange={(id) => { setShow(id); setSelected(new Set()); }} />
          </div>
        </div>

        <dl className="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <Stat label="Chats" value={String(summary.count)} />
          <Stat label="Tokens in" value={formatTokens(summary.tokensIn)} />
          <Stat label="Tokens out" value={formatTokens(summary.tokensOut)} />
          <Stat
            label="Cost"
            value={formatCost(summary.costUSD, summary.costKnown, summary.costPartial)}
            hint={summary.costPartial ? "Some chats report no cost" : undefined}
          />
        </dl>

        {/* Always in the layout, hidden without a selection, so selecting never moves the table. */}
        <div
          aria-hidden={!selecting}
          className={`flex items-center gap-2 h-9 pl-3 pr-0 rounded-[var(--radius-md)] bg-[var(--color-surface-ai-2)] text-sm ${selecting ? "" : "invisible"}`}
        >
          <span className="text-[var(--color-text-secondary)]">{picked.length} selected</span>
          <span className="flex-1" />
          {show === "active" ? (
            <Button variant="ghost" size="sm" icon={<Archive size={14} />} onClick={() => void archivePicked()}>Archive</Button>
          ) : (
            <Button variant="ghost" size="sm" icon={<ArchiveRestore size={14} />} onClick={() => void unarchive(picked.map((c) => c.id)).then(() => setSelected(new Set()))}>Unarchive</Button>
          )}
          <Button variant="ghost" size="sm" icon={<Trash2 size={14} />} onClick={() => setConfirmDelete(picked)}>Delete</Button>
          <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>Clear selection</Button>
        </div>

        {rows.length === 0 ? (
          <div className="py-16 text-center text-sm text-[var(--color-text-muted)]">
            {show === "archived" ? "No archived chats" : "No chats"}{query || agentFilter || issueId || period !== "all" ? " match these filters" : ""}
          </div>
        ) : (
          <table className="w-full text-sm">
            <thead className="text-xs text-[var(--color-text-muted)]">
              <tr className="group border-b border-[var(--color-border-ai-subtle)]">
                <th className={`${cell} text-left font-medium`}>
                  <span className="flex items-center gap-2">
                    <SelectSlot
                      icon={null}
                      label="Select all chats"
                      selecting={selecting}
                      checked={allPicked}
                      onChange={() => setSelected(allPicked ? new Set() : new Set(rows.map((c) => c.id)))}
                    />
                    Chat
                  </span>
                </th>
                <th className={`${cell} text-left font-medium`}>Agent</th>
                <th className={`${cell} text-left font-medium`}>Issues</th>
                <th className={`${cell} text-left font-medium`}>Last activity</th>
                <th className="w-8" />
              </tr>
            </thead>
            <tbody>
              {rows.map((c) => (
                <tr
                  key={c.id}
                  onClick={() => openChatView(c.id)}
                  className="group border-b border-[var(--color-border-ai-subtle)] cursor-pointer hover:bg-[var(--color-hover-surface-ai-1)] transition-colors"
                >
                  <td className="px-3 py-2 max-w-0 w-full">
                    <span className="flex items-center gap-2 min-w-0">
                      <SelectSlot
                        icon={chatStateIcon(c, seen, "w-3.5 h-3.5", idleChatIcon(c, "w-3.5 h-3.5", IDLE_DOT))}
                        label={`Select ${c.title || "Untitled chat"}`}
                        selecting={selecting}
                        checked={selected.has(c.id)}
                        onChange={() => toggle(c.id)}
                      />
                      {renaming?.id === c.id ? (
                        <input
                          autoFocus
                          aria-label="Chat title"
                          value={renaming.draft}
                          maxLength={200}
                          onClick={(e) => e.stopPropagation()}
                          onChange={(e) => setRenaming({ id: c.id, draft: e.target.value })}
                          onBlur={() => setRenaming(null)}
                          onKeyDown={(e) => {
                            if (e.key === "Enter") {
                              e.preventDefault();
                              void saveTitle(c);
                            } else if (e.key === "Escape") {
                              e.preventDefault();
                              e.stopPropagation();
                              setRenaming(null);
                            }
                          }}
                          className="flex-1 min-w-0 h-7 px-2 -my-1 rounded-[var(--radius-sm)] bg-[var(--color-surface-ai-2)] border border-[var(--color-border-focus)] text-sm text-[var(--color-text-primary)] outline-none"
                        />
                      ) : (
                        <span className="truncate text-[var(--color-text-primary)]">{c.title || "Untitled chat"}</span>
                      )}
                    </span>
                  </td>
                  <td className={`${cell} text-[var(--color-text-secondary)]`}>
                    {c.agent ? (
                      <TooltipPrimitive.Root>
                        <TooltipPrimitive.Trigger asChild>
                          <span className="inline-flex items-center gap-1.5">
                            <ProviderIcon provider={c.agent} />
                            {c.model || "Default"}
                          </span>
                        </TooltipPrimitive.Trigger>
                        <TooltipPrimitive.Portal>
                          <TooltipPrimitive.Content side="bottom" align="start" sideOffset={6} className="surface-ai z-50">
                            <UsageCard chat={c} />
                          </TooltipPrimitive.Content>
                        </TooltipPrimitive.Portal>
                      </TooltipPrimitive.Root>
                    ) : "—"}
                  </td>
                  <td className={cell} onClick={(e) => e.stopPropagation()}>
                    <span className="flex gap-1.5">
                      {(c.worked_issues ?? []).map((id) => <IssueRef key={id} id={id} />)}
                    </span>
                  </td>
                  <td className={`${cell} text-[var(--color-text-muted)]`}>{formatRelativeTime(c.updated_at)}</td>
                  <td className="px-1 py-2" onClick={(e) => e.stopPropagation()}>
                    <RowMenu
                      chat={c}
                      onRename={() => setRenaming({ id: c.id, draft: c.title })}
                      onArchive={() => void archive([c.id])}
                      onUnarchive={() => void unarchive([c.id])}
                      onDelete={() => setConfirmDelete([c])}
                    />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {confirmDelete && (
        <ConfirmDialog
          scope="agent.history.confirm-delete"
          title={confirmDelete.length === 1 ? "Delete this chat?" : `Delete ${confirmDelete.length} chats?`}
          description={confirmDelete.length === 1
            ? `"${confirmDelete[0].title || "Untitled chat"}" will be deleted. This cannot be undone.`
            : `${confirmDelete.length} chats will be deleted. This cannot be undone.`}
          onCancel={() => setConfirmDelete(null)}
        >
          <Button variant="ghost" size="sm" onClick={() => setConfirmDelete(null)}>Cancel</Button>
          <Button
            variant="danger"
            size="sm"
            onClick={() => {
              const ids = confirmDelete.map((c) => c.id);
              setConfirmDelete(null);
              setSelected(new Set());
              void remove(ids);
            }}
          >
            Delete
          </Button>
        </ConfirmDialog>
      )}
    </div>
    </TooltipPrimitive.Provider>
  );
}

/** A chat's turns, tokens and cost, for the hover card on its agent. */
function UsageCard({ chat }: { chat: ChatMeta }) {
  const u = chat.usage;
  const rows: [string, string][] = [
    ["Turns", String(u?.turns ?? 0)],
    ["Tokens in", formatTokens(u?.tokens_in ?? 0)],
    ["Tokens out", formatTokens(u?.tokens_out ?? 0)],
    ["Cost", formatCost(u?.cost_usd ?? 0, !!u?.cost_known)],
  ];
  return (
    <dl className="grid grid-cols-[auto_auto] gap-x-6 gap-y-1 px-3 py-2 text-xs rounded-[var(--radius-md)] border border-[var(--color-border-ai)] bg-[var(--color-surface-ai-1)] shadow-[var(--shadow-popover)]">
      {rows.map(([label, value]) => (
        <div key={label} className="contents">
          <dt className="text-[var(--color-text-muted)]">{label}</dt>
          <dd className="text-right tabular-nums text-[var(--color-text-primary)]">{value}</dd>
        </div>
      ))}
    </dl>
  );
}

interface SelectSlotProps {
  /** The state icon; null leaves the slot blank, as in the header. */
  icon: ReactNode;
  label: string;
  /** Some row is selected: every slot shows its checkbox. */
  selecting: boolean;
  checked: boolean;
  onChange(): void;
}

/**
 * The state icon slot with a checkbox stacked on it. The checkbox takes the
 * icon's place on row hover, on focus, and while any row is selected, so the
 * table needs no checkbox column. Clicks on it never open the row.
 */
function SelectSlot({ icon, label, selecting, checked, onChange }: SelectSlotProps) {
  return (
    <span className="relative shrink-0 flex items-center justify-center w-3.5 h-3.5" onClick={(e) => e.stopPropagation()}>
      {/* The checkbox comes first, as the peer: the icon hides on its keyboard focus only, so a
          clicked (mouse-focused) checkbox does not leave the slot empty after unselecting. */}
      <input
        type="checkbox"
        aria-label={label}
        checked={checked}
        onChange={onChange}
        className={cn("peer", checkbox, !selecting && "opacity-0 group-hover:opacity-100 focus-visible:opacity-100")}
      />
      <span className={cn("flex", selecting ? "invisible" : "group-hover:invisible peer-focus-visible:invisible")}>{icon}</span>
    </span>
  );
}

function Stat({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="px-4 py-3 rounded-[var(--radius-lg)] border border-[var(--color-border-ai-subtle)]" title={hint}>
      <dt className="text-xs text-[var(--color-text-muted)]">{label}</dt>
      <dd className="mt-1 text-lg font-semibold tabular-nums text-[var(--color-text-primary)]">{value}</dd>
    </div>
  );
}

/** A row's "⋯" menu: Rename, Archive or Unarchive, Delete. */
function RowMenu({ chat, onRename, onArchive, onUnarchive, onDelete }: { chat: ChatMeta; onRename(): void; onArchive(): void; onUnarchive(): void; onDelete(): void }) {
  const btnRef = useRef<HTMLButtonElement>(null);
  const [open, setOpen] = useState(false);
  const blocker = archiveBlocker(chat);
  const act = (fn: () => void) => () => {
    setOpen(false);
    fn();
  };

  return (
    <>
      <button
        ref={btnRef}
        type="button"
        aria-label="Chat actions"
        onClick={() => setOpen((o) => !o)}
        className="p-1.5 rounded-[var(--radius-sm)] text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] hover:bg-[var(--color-hover-surface-ai-1)]"
      >
        <MoreHorizontal className="w-4 h-4" />
      </button>
      {open && (
        <Popover anchorRef={btnRef} onClose={() => setOpen(false)}>
          <Menu onClose={() => setOpen(false)} aria-label="Chat actions" className="surface-ai">
            <MenuItem label="Rename" icon={<Pencil size={14} />} onClick={act(onRename)} />
            {chat.archived_at ? (
              <MenuItem label="Unarchive" icon={<ArchiveRestore size={14} />} onClick={act(onUnarchive)} />
            ) : (
              <MenuItem label="Archive" icon={<Archive size={14} />} disabled={!!blocker} onClick={act(onArchive)} />
            )}
            {!chat.archived_at && blocker && <MenuLabel className="pl-8 normal-case tracking-normal font-normal">{blocker}</MenuLabel>}
            <MenuItem label="Delete" icon={<Trash2 size={14} />} destructive onClick={act(onDelete)} />
          </Menu>
        </Popover>
      )}
    </>
  );
}
