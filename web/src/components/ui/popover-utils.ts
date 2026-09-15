export type Placement = 'bottom-start' | 'bottom-center' | 'bottom-end' | 'right-start';

interface Rect {
  top: number;
  bottom: number;
  left: number;
  right: number;
  width: number;
  height: number;
}

interface Dims {
  width: number;
  height: number;
}

interface Viewport {
  width: number;
  height: number;
}

const PAD = 8;

export function computePopoverPosition(
  anchor: Rect,
  popover: Dims,
  placement: Placement,
  offset: number,
  viewport: Viewport,
): { top: number; left: number } {
  let top: number;
  let left: number;

  if (placement === 'right-start') {
    top = anchor.top;
    left = anchor.right + offset;
    if (left + popover.width > viewport.width - PAD) {
      left = anchor.left - popover.width - offset;
    }
  } else {
    top = anchor.bottom + offset;
    switch (placement) {
      case 'bottom-start':
        left = anchor.left;
        break;
      case 'bottom-center':
        left = anchor.left + anchor.width / 2 - popover.width / 2;
        break;
      case 'bottom-end':
        left = anchor.right - popover.width;
        break;
    }
    if (top + popover.height > viewport.height - PAD) {
      top = anchor.top - popover.height - offset;
    }
  }

  if (left < PAD) left = PAD;
  if (left + popover.width > viewport.width - PAD) left = viewport.width - popover.width - PAD;
  if (top < PAD) top = PAD;
  if (top + popover.height > viewport.height - PAD) top = viewport.height - popover.height - PAD;

  return { top, left };
}
