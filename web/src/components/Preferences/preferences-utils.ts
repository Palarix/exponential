import type { DiffView, MotionPreference } from "../../preferences";
import type { NotifyState } from "../Agent/agent-state-utils";

export const MOTION_OPTIONS: { value: MotionPreference; label: string }[] = [
  { value: "system", label: "System" },
  { value: "on", label: "On" },
  { value: "off", label: "Off" },
];

export const DIFF_VIEW_OPTIONS: { value: DiffView; label: string }[] = [
  { value: "unified", label: "Unified" },
  { value: "split", label: "Split" },
];

/** The line under the Motion control: what the current choice does. */
export function motionHint(pref: MotionPreference, systemReduced: boolean): string {
  switch (pref) {
    case "on": return "Animations always play, even if your system asks for reduced motion.";
    case "off": return "No movement or fades. Loading and progress indicators still animate.";
    case "system": return systemReduced
      ? "Your system asks for reduced motion, so movement becomes a fade."
      : "Your system allows motion, so animations play.";
  }
}

/** The notifications card: a switch, or why there is none. */
export function notifyCopy(state: NotifyState): { switchable: true } | { switchable: false; status: string; hint?: string } {
  switch (state) {
    case "on":
    case "off": return { switchable: true };
    case "blocked": return { switchable: false, status: "Blocked by the browser", hint: "Allow notifications for this site in your browser's settings." };
    case "unsupported": return { switchable: false, status: "Not supported" };
  }
}
