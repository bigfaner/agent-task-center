import { useMemo, useCallback } from 'react'
import { useParams, Link, useSearchParams } from 'react-router-dom'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { AppHeader } from '@/components/AppHeader'
import { PriorityBadge } from '@/components/PriorityBadge'
import { ErrorState } from '@/components/ErrorState'
import { getFeatureTasks } from '@/api/features'
import type { TaskSummary, TaskStatus, TaskFilter } from '@/types'

// ── Constants ──

const COLUMNS: { status: TaskStatus; label: string }[] = [
  { status: 'pending', label: 'Pending' },
  { status: 'in_progress', label: 'In Progress' },
  { status: 'completed', label: 'Completed' },
  { status: 'blocked', label: 'Blocked' },
]

const PRIORITY_OPTIONS = ['P0', 'P1', 'P2'] as const
const STATUS_OPTIONS: TaskStatus[] = [
  'pending',
  'in_progress',
  'completed',
  'blocked',
]

// ── Helper: parse CSV from URL param ──

function parseCsv(value: string | null): string[] {
  if (!value) return []
  return value.split(',').map((s) => s.trim()).filter(Boolean)
}

// ── Kanban Skeleton ──

function KanbanSkeleton() {
  return (
    <div className="mt-4 grid grid-cols-4 gap-4 overflow-x-auto">
      {COLUMNS.map((col) => (
        <div key={col.status} className="min-w-[200px]">
          <div className="mb-2 h-6 w-24 animate-pulse rounded bg-muted" />
          {Array.from({ length: 2 }).map((_, i) => (
            <div
              key={i}
              data-testid="kanban-skeleton"
              className="mb-2 h-24 animate-pulse rounded-lg bg-muted"
            />
          ))}
        </div>
      ))}
    </div>
  )
}

// ── TaskCard ──

function TaskCard({ task }: { task: TaskSummary }) {
  return (
    <Link
      to={`/tasks/${task.id}`}
      data-testid="task-card-link"
      className="block rounded-lg border bg-card p-3 shadow-sm transition-shadow hover:shadow-md"
    >
      <div className="flex items-center gap-2">
        <span className="text-sm font-semibold text-muted-foreground">
          {task.taskId}
        </span>
        <PriorityBadge priority={task.priority} />
      </div>
      <p className="mt-1 text-sm font-medium">{task.title}</p>
      {task.tags.length > 0 && (
        <div className="mt-1 flex flex-wrap gap-1">
          {task.tags.map((tag) => (
            <span
              key={tag}
              className="rounded bg-secondary px-1.5 py-0.5 text-xs text-secondary-foreground"
            >
              {tag}
            </span>
          ))}
        </div>
      )}
      {task.claimedBy && (
        <p className="mt-1 text-xs text-muted-foreground">
          {task.claimedBy}
        </p>
      )}
    </Link>
  )
}

// ── KanbanColumn ──

function KanbanColumn({
  status,
  label,
  tasks,
}: {
  status: TaskStatus
  label: string
  tasks: TaskSummary[]
}) {
  return (
    <div data-testid={`column-${status}`} className="min-w-[200px]">
      <h3 className="mb-2 text-sm font-semibold text-muted-foreground">
        {label} ({tasks.length})
      </h3>
      <div className="space-y-2">
        {tasks.length === 0 ? (
          <div className="rounded-lg border border-dashed p-4 text-center text-muted-foreground">
            {'\u2014'}
          </div>
        ) : (
          tasks.map((task) => <TaskCard key={task.id} task={task} />)
        )}
      </div>
    </div>
  )
}

// ── Multi-Select Filter ──

function MultiSelectFilter({
  label,
  options,
  selected,
  onChange,
  testId,
}: {
  label: string
  options: readonly string[]
  selected: string[]
  onChange: (selected: string[]) => void
  testId: string
}) {
  const handleToggle = (option: string) => {
    if (selected.includes(option)) {
      onChange(selected.filter((s) => s !== option))
    } else {
      onChange([...selected, option])
    }
  }

  const displayLabel =
    selected.length === 0
      ? label
      : `${label}: ${selected.join(', ')}`

  return (
    <div className="relative" data-testid={testId}>
      <details className="group">
        <summary className="cursor-pointer rounded-md border px-3 py-1.5 text-sm hover:bg-accent">
          {displayLabel}
        </summary>
        <div
          data-testid={`${testId}-dropdown`}
          className="absolute z-10 mt-1 rounded-md border bg-popover p-1 shadow-md"
        >
          {options.map((option) => (
            <label
              key={option}
              role="option"
              aria-selected={selected.includes(option)}
              className="flex cursor-pointer items-center gap-2 rounded px-2 py-1 text-sm hover:bg-accent"
            >
              <input
                type="checkbox"
                checked={selected.includes(option)}
                onChange={() => handleToggle(option)}
                className="rounded"
              />
              {option}
            </label>
          ))}
        </div>
      </details>
    </div>
  )
}

// ── FilterBar ──

