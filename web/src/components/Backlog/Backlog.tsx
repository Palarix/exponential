import type { Issue } from '../../api/client';

interface BacklogProps {
  issues: Issue[];
  onRefresh: () => void;
}

export default function Backlog({ issues }: BacklogProps) {
  const backlogIssues = issues.filter((i) => i.status === 'BACKLOG');
  const activeIssues = issues.filter((i) => ['PLANNED', 'DOING', 'BLOCKED'].includes(i.status));
  const doneIssues = issues.filter((i) => i.status === 'DONE');

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Backlog</h1>

      {/* Active Table */}
      <IssueTable title={`Active (${activeIssues.length})`} issues={activeIssues} />

      {/* Backlog Table */}
      <IssueTable title={`Backlog (${backlogIssues.length})`} issues={backlogIssues} />

      {/* Done Table (Collapsed) */}
      <details className="mt-6">
        <summary className="text-lg font-semibold text-gray-700 cursor-pointer hover:text-gray-900">
          Done ({doneIssues.length})
        </summary>
        <div className="mt-2">
          <IssueTable title="" issues={doneIssues} />
        </div>
      </details>
    </div>
  );
}

function IssueTable({ title, issues }: { title: string; issues: Issue[] }) {
  if (issues.length === 0 && !title) return null;

  return (
    <div className="mb-6">
      {title && <h2 className="text-lg font-semibold text-gray-700 mb-3">{title}</h2>}
      <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
        <table className="w-full">
          <thead className="bg-gray-50">
            <tr>
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-600">ID</th>
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-600">Type</th>
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-600">Title</th>
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-600">Status</th>
              <th className="text-left px-4 py-3 text-sm font-medium text-gray-600">Estimate</th>
            </tr>
          </thead>
          <tbody>
            {issues.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-gray-500">
                  No issues
                </td>
              </tr>
            ) : (
              issues.map((issue) => (
                <tr key={issue.id} className="border-t border-gray-100 hover:bg-gray-50">
                  <td className="px-4 py-3 text-sm font-mono text-gray-500">{issue.id}</td>
                  <td className="px-4 py-3">
                    <KindBadge kind={issue.kind} />
                  </td>
                  <td className="px-4 py-3 text-sm">{issue.title}</td>
                  <td className="px-4 py-3">
                    <StatusBadge status={issue.status} />
                  </td>
                  <td className="px-4 py-3 text-sm text-gray-600">
                    {issue.estimate ? `${issue.estimate} pts` : '-'}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function KindBadge({ kind }: { kind: string }) {
  const styles: Record<string, string> = {
    EPIC: 'bg-purple-100 text-purple-700',
    TASK: 'bg-blue-100 text-blue-700',
    BUG: 'bg-red-100 text-red-700',
  };

  return (
    <span className={`text-xs font-medium px-2 py-1 rounded ${styles[kind] || 'bg-gray-100 text-gray-700'}`}>
      {kind}
    </span>
  );
}

function StatusBadge({ status }: { status: string }) {
  const styles: Record<string, string> = {
    BACKLOG: 'bg-slate-100 text-slate-700',
    PLANNED: 'bg-blue-100 text-blue-700',
    DOING: 'bg-amber-100 text-amber-700',
    BLOCKED: 'bg-red-100 text-red-700',
    DONE: 'bg-green-100 text-green-700',
  };

  return (
    <span className={`text-xs font-medium px-2 py-1 rounded ${styles[status] || 'bg-gray-100 text-gray-700'}`}>
      {status}
    </span>
  );
}
