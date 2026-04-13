import type { FeatureStatus, TaskStatus } from '@/types'
import { cn } from '@/lib/utils'

type Status = FeatureStatus | TaskStatus

const statusConfig: Record<string, { label: string; className: string }> = {
  prd: { label: 'PRD', className: 'bg-blue-100 text-blue-700' },
  design: { label: 'Design', className: 'bg-purple-100 text-purple-700' },
  tasks: { label: 'Tasks', className: 'bg-yellow-100 text-yellow-700' },
  'in-progress': {
    label: 'In Progress',
    className: 'bg-orange-100 text-orange-700',
  },
  in_progress: {
    label: 'In Progress',
    className: 'bg-orange-100 text-orange-700',
  },
  done: { label: 'Done', className: 'bg-green-100 text-green-700' },
  pending: { label: 'Pending', className: 'bg-gray-100 text-gray-700' },
  completed: {
    label: 'Completed',
    className: 'bg-green-100 text-green-700',
  },
  blocked: { label: 'Blocked', className: 'bg-red-100 text-red-700' },
}

interface StatusBadgeProps {
  status: Status
  className?: string
}

export function StatusBadge({ status, className }: StatusBadgeProps) {
  const config = statusConfig[status] ?? {
    label: status,
    className: 'bg-gray-100 text-gray-700',
  }
  return (
    <span
      className={cn(
        'inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium',
        config.className,
        className,
      )}
    >
      {config.label}
    </span>
  )
}
