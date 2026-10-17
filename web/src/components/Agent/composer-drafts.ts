import { useSyncExternalStore } from "react";
import { addAttachments, isAgentImage } from "./attachment-utils";

/** A file waiting in the composer. preview is an object URL for images. */
export interface PendingAttachment {
  key: string;
  file: File;
  preview?: string;
}

/** What a chat's composer holds before it is sent. */
export interface ComposerDraft {
  text: string;
  attachments: PendingAttachment[];
  /** Set by prefill, until the composer has taken focus for it. */
  prefilled?: true;
}

export const EMPTY_DRAFT: ComposerDraft = { text: "", attachments: [] };

// The new chat's drafts, apart in the popup and the Agent view: a side
// question in the popup leaves the main draft alone. No chat ID has a NUL.
export const NEW_CHAT_POPUP = "\0new-popup";
export const NEW_CHAT_VIEW = "\0new-view";

/**
 * Each unsent draft, by draft key: a chat ID, or NEW_CHAT_POPUP or
 * NEW_CHAT_VIEW for the new chat. Every composer for a chat reads and
 * writes the same draft, so it follows the chat between the popup and the
 * Agent view. Drafts last until a reload. The store owns the previews'
 * object URLs and frees them as attachments go.
 */
export interface ComposerDrafts {
  get(key: string): ComposerDraft;
  setText(key: string, text: string): void;
  /** Adds the files that fit; returns why any did not. */
  addFiles(key: string, files: File[]): string | undefined;
  removeAttachments(key: string, keys: Set<string>): void;
  /** Clears the text and the sent attachments; ones added while sending stay. */
  sent(key: string, keys: Set<string>): void;
  /** Hands from's draft to to, replacing to's, as Maximize does for the new chat. An empty draft moves nothing. */
  move(from: string, to: string): void;
  /** Puts text in the input, such as Start Work's first message. */
  prefill(chatId: string, text: string): void;
  /** Called once the composer has taken focus for a prefill. */
  prefillUsed(chatId: string): void;
  subscribe(listener: () => void): () => void;
}

export function createComposerDrafts(): ComposerDrafts {
  const drafts = new Map<string, ComposerDraft>();
  const listeners = new Set<() => void>();
  const get = (key: string) => drafts.get(key) ?? EMPTY_DRAFT;
  const update = (key: string, change: (d: ComposerDraft) => ComposerDraft) => {
    const next = change(get(key));
    if (next.text === "" && next.attachments.length === 0 && !next.prefilled) drafts.delete(key);
    else drafts.set(key, next);
    listeners.forEach((l) => l());
  };
  const revoke = (attachments: PendingAttachment[]) => attachments.forEach((a) => a.preview && URL.revokeObjectURL(a.preview));
  const removeAttachments = (key: string, keys: Set<string>) => update(key, (d) => {
    revoke(d.attachments.filter((a) => keys.has(a.key)));
    return { ...d, attachments: d.attachments.filter((a) => !keys.has(a.key)) };
  });

  return {
    get,
    setText: (key, text) => update(key, (d) => ({ ...d, text })),
    addFiles: (key, files) => {
      if (files.length === 0) return undefined;
      const { added, error } = addAttachments(get(key).attachments.map((a) => a.file), files);
      const items = added.map((file) => ({
        key: crypto.randomUUID(),
        file,
        preview: isAgentImage(file.type) ? URL.createObjectURL(file) : undefined,
      }));
      if (items.length > 0) update(key, (d) => ({ ...d, attachments: [...d.attachments, ...items] }));
      return error;
    },
    removeAttachments,
    sent: (key, keys) => {
      removeAttachments(key, keys);
      update(key, (d) => ({ ...d, text: "" }));
    },
    move: (from, to) => {
      const moved = drafts.get(from);
      if (!moved) return;
      drafts.delete(from);
      update(to, (d) => {
        revoke(d.attachments);
        return moved;
      });
    },
    prefill: (chatId, text) => update(chatId, (d) => ({ ...d, text, prefilled: true })),
    prefillUsed: (chatId) => update(chatId, (d) => ({ text: d.text, attachments: d.attachments })),
    subscribe: (listener) => {
      listeners.add(listener);
      return () => listeners.delete(listener);
    },
  };
}

/** The draft for key; re-renders the caller only when it changes. */
export function useComposerDraft(drafts: ComposerDrafts, key: string): ComposerDraft {
  const get = () => drafts.get(key);
  return useSyncExternalStore(drafts.subscribe, get, get);
}
