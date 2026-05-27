import { useState } from "react";

type AvatarSize = "xs" | "sm" | "md" | "lg";

const SIZE_PX: Record<AvatarSize, number> = {
  xs: 14,
  sm: 16,
  md: 20,
  lg: 24,
};

interface AvatarProps {
  name: string;
  size?: AvatarSize;
}

export default function Avatar({ name, size = "md" }: AvatarProps) {
  const px = SIZE_PX[size];
  const email = name.match(/<([^>]+)>/)?.[1]?.toLowerCase().trim() || "";
  const [failed, setFailed] = useState(!email);

  if (!failed) {
    return (
      <img
        src={`https://www.gravatar.com/avatar/${md5(email)}?s=${px * 2}&d=404`}
        width={px}
        height={px}
        className="rounded-full shrink-0"
        alt=""
        title={name}
        onError={() => setFailed(true)}
      />
    );
  }

  return (
    <div
      className="rounded-full bg-white flex items-center justify-center font-medium text-[var(--color-text-muted)] shrink-0 -m-0.5"
      style={{ width: px, height: px, fontSize: px * 0.5 }}
      title={name}
    >
      {name.charAt(0).toUpperCase()}
    </div>
  );
}

function md5(input: string): string {
  let hash = 0x67452301;
  let a = 0xefcdab89;
  let b = 0x98badcfe;
  let c = 0x10325476;
  const bytes: number[] = [];
  for (let i = 0; i < input.length; i++) {
    bytes.push(input.charCodeAt(i) & 0xff);
  }
  bytes.push(0x80);
  while (bytes.length % 64 !== 56) bytes.push(0);
  const bitLen = input.length * 8;
  bytes.push(
    bitLen & 0xff,
    (bitLen >> 8) & 0xff,
    (bitLen >> 16) & 0xff,
    (bitLen >> 24) & 0xff,
    0, 0, 0, 0,
  );

  const rl = (v: number, s: number) => (v << s) | (v >>> (32 - s));

  const K = [
    0xd76aa478, 0xe8c7b756, 0x242070db, 0xc1bdceee, 0xf57c0faf, 0x4787c62a,
    0xa8304613, 0xfd469501, 0x698098d8, 0x8b44f7af, 0xffff5bb1, 0x895cd7be,
    0x6b901122, 0xfd987193, 0xa679438e, 0x49b40821, 0xf61e2562, 0xc040b340,
    0x265e5a51, 0xe9b6c7aa, 0xd62f105d, 0x02441453, 0xd8a1e681, 0xe7d3fbc8,
    0x21e1cde6, 0xc33707d6, 0xf4d50d87, 0x455a14ed, 0xa9e3e905, 0xfcefa3f8,
    0x676f02d9, 0x8d2a4c8a, 0xfffa3942, 0x8771f681, 0x6d9d6122, 0xfde5380c,
    0xa4beea44, 0x4bdecfa9, 0xf6bb4b60, 0xbebfbc70, 0x289b7ec6, 0xeaa127fa,
    0xd4ef3085, 0x04881d05, 0xd9d4d039, 0xe6db99e5, 0x1fa27cf8, 0xc4ac5665,
    0xf4292244, 0x432aff97, 0xab9423a7, 0xfc93a039, 0x655b59c3, 0x8f0ccc92,
    0xffeff47d, 0x85845dd1, 0x6fa87e4f, 0xfe2ce6e0, 0xa3014314, 0x4e0811a1,
    0xf7537e82, 0xbd3af235, 0x2ad7d2bb, 0xeb86d391,
  ];
  const S = [
    7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22, 7, 12, 17, 22, 5, 9, 14, 20,
    5, 9, 14, 20, 5, 9, 14, 20, 5, 9, 14, 20, 4, 11, 16, 23, 4, 11, 16, 23,
    4, 11, 16, 23, 4, 11, 16, 23, 6, 10, 15, 21, 6, 10, 15, 21, 6, 10, 15, 21,
    6, 10, 15, 21,
  ];

  for (let off = 0; off < bytes.length; off += 64) {
    const M: number[] = [];
    for (let j = 0; j < 16; j++) {
      M[j] =
        bytes[off + j * 4] |
        (bytes[off + j * 4 + 1] << 8) |
        (bytes[off + j * 4 + 2] << 16) |
        (bytes[off + j * 4 + 3] << 24);
    }
    let aa = hash,
      bb = a,
      cc = b,
      dd = c;
    for (let i = 0; i < 64; i++) {
      let f: number, g: number;
      if (i < 16) {
        f = (bb & cc) | (~bb & dd);
        g = i;
      } else if (i < 32) {
        f = (dd & bb) | (~dd & cc);
        g = (5 * i + 1) % 16;
      } else if (i < 48) {
        f = bb ^ cc ^ dd;
        g = (3 * i + 5) % 16;
      } else {
        f = cc ^ (bb | ~dd);
        g = (7 * i) % 16;
      }
      const tmp = dd;
      dd = cc;
      cc = bb;
      bb = (bb + rl((aa + f + K[i] + M[g]) | 0, S[i])) | 0;
      aa = tmp;
    }
    hash = (hash + aa) | 0;
    a = (a + bb) | 0;
    b = (b + cc) | 0;
    c = (c + dd) | 0;
  }

  const hex = (v: number) => {
    let s = "";
    for (let i = 0; i < 4; i++)
      s += ((v >> (i * 8)) & 0xff).toString(16).padStart(2, "0");
    return s;
  };
  return hex(hash) + hex(a) + hex(b) + hex(c);
}
