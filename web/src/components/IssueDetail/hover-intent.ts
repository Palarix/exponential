/**
 * Calls onChange(true) once the pointer has rested for `delay` ms, and
 * onChange(false) as soon as it leaves.
 */
export function createHoverIntent(delay: number, onChange: (open: boolean) => void) {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const cancel = () => {
    clearTimeout(timer);
    timer = undefined;
  };
  return {
    enter() {
      cancel();
      timer = setTimeout(() => {
        timer = undefined;
        onChange(true);
      }, delay);
    },
    leave() {
      cancel();
      onChange(false);
    },
    cancel,
  };
}
