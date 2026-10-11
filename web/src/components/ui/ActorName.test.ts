import { describe, expect, it } from 'vitest';
import { actorNameDisplay } from './actor-name-utils';

describe('actorNameDisplay', () => {
  describe('direct action (no via)', () => {
    const d = actorNameDisplay('Alice <alice@x.com>', undefined, 'font-medium');

    it('shows the short name', () => {
      expect(d.label).toBe('Alice');
    });

    it('has no tooltip, no sr-only text and is not focusable', () => {
      expect(d.tooltip).toBe('');
      expect(d.srText).toBe('');
      expect(d.focusable).toBe(false);
    });

    it('has no dashed underline but keeps the caller className', () => {
      expect(d.className).not.toContain('decoration-dashed');
      expect(d.className).not.toContain('underline');
      expect(d.className).toContain('font-medium');
    });
  });

  describe('agent acting on behalf of the principal', () => {
    const d = actorNameDisplay('Nicolas Bettenburg <nic@x.com>', 'claude-code/2.1.294 <agent@mcp>', 'font-medium');

    it('shows the principal short name', () => {
      expect(d.label).toBe('Nicolas Bettenburg');
    });

    it('renders a dashed underline and keeps the caller className', () => {
      expect(d.className).toContain('underline');
      expect(d.className).toContain('decoration-dashed');
      expect(d.className).toContain('font-medium');
    });

    it('reveals the agent in the tooltip', () => {
      expect(d.tooltip).toBe('Claude Code for Nicolas Bettenburg');
    });

    it('exposes the agent to assistive tech and is keyboard focusable', () => {
      expect(d.srText).toBe(' (via Claude Code)');
      expect(d.focusable).toBe(true);
    });
  });

  it('accepts an already-formatted agent display name and principal', () => {
    const d = actorNameDisplay('Alice', 'Claude Code');
    expect(d.tooltip).toBe('Claude Code for Alice');
    expect(d.srText).toBe(' (via Claude Code)');
  });

  it('treats an empty via as a direct action', () => {
    const d = actorNameDisplay('Alice', '');
    expect(d.tooltip).toBe('');
    expect(d.className).not.toContain('decoration-dashed');
  });

  it('can opt out of focus when nested in an interactive control', () => {
    const d = actorNameDisplay('Alice', 'Claude Code', undefined, { focusable: false });
    expect(d.focusable).toBe(false);
    expect(d.className).toContain('decoration-dashed');
    expect(d.srText).toBe(' (via Claude Code)');
  });
});
