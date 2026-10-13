import { Paperclip } from "lucide-react";
import { indicatorPill } from "./indicator-pill";

/** A row indicator with the issue's attachment count. */
export default function AttachmentIndicator({ count }: { count: number }) {
  return (
    <span
      className={indicatorPill}
      title={`${count} ${count === 1 ? "attachment" : "attachments"}`}
    >
      <Paperclip size={12} strokeWidth={1.75} />
      {count}
    </span>
  );
}