function FilterBar({
  priority,
  tag,
  status,
  allTags,
  onPriorityChange,
  onTagChange,
  onStatusChange,
  onClear,
  hasFilters,
}: {
  priority: string[]
  tag: string[]
  status: string[]
  allTags: string[]
  onPriorityChange: (val: string[]) => void
  onTagChange: (val: string[]) => void
  onStatusChange: (val: string[]) => void
  onClear: () => void
  hasFilters: boolean
}) {
  return (
    <div className="mt-4 flex flex-wrap items-center gap-2">
      <MultiSelectFilter
        label="Priority"
        options={PRIORITY_OPTIONS}
        selected={priority}
        onChange={onPriorityChange}
        testId="filter-priority"
      />
      <MultiSelectFilter
        label="Tags"
        options={allTags}
        selected={tag}
        onChange={onTagChange}
        testId="filter-tag"
      />
      <MultiSelectFilter
        label="Status"
        options={STATUS_OPTIONS}
        selected={status}
        onChange={onStatusChange}
        testId="filter-status"
      />
      {hasFilters && (
        <button
          type="button"
          data-testid="clear-filters"
          onClick={onClear}
          className="rounded-md border px-3 py-1.5 text-sm text-muted-foreground hover:bg-accent"
        >
          Clear
        </button>
      )}
    </div>
  )
}

// ── Main Page ──

export default function FeatureKanbanPage() {
  const { id } = useParams<{ id: string }>()
  const [searchParams, setSearchParams] = useSearchParams()
  const queryClient = useQueryClient()

  // Read filters from URL
  const priorityFilter = parseCsv(searchParams.get('priority'))
  const tagFilter = parseCsv(searchParams.get('tag'))
  const statusFilter = parseCsv(searchParams.get('status'))

  const hasFilters =
    priorityFilter.length > 0 ||
    tagFilter.length > 0 ||
    statusFilter.length > 0

  // Build API filter object
  const apiFilter: TaskFilter = useMemo(() => {
    const filter: TaskFilter = {}
    if (priorityFilter.length > 0) filter.priority = priorityFilter.join(',')
    if (tagFilter.length > 0) filter.tag = tagFilter.join(',')
    if (statusFilter.length > 0) filter.status = statusFilter.join(',')
    return filter
  }, [priorityFilter, tagFilter, statusFilter])

  const { data, isLoading, isError, refetch } = useQuery({
    queryKey: ['feature-tasks', id, apiFilter],
    queryFn: () => getFeatureTasks(Number(id), apiFilter),
  })

  // Aggregate unique tags from tasks for TagSelect
  const allTags = useMemo(() => {
    if (!data?.tasks) return []
    const tagSet = new Set<string>()
    data.tasks.forEach((t) => t.tags.forEach((tag) => tagSet.add(tag)))
    return Array.from(tagSet).sort()
  }, [data?.tasks])

  // Group tasks by status
  const tasksByStatus = useMemo(() => {
    const grouped: Record<TaskStatus, TaskSummary[]> = {
      pending: [],
      in_progress: [],
      completed: [],
      blocked: [],
    }
    if (data?.tasks) {
      data.tasks.forEach((task) => {
        grouped[task.status].push(task)
      })
    }
    return grouped
  }, [data?.tasks])

  // Update URL search params
  const updateFilters = useCallback(
    (updates: Record<string, string[]>) => {
      setSearchParams(
        (prev) => {
          const next = new URLSearchParams(prev)
          for (const [key, values] of Object.entries(updates)) {
            if (values.length > 0) {
              next.set(key, values.join(','))
            } else {
              next.delete(key)
            }
          }
          return next
        },
        { replace: true },
      )
    },
    [setSearchParams],
  )

  const handleClear = () => {
    setSearchParams({}, { replace: true })
  }

  const handleUploadSuccess = () => {
    queryClient.invalidateQueries({ queryKey: ['feature-tasks', id] })
  }

  return (
    <div className="flex min-h-screen flex-col">
      <AppHeader
        projectName={data?.featureName}
        onUploadSuccess={handleUploadSuccess}
      />
      <main className="flex-1 p-6">
        <Link
          to={`/projects/${id}`}
          className="text-sm text-muted-foreground hover:underline"
        >
          &larr; Back to project
        </Link>

        {isLoading ? (
          <>
            <div className="mt-4 h-8 w-48 animate-pulse rounded bg-muted" />
            <KanbanSkeleton />
          </>
        ) : isError ? (
          <ErrorState message="Failed to load tasks" onRetry={() => refetch()} />
        ) : data ? (
          <>
            <h1 className="mt-4 text-2xl font-bold">{data.featureName}</h1>

            <FilterBar
              priority={priorityFilter}
              tag={tagFilter}
              status={statusFilter}
              allTags={allTags}
              onPriorityChange={(val) =>
                updateFilters({ priority: val })
              }
              onTagChange={(val) => updateFilters({ tag: val })}
              onStatusChange={(val) =>
                updateFilters({ status: val })
              }
              onClear={handleClear}
              hasFilters={hasFilters}
            />

            <div className="mt-4 grid grid-cols-4 gap-4 overflow-x-auto">
              {COLUMNS.map((col) => (
                <KanbanColumn
                  key={col.status}
                  status={col.status}
                  label={col.label}
                  tasks={tasksByStatus[col.status]}
                />
              ))}
            </div>
          </>
        ) : null}
      </main>
    </div>
  )
}
