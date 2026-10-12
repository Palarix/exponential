import { useState, useMemo } from "react";
import { filterPeople } from "../../utils/issues";
import Avatar from "./Avatar";
import { Menu, MenuItem, MenuFilter, MenuLabel } from "./Menu";

interface AssigneePickerProps {
  people: string[];
  current?: string;
  /** Called with the chosen person, or null for "No assignee". Not called when the choice is unchanged. */
  onSelect: (person: string | null) => void;
  onClose?: () => void;
  /** Accessible name for the menu. */
  ariaLabel?: string;
}

export default function AssigneePicker({ people, current, onSelect, onClose, ariaLabel = "Change assignee" }: AssigneePickerProps) {
  const [filterText, setFilterText] = useState("");
  const visible = useMemo(() => filterPeople(people, filterText), [people, filterText]);
  const hasQuery = filterText.trim().length > 0;

  const select = (person: string | null) => {
    if ((person ?? "") !== (current ?? "")) onSelect(person);
    onClose?.();
  };

  return (
    <Menu onClose={onClose} aria-label={ariaLabel} bare autoFocus={false} maxHeight="18rem">
      <MenuFilter value={filterText} onChange={setFilterText} placeholder="Set assignee..." />
      {!hasQuery && (
        <MenuItem
          label="No assignee"
          icon={<Avatar size="sm" />}
          radio
          checked={!current}
          onClick={() => select(null)}
        />
      )}
      {visible.map(person => (
        <MenuItem
          key={person}
          label={person.split(" <")[0]}
          icon={<Avatar name={person} size="sm" />}
          radio
          checked={person === current}
          truncate
          onClick={() => select(person)}
        />
      ))}
      {visible.length === 0 && <MenuLabel>No matching people</MenuLabel>}
    </Menu>
  );
}
