import type { TaskPriority } from '@/types'
import { cn } from '@/lib/utils'

const priorityConfig: Record<
  TaskPriority,
  { label: string; className: string }
> = {
  P0: { label: 'P0', className: 'bg-red-100 text-red-700' },
  P1: { label: 'P1', className: 'bg-orange-100 text-orange-700' },
  P2: { label: 'P2', className: 'bg-blue-100 text-blue-700' },
}

interface PriorityBadgeProps {
  priority: TaskPriority
  className?: string
}

export function PriorityBadge({ priority, className }: PriorityBadgeProps) {
  const config = priorityConfig[priority]
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
