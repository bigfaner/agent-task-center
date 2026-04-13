import { useState, useCallback } from 'react'
import { useParams, Link, useLocation } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { AppHeader } from '@/components/AppHeader'
import { StatusBadge } from '@/components/StatusBadge'
import { PriorityBadge } from '@/components/PriorityBadge'
import { MarkdownRenderer } from '@/components/MarkdownRenderer'
import { ErrorState } from '@/components/ErrorState'
import { getTask, listTaskRecords } from '@/api/tasks'
import type { ExecutionRecord } from '@/types'

// ── Skeleton ──

function DetailSkeleton() {
  return (
    <div data-testid="task-detail-skeleton" className="space-y-6 animate-pulse">
      <div className="h-8 w-64 rounded bg-muted" />
      <div className="flex gap-2">
        <div className="h-6 w-20 rounded-full bg-muted" />
        <div className="h-6 w-16 rounded-full bg-muted" />
      </div>
      <div className="h-40 w-full rounded bg-muted" />
      <div className="h-6 w-48 rounded bg-muted" />
      <div className="space-y-3">
        <div className="h-20 w-full rounded bg-muted" />
        <div className="h-20 w-full rounded bg-muted" />
      </div>
    </div>
  )
}

// ── AcceptanceCriterionRow ──

function CriterionRow({
  criterion,
  met,
}: {
  criterion: string
  met: boolean
}) {
  return (
    <div
      data-testid="acceptance-criterion"
      data-met={String(met)}
      className="flex items-start gap-2 text-sm"
    >
      <span className="mt-0.5 flex-shrink-0">
        {met ? (
          <span className="text-green-600">&#10003;</span>
        ) : (
          <span className="text-red-600">&#10007;</span>
        )}
      </span>
      <span>{criterion}</span>
    </div>
  )
}

// ── RecordDetail ──

