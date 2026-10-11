import Tooltip from './Tooltip';
import { actorNameDisplay } from './actor-name-utils';

/**
 * A principal's name in an author/actor position. When an agent acted on
 * their behalf (`via`), the name gets a dashed underline and reveals the
 * agent on hover, keyboard focus, and to screen readers.
 */
export default function ActorName({
  name,
  via,
  className,
  focusable,
}: {
  name: string;
  via?: string;
  className?: string;
  focusable?: boolean;
}) {
  const d = actorNameDisplay(name, via, className, { focusable });
  return (
    <Tooltip content={d.tooltip}>
      <span className={d.className} tabIndex={d.focusable ? 0 : undefined}>
        {d.label}
        {d.srText && <span className="sr-only">{d.srText}</span>}
      </span>
    </Tooltip>
  );
}
