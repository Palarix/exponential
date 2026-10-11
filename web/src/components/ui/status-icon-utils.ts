import { cn } from '../../utils/cn';

// Icons must keep their intrinsic size; without shrink-0 a flex row with a
// long truncating sibling squeezes the SVG below its width.
export function statusIconClass(className?: string): string {
  return cn('shrink-0', className);
}
