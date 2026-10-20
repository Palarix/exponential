import type { ReactNode } from "react";
import { useUser } from "../../api/queries";
import { setPreference, usePreferences, useSystemReducedMotion } from "../../preferences";
import { cn } from "../../utils/cn";
import { useAgentChat } from "../Agent/agent-context";
import { ModelPickers } from "../Agent/ComposerPickers";
import { Avatar, Card, Heading, Toggle, TopBar, ViewContainer } from "../ui";
import { DIFF_VIEW_OPTIONS, MOTION_OPTIONS, motionHint, notifyCopy } from "./preferences-utils";

/** The `#/preferences` page: who am I, and settings kept in this browser. */
export default function Preferences() {
  const agentAvailable = !!useAgentChat().status?.available;
  return (
    <ViewContainer topBar={<TopBar left={<Heading title="Preferences" as="h1" />} />}>
      <div className="mx-auto w-full max-w-2xl px-6 py-8 space-y-8">
        <Section title="My Profile">
          <ProfileCard />
        </Section>
        {agentAvailable && (
          <Section title="Agent Chat">
            <DefaultModelCard />
            <NotificationsCard />
          </Section>
        )}
        <Section title="Preferences">
          <MotionCard />
          <DiffViewCard />
        </Section>
      </div>
    </ViewContainer>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-2">
      <h2 className="px-1 text-xs font-medium text-[var(--color-text-muted)]">{title}</h2>
      <div className="space-y-3">{children}</div>
    </section>
  );
}

function ProfileCard() {
  const user = useUser();
  return (
    <Card padding="md">
      {user ? (
        <div className="flex items-center gap-4">
          <Avatar name={`${user.name} <${user.email}>`} size="xl" />
          <div className="min-w-0">
            <div className="text-sm font-medium text-[var(--color-text-primary)] truncate">{user.name}</div>
            <div className="text-sm text-[var(--color-text-muted)] truncate">{user.email}</div>
          </div>
        </div>
      ) : (
        <p className="text-sm text-[var(--color-text-muted)]">Not signed in.</p>
      )}
    </Card>
  );
}

/** A card for one setting: title and description on the left, the control on the right, a hint below. */
function SettingCard({ title, description, hint, children }: { title: string; description: string; hint?: string; children: ReactNode }) {
  return (
    <Card padding="md">
      <div className="flex items-center gap-4">
        <div className="flex-1 min-w-0 space-y-0.5">
          <div className="text-sm font-medium text-[var(--color-text-primary)]">{title}</div>
          <div className="text-sm text-[var(--color-text-muted)]">{description}</div>
        </div>
        <div className="shrink-0 flex items-center">{children}</div>
      </div>
      {hint && (
        <p className="mt-3 pt-3 border-t border-[var(--color-border-subtle)] text-xs text-[var(--color-text-muted)]">{hint}</p>
      )}
    </Card>
  );
}

/** A row of mutually exclusive choices, built from native radios so arrow keys move between them. */
function Segmented<T extends string>({ name, label, options, value, onChange }: {
  name: string;
  label: string;
  options: { value: T; label: string }[];
  value: T;
  onChange(value: T): void;
}) {
  return (
    <div
      role="radiogroup"
      aria-label={label}
      className="flex p-0.5 rounded-[var(--radius-md)] bg-[var(--color-surface)] border border-[var(--color-border-subtle)]"
    >
      {options.map((opt) => {
        const checked = value === opt.value;
        return (
          <label
            key={opt.value}
            className={cn(
              "relative px-3 py-1 text-xs rounded-[var(--radius-sm)] cursor-pointer transition-colors duration-[var(--duration-fast)]",
              "has-[:focus-visible]:outline has-[:focus-visible]:outline-2 has-[:focus-visible]:outline-[var(--color-border-focus)]",
              checked
                ? "bg-[var(--color-surface-3)] text-[var(--color-text-primary)] font-medium"
                : "text-[var(--color-text-secondary)] hover:text-[var(--color-text-primary)]",
            )}
          >
            <input type="radio" name={name} value={opt.value} checked={checked} onChange={() => onChange(opt.value)} className="sr-only" />
            {opt.label}
          </label>
        );
      })}
    </div>
  );
}

function MotionCard() {
  const { motion } = usePreferences();
  const systemReduced = useSystemReducedMotion();
  return (
    <SettingCard title="Motion" description="How much the interface moves." hint={motionHint(motion, systemReduced)}>
      <Segmented name="motion" label="Motion" options={MOTION_OPTIONS} value={motion} onChange={(v) => setPreference("motion", v)} />
    </SettingCard>
  );
}

function DiffViewCard() {
  const { diffView } = usePreferences();
  return (
    <SettingCard title="Diff view" description="How the Changes tab shows diffs.">
      <Segmented name="diff-view" label="Diff view" options={DIFF_VIEW_OPTIONS} value={diffView} onChange={(v) => setPreference("diffView", v)} />
    </SettingCard>
  );
}

function DefaultModelCard() {
  const { status, defaultPrefs, setDefaultPrefs } = useAgentChat();
  return (
    <SettingCard title="Default model" description="New chats start with this model and effort.">
      <ModelPickers status={status} prefs={defaultPrefs} onPrefsChange={setDefaultPrefs} />
    </SettingCard>
  );
}

function NotificationsCard() {
  const { notify, toggleNotify } = useAgentChat();
  const copy = notifyCopy(notify);
  return (
    <SettingCard
      title="Desktop notifications"
      description="Notify me when a chat waits for my input while this tab is in the background."
      hint={copy.switchable ? undefined : copy.hint}
    >
      {copy.switchable ? (
        <Toggle checked={notify === "on"} onChange={toggleNotify} ariaLabel="Desktop notifications" size="md" />
      ) : (
        <span className="text-xs text-[var(--color-text-muted)]">{copy.status}</span>
      )}
    </SettingCard>
  );
}
