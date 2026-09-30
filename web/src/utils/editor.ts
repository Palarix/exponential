// Builds a vscode://file URL that opens the given absolute path in VS Code.
// Each path segment is percent-encoded (so '#', '?', spaces survive), except
// a Windows drive letter segment like "C:".
export function vscodeUrl(path: string): string {
  let p = path.replace(/\\/g, '/');
  if (!p.startsWith('/')) p = `/${p}`;
  const encoded = p
    .split('/')
    .map((seg) => (/^[A-Za-z]:$/.test(seg) ? seg : encodeURIComponent(seg)))
    .join('/');
  return `vscode://file${encoded}`;
}
