import type { SendAttachment } from "../../api/chat";

// The server's limits (internal/chat/attachment.go). Images are capped at
// Anthropic's per-image limit.
export const MAX_ATTACHMENTS = 10;
export const MAX_IMAGE_BYTES = 5 * 1024 * 1024;
export const MAX_FILE_BYTES = 10 * 1024 * 1024;

const AGENT_IMAGES = new Set(["image/png", "image/jpeg", "image/gif", "image/webp"]);

/** The agents take this type as image input; other files go in as paths. */
export function isAgentImage(mime: string): boolean {
  return AGENT_IMAGES.has(mime);
}

interface FileLike {
  name: string;
  type: string;
  size: number;
}

/**
 * The files that fit next to the pending ones, and why any did not: the
 * count limit, an empty file, or a size limit.
 */
export function addAttachments<T extends FileLike>(pending: T[], files: T[]): { added: T[]; error?: string } {
  const added: T[] = [];
  let error: string | undefined;
  for (const f of files) {
    if (pending.length + added.length >= MAX_ATTACHMENTS) {
      error = `At most ${MAX_ATTACHMENTS} attachments per message`;
      break;
    }
    const limit = isAgentImage(f.type) ? MAX_IMAGE_BYTES : MAX_FILE_BYTES;
    if (f.size === 0) error = `${f.name} is empty`;
    else if (f.size > limit) error = `${f.name} is over ${limit / 1024 / 1024} MB, the limit for ${isAgentImage(f.type) ? "images" : "files"}`;
    else added.push(f);
  }
  return error ? { added, error } : { added };
}

/** A file as the send request carries it: its content in base64. */
export async function toSendAttachment(file: File): Promise<SendAttachment> {
  const bytes = new Uint8Array(await file.arrayBuffer());
  let binary = "";
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  const ext = file.type.startsWith("image/") ? `.${file.type.slice(6).replace(/\+.*/, "")}` : "";
  return { name: file.name || `pasted${ext}`, mime: file.type, data: btoa(binary) };
}

export function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${Math.round(bytes / 1024)} KB`;
  return `${Math.round((bytes / 1024 / 1024) * 10) / 10} MB`;
}

/** Where the board serves a sent attachment. */
export function attachmentUrl(chatId: string, attachmentId: string): string {
  return `/api/local/chats/${encodeURIComponent(chatId)}/attachments/${encodeURIComponent(attachmentId)}`;
}
