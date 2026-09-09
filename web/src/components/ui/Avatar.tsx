import { useState } from "react";
import { md5 } from "../../utils/md5";

type AvatarSize = "xs" | "sm" | "md" | "lg";

const SIZE_PX: Record<AvatarSize, number> = {
  xs: 14,
  sm: 16,
  md: 20,
  lg: 24,
};

const failedEmails = new Set<string>();
const loadedHashes = new Set<string>();

interface AvatarProps {
  name: string;
  size?: AvatarSize;
}

export default function Avatar({ name, size = "md" }: AvatarProps) {
  const px = SIZE_PX[size];
  const email = name.match(/<([^>]+)>/)?.[1]?.toLowerCase().trim() || "";
  const hash = email ? md5(email) : "";
  const [failed, setFailed] = useState(!email || failedEmails.has(email));

  if (!failed) {
    const src = `https://www.gravatar.com/avatar/${hash}?s=${px * 2}&d=404`;
    return (
      <img
        src={src}
        width={px}
        height={px}
        className="rounded-full shrink-0"
        alt=""
        title={name}
        loading="lazy"
        onLoad={() => loadedHashes.add(hash)}
        onError={() => {
          failedEmails.add(email);
          setFailed(true);
        }}
      />
    );
  }

  return (
    <div
      className="rounded-full bg-zinc-700 flex items-center justify-center font-medium text-[var(--color-text-secondary)] shrink-0"
      style={{ width: px, height: px, fontSize: px * 0.5 }}
      title={name}
    >
      {name.charAt(0).toUpperCase()}
    </div>
  );
}


