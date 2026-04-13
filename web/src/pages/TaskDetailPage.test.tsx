import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import TaskDetailPage from './TaskDetailPage'
import * as tasksApi from '@/api/tasks'
import type { TaskDetail, ListRecordsResponse } from '@/types'

// Mock child components that are tested separately
vi.mock('@/components/StatusBadge', () => ({
  StatusBadge: ({ status }: { status: string }) => (
    <span data-testid="status-badge" data-status={status}>
      {status}
    </span>
  ),
}))

vi.mock('@/components/PriorityBadge', () => ({
  PriorityBadge: ({ priority }: { priority: string }) => (
    <span data-testid="priority-badge" data-priority={priority}>
      {priority}
    </span>
  ),
}))

vi.mock('@/components/MarkdownRenderer', () => ({
  MarkdownRenderer: ({ content }: { content: string }) => (
    <div data-testid="markdown-renderer">{content}</div>
  ),
}))

vi.mock('@/components/AppHeader', () => ({
  AppHeader: ({
    projectName,
    onUploadSuccess,
  }: {
    projectName?: string
    onUploadSuccess?: () => void
  }) => (
    <header data-testid="app-header">
      <span>{projectName}</span>
      <button data-testid="upload-btn" onClick={onUploadSuccess}>
        Upload
      </button>
    </header>
  ),
}))

vi.mock('@/components/ErrorState', () => ({
  ErrorState: ({
    message,
    onRetry,
  }: {
    message?: string
    onRetry?: () => void
  }) => (
    <div data-testid="error-state">
      <span>{message}</span>
      {onRetry && (
        <button data-testid="retry-btn" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  ),
}))

// Helper to show current URL
function LocationDisplay() {
  const location = useLocation()
  return (
    <div data-testid="location">
      {location.pathname}
      {location.search}
    </div>
  )
}

// ── Mock Data ──

const mockTaskDetail: TaskDetail = {
  id: 101,
  taskId: '1.1',
  title: 'Setup project scaffold',
  description: '## Task Description\n\nImplement all scaffolding...',
  status: 'completed',
  priority: 'P0',
  tags: ['core', 'setup'],
  claimedBy: 'agent-01',
  dependencies: ['1.0'],
  createdAt: '2026-04-12T10:00:00Z',
  updatedAt: '2026-04-12T14:30:00Z',
}

const mockRecords: ListRecordsResponse = {
  items: [
    {
      id: 1,
      agentId: 'agent-01',
      summary: 'Implemented auth middleware',
      filesCreated: ['src/middleware/auth.go'],
      filesModified: ['server/main.go'],
      keyDecisions: ['Used JWT instead of session cookie'],
      testsPassed: 12,
      testsFailed: 0,
      coverage: 85.6,
      acceptanceCriteria: [
        { criterion: 'Unauthenticated requests return 401', met: true },
        { criterion: 'Valid tokens pass verification', met: true },
      ],
      createdAt: '2026-04-13T10:30:00Z',
    },
    {
      id: 2,
      agentId: 'agent-02',
      summary: 'Initial scaffold',
      filesCreated: ['server/go.mod', 'web/package.json'],
      filesModified: [],
      keyDecisions: ['Used chi as HTTP router'],
      testsPassed: 8,
      testsFailed: 1,
      coverage: 72.0,
      acceptanceCriteria: [
        { criterion: 'Project compiles', met: true },
        { criterion: 'Tests pass', met: false },
      ],
      createdAt: '2026-04-12T15:00:00Z',
    },
  ],
  total: 2,
  page: 1,
  pageSize: 10,
}

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  })
}

