import type { Issue } from '../../api/client';

interface BoardProps {
  issues: Issue[];
  onRefresh: () => void;
}

const COLUMNS = [
  { id: 'PLANNED', label: 'Planned', color: 'blue' },
  { id: 'DOING', label: 'In Progress', color: 'amber' },
  { id: 'BLOCKED', label: 'Blocked', color: 'red' },
  { id: 'DONE', label: 'Done', color: 'green' },
];

export default function Board({ issues }: BoardProps) {
  // Filter out BACKLOG by default and group by status
  const boardIssues = issues.filter((i) => i.status !== 'BACKLOG');

  return (
    <div>
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Board</h1>
        <div className="text-sm text-gray-500">
          {boardIssues.length} issue{boardIssues.length !== 1 ? 's' : ''} on board
        </div>
      </div>

      <div className="grid grid-cols-4 gap-4">
        {COLUMNS.map((column) => (
          <Column
            key={column.id}
            id={column.id}
            label={column.label}
            color={column.color}
            issues={boardIssues.filter((i) => i.status === column.id)}
          />
        ))}
      </div>
    </div>
  );
}

function Column({
  id: _id,
  label,
  color,
  issues,
}: {
  id: string;
  label: string;
  color: string;
  issues: Issue[];
}) {
  const headerColors: Record<string, string> = {
    blue: 'bg-blue-500',
    amber: 'bg-amber-500',
    red: 'bg-red-500',
    green: 'bg-green-500',
  };

  return (
    <div className="bg-gray-100 rounded-lg overflow-hidden">
      {/* Column Header */}
      <div className={`${headerColors[color]} px-3 py-2 flex items-center justify-between`}>
        <span className="text-white font-medium text-sm">{label}</span>
        <span className="text-white/80 text-sm">{issues.length}</span>
      </div>

      {/* Cards */}
      <div className="p-2 space-y-2 min-h-[200px]">
        {issues.map((issue) => (
          <IssueCard key={issue.id} issue={issue} />
        ))}
        {issues.length === 0 && (
          <div className="text-center text-gray-400 text-sm py-8">
            No issues
          </div>
        )}
      </div>
    </div>
  );
}

function IssueCard({ issue }: { issue: Issue }) {
  const kindBadges: Record<string, string> = {
    EPIC: 'bg-purple-100 text-purple-700',
    TASK: 'bg-blue-100 text-blue-700',
    BUG: 'bg-red-100 text-red-700',
  };

  // Calculate checklist progress
  const checklistTotal = issue.checklist?.length || 0;
  const checklistDone = issue.checklist?.filter((c) => c.state === 'done').length || 0;

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 p-3 cursor-pointer hover:shadow-md transition-shadow">
      {/* Kind Badge */}
      <div className="flex items-center gap-2 mb-2">
        <span className={`text-xs font-medium px-1.5 py-0.5 rounded ${kindBadges[issue.kind] || 'bg-gray-100'}`}>
          {issue.kind}
        </span>
        <span className="text-xs text-gray-400">{issue.id}</span>
      </div>

      {/* Title */}
      <p className="text-sm font-medium text-gray-900 mb-2">{issue.title}</p>

      {/* Footer */}
      <div className="flex items-center justify-between text-xs text-gray-500">
        {issue.estimate !== undefined && issue.estimate > 0 && (
          <span>{issue.estimate} pts</span>
        )}
        {checklistTotal > 0 && (
          <span>{checklistDone}/{checklistTotal} ✓</span>
        )}
      </div>

      {/* Checklist Progress Bar */}
      {checklistTotal > 0 && (
        <div className="mt-2 h-1 bg-gray-200 rounded-full overflow-hidden">
          <div
            className="h-full bg-green-500"
            style={{ width: `${(checklistDone / checklistTotal) * 100}%` }}
          />
        </div>
      )}
    </div>
  );
}
