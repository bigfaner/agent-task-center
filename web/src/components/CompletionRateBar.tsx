import { cn } from '@/lib/utils'

interface CompletionRateBarProps {
  rate: number
  className?: string
}

export function CompletionRateBar({ rate, className }: CompletionRateBarProps) {
  const clamped = Math.max(0, Math.min(100, rate))
  return (
    <div className={cn('flex items-center gap-2', className)}>
      <div className="h-2 w-20 overflow-hidden rounded-full bg-gray-200">
        <div
          className="h-full rounded-full bg-primary transition-all"
          style={{ width: `${clamped}%` }}
        />
      </div>
      <span className="text-sm text-muted-foreground">
        {clamped.toFixed(1)}%
      </span>
    </div>
  )
}
