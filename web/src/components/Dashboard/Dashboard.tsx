import type { Issue } from '../../api/client';

interface DashboardProps {
  issues: Issue[];
}

export default function Dashboard({ issues }: DashboardProps) {
  // Calculate stats
  const stats = {
    total: issues.length,
    backlog: issues.filter((i) => i.status === 'BACKLOG').length,
    planned: issues.filter((i) => i.status === 'PLANNED').length,
    doing: issues.filter((i) => i.status === 'DOING').length,
    done: issues.filter((i) => i.status === 'DONE').length,
    epics: issues.filter((i) => i.kind === 'EPIC').length,
    tasks: issues.filter((i) => i.kind === 'TASK').length,
    bugs: issues.filter((i) => i.kind === 'BUG').length,
  };

  const totalEstimate = issues.reduce((sum, i) => sum + (i.estimate || 0), 0);
  const totalLogged = issues.reduce((sum, i) => sum + (i.logged_effort || 0), 0);

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900 mb-6">Dashboard</h1>

      {/* Status Overview */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <StatCard label="Total Issues" value={stats.total} color="gray" />
        <StatCard label="Backlog" value={stats.backlog} color="slate" />
        <StatCard label="Planned" value={stats.planned} color="blue" />
        <StatCard label="In Progress" value={stats.doing} color="amber" />
        <StatCard label="Done" value={stats.done} color="green" />
        <StatCard label="Epics" value={stats.epics} color="purple" />
        <StatCard label="Tasks" value={stats.tasks} color="indigo" />
        <StatCard label="Bugs" value={stats.bugs} color="red" />
      </div>

      {/* Effort Overview */}
      <div className="bg-white rounded-lg border border-gray-200 p-6">
        <h2 className="text-lg font-semibold mb-4">Effort Overview</h2>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <span className="text-sm text-gray-500">Total Estimated</span>
            <p className="text-2xl font-bold">{totalEstimate} pts</p>
          </div>
          <div>
            <span className="text-sm text-gray-500">Total Logged</span>
            <p className="text-2xl font-bold">{totalLogged} pts</p>
          </div>
        </div>
        {totalEstimate > 0 && (
          <div className="mt-4">
            <div className="h-4 bg-gray-200 rounded-full overflow-hidden">
              <div
                className="h-full bg-green-500 transition-all"
                style={{ width: `${Math.min(100, (totalLogged / totalEstimate) * 100)}%` }}
              />
            </div>
            <p className="text-sm text-gray-500 mt-1">
              {Math.round((totalLogged / totalEstimate) * 100)}% complete
            </p>
          </div>
        )}
      </div>
    </div>
  );
}

function StatCard({
  label,
  value,
  color,
}: {
  label: string;
  value: number;
  color: string;
}) {
  const colorClasses: Record<string, string> = {
    gray: 'bg-gray-50 border-gray-200',
    slate: 'bg-slate-50 border-slate-200',
    blue: 'bg-blue-50 border-blue-200',
    amber: 'bg-amber-50 border-amber-200',
    green: 'bg-green-50 border-green-200',
    purple: 'bg-purple-50 border-purple-200',
    indigo: 'bg-indigo-50 border-indigo-200',
    red: 'bg-red-50 border-red-200',
  };

  return (
    <div className={`rounded-lg border p-4 ${colorClasses[color] || colorClasses.gray}`}>
      <p className="text-sm text-gray-600">{label}</p>
      <p className="text-2xl font-bold">{value}</p>
    </div>
  );
}
