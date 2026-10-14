import logo from '../public/favicon.svg?raw';
import type { TabAttention } from './components/Agent/agent-state-utils';

export const FAVICON_URL = '/favicon.svg';

// Favicons can't read CSS variables: these match --color-warning and --color-info.
const DOT_COLOR: Record<Exclude<TabAttention, 'none'>, string> = { approval: '#f2c94c', output: '#4da6e8' };

// A dot in the top-right corner. Its ring takes the logo's contrast color so
// the dot stands apart from the logo in both color schemes.
function dot(color: string): string {
  return `<style>.ring { stroke: #ffffff; } @media (prefers-color-scheme: dark) { .ring { stroke: #000000; } }</style>`
    + `<circle class="ring" cx="264" cy="86" r="73" stroke-width="26" fill="${color}"/>`;
}

const badged = Object.fromEntries(
  Object.entries(DOT_COLOR).map(([k, color]) => [
    k,
    'data:image/svg+xml,' + encodeURIComponent(logo.replace(/<\/svg>\s*$/, `${dot(color)}</svg>\n`)),
  ]),
) as Record<Exclude<TabAttention, 'none'>, string>;

/** The favicon URL for the tab's attention: the logo, plus a colored dot when something needs the user. */
export function faviconFor(attention: TabAttention): string {
  return attention === 'none' ? FAVICON_URL : badged[attention];
}
