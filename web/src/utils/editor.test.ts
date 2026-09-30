import { describe, it, expect } from 'vitest';
import { vscodeUrl } from './editor';

describe('vscodeUrl', () => {
  it('builds a vscode:// file URL from a POSIX path', () => {
    expect(vscodeUrl('/Users/me/proj/.xpo/worktrees/xpo-1-foo')).toBe(
      'vscode://file/Users/me/proj/.xpo/worktrees/xpo-1-foo',
    );
  });

  it('percent-encodes spaces and special characters', () => {
    expect(vscodeUrl('/Volumes/My Work/proj#1')).toBe('vscode://file/Volumes/My%20Work/proj%231');
  });

  it('normalizes Windows paths', () => {
    expect(vscodeUrl('C:\\Users\\me\\proj')).toBe('vscode://file/C:/Users/me/proj');
  });
});
