import { Fragment } from "react";
import type { HistoryEvent } from "../../api/client";
import { LabelBadge } from "../ui";
import { ActivityValue, CycleName, EstimateChip, PriorityChip, StatusChip } from "./activity-chips";
import { IssueRef } from "../IssueRef/IssueRef";
import { resolveAssignee, shortName } from "../../utils/format";

export function describeEvent(evt: HistoryEvent): React.ReactNode | null {
  const p = evt.payload || {};
  switch (evt.type) {
    case "CREATE":
      return "created this issue";
    case "UPDATE": {
      const fragments: React.ReactNode[] = [];
      if (p.status)
        fragments.push(
          <>
            changed status to <StatusChip status={String(p.status)} />
          </>,
        );
      if (p.estimate != null)
        fragments.push(
          Number(p.estimate) === 0 ? (
            <>cleared the estimate</>
          ) : (
            <>
              set estimate to <EstimateChip points={Number(p.estimate)} />
            </>
          ),
        );
      if (p.title) fragments.push(<>updated the title</>);
      if (p.description !== undefined)
        fragments.push(<>updated the description</>);
      if (Array.isArray(p.labels) && p.labels.length === 0)
        fragments.push(<>removed all labels</>);
      else if (p.labels)
        fragments.push(
          <>
            updated labels to{" "}
            {(p.labels as string[]).map((l, i) => (
              <Fragment key={l}>
                {i > 0 && " "}
                <LabelBadge label={l} />
              </Fragment>
            ))}
          </>,
        );
      if (p.assignee === "")
        fragments.push(<>removed the assignee</>);
      else if (p.assignee)
        fragments.push(
          <>
            assigned to{" "}
            <ActivityValue>
              {shortName(resolveAssignee(String(p.assignee), evt.created_by, evt.on_behalf_of))}
            </ActivityValue>
          </>,
        );
      if (p.priority != null)
        fragments.push(
          Number(p.priority) === 0 ? (
            <>cleared the priority</>
          ) : (
            <>
              set priority to <PriorityChip priority={Number(p.priority)} />
            </>
          ),
        );
      if (p.parent_id != null)
        fragments.push(
          p.parent_id === "" ? (
            <>removed the parent issue</>
          ) : (
            <>
              set parent to <IssueRef id={String(p.parent_id)} />
            </>
          ),
        );
      if (p.cycle_id != null)
        fragments.push(
          p.cycle_id === "" ? (
            <>removed this issue from its cycle</>
          ) : (
            <>
              added this issue to <CycleName id={String(p.cycle_id)} />
            </>
          ),
        );
      if (fragments.length === 0) return null;
      return fragments.reduce<React.ReactNode[]>((acc, f, i) => {
        if (i > 0) acc.push(<span key={`sep-${i}`}> and </span>);
        acc.push(<Fragment key={`part-${i}`}>{f}</Fragment>);
        return acc;
      }, []);
    }
    case "DELETE":
      return "deleted this issue";
    case "MERGE": {
      const branch = p.branch ? String(p.branch) : "";
      const shortBranch =
        branch.length > 40 ? branch.slice(0, 40) + "…" : branch;
      return (
        <>
          merged{" "}
          {branch && (
            <span
              className="font-mono text-xs text-[var(--color-accent-primary)]"
              title={branch}
            >
              {shortBranch}
            </span>
          )}
          {p.strategy && <> via {String(p.strategy)}</>}
        </>
      );
    }
    case "ARTIFACT": {
      const action = String(p.action || "updated");
      const filename = String(p.filename || "artifact");
      return (
        <>
          {action}{" "}
          <ActivityValue>
            <span className="font-mono text-xs">{filename}</span>
          </ActivityValue>
        </>
      );
    }
    default:
      return null;
  }
}
