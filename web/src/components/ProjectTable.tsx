import { Link } from 'react-router-dom'
import { CompletionRateBar } from './CompletionRateBar'
import { RelativeTime } from './RelativeTime'
import type { ProjectSummary } from '@/types'

interface ProjectTableProps {
  projects: ProjectSummary[]
}

export function ProjectTable({ projects }: ProjectTableProps) {
  return (
    <div className="overflow-hidden rounded-lg border">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b bg-muted/50">
            <th className="px-4 py-3 text-left font-medium">Project Name</th>
            <th className="px-4 py-3 text-right font-medium">Features</th>
            <th className="px-4 py-3 text-right font-medium">Tasks</th>
            <th className="px-4 py-3 text-left font-medium">Completion</th>
            <th className="px-4 py-3 text-left font-medium">Updated</th>
          </tr>
        </thead>
        <tbody>
          {projects.map((project) => (
            <ProjectTableRow key={project.id} project={project} />
          ))}
        </tbody>
      </table>
    </div>
  )
}

function ProjectTableRow({ project }: { project: ProjectSummary }) {
  return (
    <tr className="border-b last:border-b-0 hover:bg-muted/30 transition-colors">
      <td className="px-4 py-3">
        <Link
          to={`/projects/${project.id}`}
          className="font-medium text-primary hover:underline"
        >
          {project.name}
        </Link>
      </td>
      <td className="px-4 py-3 text-right">{project.featureCount}</td>
      <td className="px-4 py-3 text-right">{project.taskTotal}</td>
      <td className="px-4 py-3">
        <CompletionRateBar rate={project.completionRate} />
      </td>
      <td className="px-4 py-3">
        <RelativeTime date={project.updatedAt} />
      </td>
    </tr>
  )
}
