import type { Issue } from '../../api/client';

interface DependenciesProps {
  issues: Issue[];
}

export default function Dependencies({ issues }: DependenciesProps) {
  // Find all dependency relationships
  const dependencies: { source: Issue; target: Issue; kind: string }[] = [];

  for (const issue of issues) {
    if (issue.dependencies) {
      for (const dep of issue.dependencies) {
        const target = issues.find((i) => i.id === dep.target_id);
        if (target) {
          dependencies.push({
            source: issue,
            target,
            kind: dep.kind,
          });
        }
      }
    }
  }

  // Group by kind
  const byKind = dependencies.reduce((acc, dep) => {
    if (!acc[dep.kind]) acc[dep.kind] = [];
    acc[dep.kind].push(dep);
    return acc;
  }, {} as Record<string, typeof dependencies>);

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Dependencies</h1>

      {dependencies.length === 0 ? (
        <div className="bg-white rounded-lg border border-gray-200 p-8 text-center text-gray-500">
          No dependencies found
        </div>
      ) : (
        <div className="space-y-6">
          {Object.entries(byKind).map(([kind, deps]) => (
            <div key={kind}>
              <h2 className="text-lg font-semibold text-gray-700 mb-3 capitalize">
                {kind.replace('_', ' ')} ({deps.length})
              </h2>
              <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
                <table className="w-full">
                  <thead className="bg-gray-50">
                    <tr>
                      <th className="text-left px-4 py-3 text-sm font-medium text-gray-600">Source</th>
                      <th className="text-center px-4 py-3 text-sm font-medium text-gray-600">Relationship</th>
                      <th className="text-left px-4 py-3 text-sm font-medium text-gray-600">Target</th>
                    </tr>
                  </thead>
                  <tbody>
                    {deps.map((dep, i) => (
                      <tr key={i} className="border-t border-gray-100">
                        <td className="px-4 py-3">
                          <IssueLink issue={dep.source} />
                        </td>
                        <td className="px-4 py-3 text-center">
                          <RelationshipArrow kind={dep.kind} />
                        </td>
                        <td className="px-4 py-3">
                          <IssueLink issue={dep.target} />
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function IssueLink({ issue }: { issue: Issue }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs font-mono text-gray-400">{issue.id}</span>
      <span className="text-sm">{issue.title}</span>
    </div>
  );
}

function RelationshipArrow({ kind }: { kind: string }) {
  const colors: Record<string, string> = {
    blocked_by: 'text-red-500',
    blocks: 'text-red-500',
    child: 'text-purple-500',
    parent: 'text-purple-500',
  };

  return (
    <span className={`text-sm ${colors[kind] || 'text-gray-500'}`}>
      → {kind.replace('_', ' ')}
    </span>
  );
}
