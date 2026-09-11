import { describe, expect, it } from 'vitest';
import { computePopoverPosition, type Placement } from './popover-utils';

const viewport = { width: 1024, height: 768 };
const anchor = { top: 100, bottom: 130, left: 200, right: 280, width: 80, height: 30 };
const popover = { width: 200, height: 150 };
const offset = 4;

function pos(placement: Placement, a = anchor, p = popover, o = offset, v = viewport) {
  return computePopoverPosition(a, p, placement, o, v);
}

describe('computePopoverPosition', () => {
  describe('placement', () => {
    it('bottom-start: below anchor, left-aligned', () => {
      const { top, left } = pos('bottom-start');
      expect(top).toBe(134); // anchor.bottom + offset
      expect(left).toBe(200); // anchor.left
    });

    it('bottom-center: below anchor, centered', () => {
      const { top, left } = pos('bottom-center');
      expect(top).toBe(134);
      expect(left).toBe(140); // 200 + 80/2 - 200/2 = 140
    });

    it('bottom-end: below anchor, right-aligned', () => {
      const { top, left } = pos('bottom-end');
      expect(top).toBe(134);
      expect(left).toBe(80); // 280 - 200 = 80
    });
  });

  describe('flip above', () => {
    it('flips above when overflowing bottom', () => {
      const lowAnchor = { ...anchor, top: 600, bottom: 630 };
      const { top } = pos('bottom-start', lowAnchor);
      expect(top).toBe(446); // 600 - 150 - 4
    });
  });

  describe('clamp edges', () => {
    it('clamps left edge', () => {
      const leftAnchor = { ...anchor, left: 2, right: 82 };
      const { left } = pos('bottom-start', leftAnchor);
      expect(left).toBe(8); // PAD
    });

    it('clamps right edge', () => {
      const rightAnchor = { ...anchor, left: 900, right: 980 };
      const { left } = pos('bottom-start', rightAnchor);
      expect(left).toBe(816); // 1024 - 200 - 8
    });

    it('clamps top edge when flipped above and still overflows', () => {
      const smallViewport = { width: 1024, height: 200 };
      const midAnchor = { ...anchor, top: 80, bottom: 110 };
      const { top } = pos('bottom-start', midAnchor, popover, offset, smallViewport);
      expect(top).toBe(8); // PAD
    });

    it('clamps bottom when popover larger than available space', () => {
      const tinyViewport = { width: 1024, height: 100 };
      const topAnchor = { ...anchor, top: 10, bottom: 40 };
      const bigPopover = { width: 200, height: 200 };
      const { top } = pos('bottom-start', topAnchor, bigPopover, offset, tinyViewport);
      expect(top).toBe(tinyViewport.height - 200 - 8); // -108 clamped
    });
  });

  describe('combined', () => {
    it('flip + right clamp', () => {
      const cornerAnchor = { top: 700, bottom: 730, left: 950, right: 1010, width: 60, height: 30 };
      const { top, left } = pos('bottom-start', cornerAnchor);
      expect(top).toBe(546); // 700 - 150 - 4
      expect(left).toBe(816); // 1024 - 200 - 8
    });

    it('bottom-end with left clamp', () => {
      const leftAnchor = { top: 100, bottom: 130, left: 10, right: 50, width: 40, height: 30 };
      const { left } = pos('bottom-end', leftAnchor);
      expect(left).toBe(8); // 50 - 200 = -150, clamped to PAD
    });
  });
});
