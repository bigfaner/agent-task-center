import { cn } from '@/lib/utils'

interface TableSkeletonProps {
  rows?: number
  className?: string
}

export function TableSkeleton({ rows = 5, className }: TableSkeletonProps) {
  return (
    <div className={cn('space-y-3', className)}>
      {Array.from({ length: rows }).map((_, i) => (
        <div key={i} className="flex items-center gap-4">
          <div className="h-4 w-1/4 animate-pulse rounded bg-muted" />
          <div className="h-4 w-1/6 animate-pulse rounded bg-muted" />
          <div className="h-4 w-1/6 animate-pulse rounded bg-muted" />
          <div className="h-4 w-1/5 animate-pulse rounded bg-muted" />
          <div className="h-4 w-1/6 animate-pulse rounded bg-muted" />
        </div>
      ))}
    </div>
  )
}