function RecordDetail({ record }: { record: ExecutionRecord }) {
  return (
    <div data-testid="record-detail" className="mt-3 space-y-3 border-t pt-3 text-sm">
      {record.filesCreated.length > 0 && (
        <div>
          <h5 className="mb-1 font-medium text-muted-foreground">
            Files Created
          </h5>
          <ul className="list-disc pl-5 space-y-0.5">
            {record.filesCreated.map((f) => (
              <li key={f}>{f}</li>
            ))}
          </ul>
        </div>
      )}
      {record.filesModified.length > 0 && (
        <div>
          <h5 className="mb-1 font-medium text-muted-foreground">
            Files Modified
          </h5>
          <ul className="list-disc pl-5 space-y-0.5">
            {record.filesModified.map((f) => (
              <li key={f}>{f}</li>
            ))}
          </ul>
        </div>
      )}
      {record.keyDecisions.length > 0 && (
        <div>
          <h5 className="mb-1 font-medium text-muted-foreground">
            Key Decisions
          </h5>
          <ul className="list-disc pl-5 space-y-0.5">
            {record.keyDecisions.map((d) => (
              <li key={d}>{d}</li>
            ))}
          </ul>
        </div>
      )}
      <div data-testid="test-results" className="flex items-center gap-3">
        <span className="text-green-600">&#10003; {record.testsPassed}</span>
        <span className="text-red-600">&#10007; {record.testsFailed}</span>
        <span>Coverage {record.coverage}%</span>
      </div>
      {record.acceptanceCriteria.length > 0 && (
        <div>
          <h5 className="mb-1 font-medium text-muted-foreground">
            Acceptance Criteria
          </h5>
          <div className="space-y-1">
            {record.acceptanceCriteria.map((ac, i) => (
              <CriterionRow key={i} criterion={ac.criterion} met={ac.met} />
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// ── RecordItem ──

function RecordItem({ record }: { record: ExecutionRecord }) {
  const [expanded, setExpanded] = useState(false)

  const timeStr = new Date(record.createdAt).toLocaleString()

  return (
    <div data-testid="record-item" className="flex gap-3">
      {/* Timeline connector */}
      <div className="flex flex-col items-center">
        <div className="mt-1.5 h-3 w-3 rounded-full bg-primary flex-shrink-0" />
        <div className="w-px flex-1 bg-border" />
      </div>
      <div className="flex-1 pb-4">
        <div className="flex items-center gap-2 text-sm">
          <span className="text-muted-foreground">{timeStr}</span>
          <span className="font-medium">{record.agentId}</span>
        </div>
        <p className="mt-1 text-sm">{record.summary}</p>
        <button
          type="button"
          data-testid="expand-toggle"
          onClick={() => setExpanded(!expanded)}
          className="mt-1 text-xs text-muted-foreground hover:underline"
        >
          {expanded ? 'Collapse \u25B2' : 'Expand \u25BC'}
        </button>
        {expanded && <RecordDetail record={record} />}
      </div>
    </div>
  )
}

// ── Main Page ──

export default function TaskDetailPage() {
  const { id } = useParams<{ id: string }>()
  const location = useLocation()
  const numericId = Number(id)

  const [records, setRecords] = useState<ExecutionRecord[]>([])
  const [nextPage, setNextPage] = useState(1)
  const [totalRecords, setTotalRecords] = useState(0)

  const {
    data: task,
    isLoading: taskLoading,
    isError: taskError,
    refetch: refetchTask,
  } = useQuery({
    queryKey: ['task', numericId],
    queryFn: () => getTask(numericId),
  })

  const {
    isLoading: recordsLoading,
  } = useQuery({
    queryKey: ['task-records', numericId],
    queryFn: async () => {
      const result = await listTaskRecords(numericId, {
        page: 1,
        pageSize: 10,
      })
      setRecords(result.items)
      setTotalRecords(result.total)
      setNextPage(result.page + 1)
      return result
    },
    enabled: !!task,
  })

  const hasMore = records.length < totalRecords

  const handleLoadMore = useCallback(async () => {
    const result = await listTaskRecords(numericId, {
      page: nextPage,
      pageSize: 10,
    })
    setRecords((prev) => [...prev, ...result.items])
    setTotalRecords(result.total)
    setNextPage((prev) => prev + 1)
  }, [numericId, nextPage])

  const isLoading = taskLoading || recordsLoading

  // Determine back link destination from location state
  const backTo = (location.state as { from?: string } | undefined)?.from

  return (
    <div className="flex min-h-screen flex-col">
      <AppHeader showUpload={false} />
      <main className="flex-1 p-6">
        <Link
          to={backTo ?? '..'}
          className="text-sm text-muted-foreground hover:underline"
        >
          &larr; Back to kanban
        </Link>

        {isLoading ? (
          <DetailSkeleton />
        ) : taskError ? (
          <ErrorState
            message="Failed to load task"
            onRetry={() => refetchTask()}
          />
        ) : task ? (
          <>
            {/* Task Header */}
            <div className="mt-4">
              <h1 className="text-2xl font-bold">
                {task.taskId} {task.title}
              </h1>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <StatusBadge status={task.status} />
                <PriorityBadge priority={task.priority} />
                {task.claimedBy && (
                  <span data-testid="claimed-by" className="text-sm text-muted-foreground">
                    {task.claimedBy}
                  </span>
                )}
              </div>
              {task.tags.length > 0 && (
                <div className="mt-2 flex flex-wrap gap-1">
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
              {task.dependencies.length > 0 && (
                <div className="mt-2 flex flex-wrap items-center gap-1 text-sm">
                  <span className="text-muted-foreground">Dependencies:</span>
                  {task.dependencies.map((dep) => (
                    <span key={dep} className="text-muted-foreground">
                      {dep}
                    </span>
                  ))}
                </div>
              )}
            </div>

            {/* Description */}
            <div className="mt-6">
              <h2 className="mb-2 text-lg font-semibold">Description</h2>
              <div className="rounded-lg border p-4">
                <MarkdownRenderer content={task.description} />
              </div>
            </div>

            {/* Execution Records */}
            <div className="mt-6">
              <h2
                data-testid="records-heading"
                className="mb-3 text-lg font-semibold"
              >
                Execution Records ({totalRecords})
              </h2>

              {records.length === 0 && !recordsLoading ? (
                <div
                  data-testid="no-records"
                  className="rounded-lg border border-dashed p-6 text-center text-muted-foreground"
                >
                  No execution records
                </div>
              ) : (
                <div>
                  {records.map((record) => (
                    <RecordItem key={record.id} record={record} />
                  ))}
                  {hasMore && (
                    <button
                      type="button"
                      data-testid="load-more-btn"
                      onClick={handleLoadMore}
                      className="mt-2 rounded-md border px-4 py-2 text-sm hover:bg-accent"
                    >
                      Load more
                    </button>
                  )}
                </div>
              )}
            </div>
          </>
        ) : null}
      </main>
    </div>
  )
}