function renderPage(taskId = '101') {
  const qc = createQueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/tasks/${taskId}`]}>
        <Routes>
          <Route path="/tasks/:id" element={<TaskDetailPage />} />
          <Route
            path="/features/:id/tasks"
            element={<div data-testid="navigated-kanban" />}
          />
        </Routes>
        <LocationDisplay />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('TaskDetailPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  // ── Loading state ──

  it('shows loading skeleton on initial load', async () => {
    vi.spyOn(tasksApi, 'getTask').mockReturnValue(
      new Promise(() => {}) as Promise<TaskDetail>,
    )
    vi.spyOn(tasksApi, 'listTaskRecords').mockReturnValue(
      new Promise(() => {}) as Promise<ListRecordsResponse>,
    )
    renderPage()

    expect(screen.getByTestId('task-detail-skeleton')).toBeInTheDocument()
  })

  // ── Task basic info ──

  it('renders task ID and title as heading', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()
    const heading = await screen.findByRole('heading', { level: 1 })
    expect(heading).toHaveTextContent('1.1 Setup project scaffold')
  })

  it('renders status and priority badges', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()
    await screen.findByRole('heading', { level: 1 })

    expect(screen.getByTestId('status-badge')).toHaveAttribute(
      'data-status',
      'completed',
    )
    expect(screen.getByTestId('priority-badge')).toHaveAttribute(
      'data-priority',
      'P0',
    )
  })

  it('renders claimedBy text when task is claimed', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()
    await screen.findByRole('heading', { level: 1 })

    expect(screen.getByText('agent-01')).toBeInTheDocument()
  })

  it('does not render claimedBy section when null', async () => {
    const unclaimed = { ...mockTaskDetail, claimedBy: null }
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(unclaimed)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()
    await screen.findByRole('heading', { level: 1 })

    expect(screen.queryByTestId('claimed-by')).not.toBeInTheDocument()
  })

  it('renders tags', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()
    await screen.findByRole('heading', { level: 1 })

    expect(screen.getByText('core')).toBeInTheDocument()
    expect(screen.getByText('setup')).toBeInTheDocument()
  })

  it('renders dependencies as text', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()
    await screen.findByRole('heading', { level: 1 })

    expect(screen.getByText('1.0')).toBeInTheDocument()
  })

  // ── Markdown description ──

  it('renders task description via MarkdownRenderer', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()
    await screen.findByRole('heading', { level: 1 })

    const md = screen.getByTestId('markdown-renderer')
    expect(md).toHaveTextContent('## Task Description')
  })

  // ── Back link ──

  it('renders back link to kanban', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()
    const backLink = await screen.findByRole('link', { name: /back/i })
    expect(backLink).toBeInTheDocument()
  })

  // ── Execution Records ──

  it('renders execution records with count in heading', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    const recordsHeading = await screen.findByTestId('records-heading')
    expect(recordsHeading).toHaveTextContent('Execution Records (2)')
  })

  it('renders record items with timestamp, agentId, and summary', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    const items = screen.getAllByTestId('record-item')
    expect(items).toHaveLength(2)

    // First record (newest first)
    expect(items[0]).toHaveTextContent('agent-01')
    expect(items[0]).toHaveTextContent('Implemented auth middleware')

    // Second record
    expect(items[1]).toHaveTextContent('agent-02')
    expect(items[1]).toHaveTextContent('Initial scaffold')
  })

  // ── Record expand/collapse ──

  it('record details are collapsed by default', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    // Details should not be visible initially
    expect(screen.queryByTestId('record-detail')).not.toBeInTheDocument()
  })

  it('expands record detail when expand button is clicked', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    // Click expand on the first record
    const expandButtons = screen.getAllByTestId('expand-toggle')
    await userEvent.click(expandButtons[0])

    // Detail should now be visible
    const details = screen.getAllByTestId('record-detail')
    expect(details).toHaveLength(1)
  })

  it('shows files created list in expanded record', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    const expandButtons = screen.getAllByTestId('expand-toggle')
    await userEvent.click(expandButtons[0])

    const detail = screen.getByTestId('record-detail')
    expect(detail).toHaveTextContent('src/middleware/auth.go')
  })

  it('shows files modified list in expanded record', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    const expandButtons = screen.getAllByTestId('expand-toggle')
    await userEvent.click(expandButtons[0])

    const detail = screen.getByTestId('record-detail')
    expect(detail).toHaveTextContent('server/main.go')
  })

  it('shows key decisions in expanded record', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    const expandButtons = screen.getAllByTestId('expand-toggle')
    await userEvent.click(expandButtons[0])

    const detail = screen.getByTestId('record-detail')
    expect(detail).toHaveTextContent('Used JWT instead of session cookie')
  })

  it('shows test results row with passed/failed/coverage', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    const expandButtons = screen.getAllByTestId('expand-toggle')
    await userEvent.click(expandButtons[0])

    const testResults = screen.getByTestId('test-results')
    expect(testResults).toHaveTextContent('12')
    expect(testResults).toHaveTextContent('0')
    expect(testResults).toHaveTextContent('85.6%')
  })

  it('shows acceptance criteria with check/cross icons', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    const expandButtons = screen.getAllByTestId('expand-toggle')
    await userEvent.click(expandButtons[0])

    const criteria = screen.getAllByTestId('acceptance-criterion')
    expect(criteria).toHaveLength(2)

    // First criterion is met (check)
    expect(criteria[0]).toHaveAttribute('data-met', 'true')
    expect(criteria[0]).toHaveTextContent('Unauthenticated requests return 401')

    // Second criterion is met (check)
    expect(criteria[1]).toHaveAttribute('data-met', 'true')
    expect(criteria[1]).toHaveTextContent('Valid tokens pass verification')
  })

  it('shows cross icon for unmet acceptance criteria', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    const expandButtons = screen.getAllByTestId('expand-toggle')
    await userEvent.click(expandButtons[1]) // Second record has unmet criteria

    const criteria = screen.getAllByTestId('acceptance-criterion')
    // "Tests pass" criterion is not met
    const unmetCriterion = criteria.find(
      (c) => c.getAttribute('data-met') === 'false',
    )
    expect(unmetCriterion).toBeInTheDocument()
    expect(unmetCriterion).toHaveTextContent('Tests pass')
  })

  // ── No records state ──

  it('shows empty state when there are no execution records', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()
    await screen.findByRole('heading', { level: 1 })

    expect(screen.getByTestId('no-records')).toHaveTextContent(
      'No execution records',
    )
  })

  // ── Load more ──

  it('shows load more button when there are more records', async () => {
    const manyRecords: ListRecordsResponse = {
      items: Array.from({ length: 10 }, (_, i) => ({
        id: i + 1,
        agentId: 'agent-01',
        summary: `Record ${i + 1}`,
        filesCreated: [],
        filesModified: [],
        keyDecisions: [],
        testsPassed: 0,
        testsFailed: 0,
        coverage: 0,
        acceptanceCriteria: [],
        createdAt: '2026-04-13T10:00:00Z',
      })),
      total: 15,
      page: 1,
      pageSize: 10,
    }
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(manyRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    expect(screen.getByTestId('load-more-btn')).toBeInTheDocument()
  })

  it('does not show load more button when all records are loaded', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue(mockRecords)
    renderPage()
    await screen.findByTestId('records-heading')

    expect(screen.queryByTestId('load-more-btn')).not.toBeInTheDocument()
  })

  it('loads next page when load more is clicked', async () => {
    const page1: ListRecordsResponse = {
      items: Array.from({ length: 10 }, (_, i) => ({
        id: i + 1,
        agentId: 'agent-01',
        summary: `Record ${i + 1}`,
        filesCreated: [],
        filesModified: [],
        keyDecisions: [],
        testsPassed: 0,
        testsFailed: 0,
        coverage: 0,
        acceptanceCriteria: [],
        createdAt: '2026-04-13T10:00:00Z',
      })),
      total: 15,
      page: 1,
      pageSize: 10,
    }
    const page2: ListRecordsResponse = {
      items: Array.from({ length: 5 }, (_, i) => ({
        id: i + 11,
        agentId: 'agent-02',
        summary: `Record ${i + 11}`,
        filesCreated: [],
        filesModified: [],
        keyDecisions: [],
        testsPassed: 0,
        testsFailed: 0,
        coverage: 0,
        acceptanceCriteria: [],
        createdAt: '2026-04-12T10:00:00Z',
      })),
      total: 15,
      page: 2,
      pageSize: 10,
    }

    const recordsSpy = vi
      .spyOn(tasksApi, 'listTaskRecords')
      .mockResolvedValueOnce(page1)
      .mockResolvedValueOnce(page2)
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    renderPage()
    await screen.findByTestId('records-heading')

    // Initially 10 items
    expect(screen.getAllByTestId('record-item')).toHaveLength(10)

    // Click load more
    await userEvent.click(screen.getByTestId('load-more-btn'))

    // Should now show all 15
    await waitFor(() => {
      expect(screen.getAllByTestId('record-item')).toHaveLength(15)
    })
    expect(recordsSpy).toHaveBeenCalledTimes(2)
    expect(recordsSpy).toHaveBeenLastCalledWith(101, { page: 2, pageSize: 10 })
  })

  // ── Error state ──

  it('shows error state when task fetch fails', async () => {
    vi.spyOn(tasksApi, 'getTask').mockRejectedValue(new Error('Network error'))
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage()

    const errorState = await screen.findByTestId('error-state')
    expect(errorState).toBeInTheDocument()
  })

  // ── API calls ──

  it('calls getTask with correct id', async () => {
    const taskSpy = vi
      .spyOn(tasksApi, 'getTask')
      .mockResolvedValue(mockTaskDetail)
    vi.spyOn(tasksApi, 'listTaskRecords').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 10,
    })
    renderPage('42')
    await screen.findByRole('heading', { level: 1 })

    expect(taskSpy).toHaveBeenCalledWith(42)
  })

  it('calls listTaskRecords with correct id', async () => {
    vi.spyOn(tasksApi, 'getTask').mockResolvedValue(mockTaskDetail)
    const recordsSpy = vi
      .spyOn(tasksApi, 'listTaskRecords')
      .mockResolvedValue({
        items: [],
        total: 0,
        page: 1,
        pageSize: 10,
      })
    renderPage('101')
    await screen.findByRole('heading', { level: 1 })

    expect(recordsSpy).toHaveBeenCalledWith(101, { page: 1, pageSize: 10 })
  })
})
