import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes, useLocation } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import FeatureKanbanPage from './FeatureKanbanPage'
import * as featuresApi from '@/api/features'
import type { FeatureTasksResponse } from '@/types'

// Mock child components that are tested separately
vi.mock('@/components/PriorityBadge', () => ({
  PriorityBadge: ({ priority }: { priority: string }) => (
    <span data-testid="priority-badge" data-priority={priority}>
      {priority}
    </span>
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

const mockFeatureTasks: FeatureTasksResponse = {
  featureId: 1,
  featureName: 'Authentication',
  tasks: [
    {
      id: 101,
      taskId: '1.1',
      title: 'Setup project scaffold',
      status: 'completed',
      priority: 'P0',
      tags: ['core', 'setup'],
      claimedBy: 'agent-01',
      dependencies: [],
    },
    {
      id: 102,
      taskId: '1.2',
      title: 'Auth middleware',
      status: 'in_progress',
      priority: 'P1',
      tags: ['core', 'api'],
      claimedBy: 'agent-02',
      dependencies: [],
    },
    {
      id: 103,
      taskId: '1.3',
      title: 'Login page UI',
      status: 'pending',
      priority: 'P1',
      tags: ['ui'],
      claimedBy: null,
      dependencies: [],
    },
    {
      id: 104,
      taskId: '1.4',
      title: 'Deploy pipeline',
      status: 'blocked',
      priority: 'P2',
      tags: ['devops'],
      claimedBy: null,
      dependencies: [],
    },
    {
      id: 105,
      taskId: '1.5',
      title: 'Write tests',
      status: 'pending',
      priority: 'P0',
      tags: ['core', 'testing'],
      claimedBy: null,
      dependencies: [],
    },
  ],
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

function renderPage(
  featureId = '1',
  initialSearch = '',
) {
  const qc = createQueryClient()
  const initialEntry = `/features/${featureId}/tasks${initialSearch}`
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <Routes>
          <Route
            path="/features/:id/tasks"
            element={<FeatureKanbanPage />}
          />
          <Route
            path="/tasks/:id"
            element={<div data-testid="navigated-task-detail" />}
          />
          <Route
            path="/projects/:id"
            element={<div data-testid="navigated-project" />}
          />
        </Routes>
        <LocationDisplay />
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('FeatureKanbanPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  // ── Loading state ──

  it('shows loading skeletons on initial load', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockReturnValue(
      new Promise(() => {}) as Promise<FeatureTasksResponse>,
    )
    renderPage()

    // Should have skeleton cards in four columns
    const skeletons = screen.getAllByTestId('kanban-skeleton')
    expect(skeletons.length).toBeGreaterThanOrEqual(4)
  })

  // ── Feature name and back link ──

  it('renders feature name as heading', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    const heading = await screen.findByRole('heading', {
      name: 'Authentication',
    })
    expect(heading).toBeInTheDocument()
  })

  it('renders back link to project page', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    const backLink = await screen.findByRole('link', { name: /back/i })
    expect(backLink).toBeInTheDocument()
  })

  // ── Kanban columns ──

  it('renders four kanban columns with correct headers', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    expect(screen.getByTestId('column-pending')).toHaveTextContent('Pending')
    expect(screen.getByTestId('column-in_progress')).toHaveTextContent(
      'In Progress',
    )
    expect(screen.getByTestId('column-completed')).toHaveTextContent(
      'Completed',
    )
    expect(screen.getByTestId('column-blocked')).toHaveTextContent('Blocked')
  })

  it('renders column task count in headers', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    // Pending: 1.3 and 1.5 = 2
    expect(screen.getByTestId('column-pending')).toHaveTextContent('(2)')
    // In Progress: 1.2 = 1
    expect(screen.getByTestId('column-in_progress')).toHaveTextContent('(1)')
    // Completed: 1.1 = 1
    expect(screen.getByTestId('column-completed')).toHaveTextContent('(1)')
    // Blocked: 1.4 = 1
    expect(screen.getByTestId('column-blocked')).toHaveTextContent('(1)')
  })

  it('groups tasks into correct columns by status', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    const pendingCol = screen.getByTestId('column-pending')
    expect(within(pendingCol).getByText('Login page UI')).toBeInTheDocument()
    expect(within(pendingCol).getByText('Write tests')).toBeInTheDocument()

    const inProgressCol = screen.getByTestId('column-in_progress')
    expect(
      within(inProgressCol).getByText('Auth middleware'),
    ).toBeInTheDocument()

    const completedCol = screen.getByTestId('column-completed')
    expect(
      within(completedCol).getByText('Setup project scaffold'),
    ).toBeInTheDocument()

    const blockedCol = screen.getByTestId('column-blocked')
    expect(
      within(blockedCol).getByText('Deploy pipeline'),
    ).toBeInTheDocument()
  })

  // ── TaskCard ──

  it('renders task cards with ID, title, priority badge, and tags', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    // Task 1.1 in Completed column
    const completedCol = screen.getByTestId('column-completed')
    expect(within(completedCol).getByText('1.1')).toBeInTheDocument()
    expect(
      within(completedCol).getByText('Setup project scaffold'),
    ).toBeInTheDocument()
    expect(
      within(completedCol).getByTestId('priority-badge'),
    ).toHaveAttribute('data-priority', 'P0')
    expect(within(completedCol).getByText('core')).toBeInTheDocument()
    expect(within(completedCol).getByText('setup')).toBeInTheDocument()
  })

  it('renders claimedBy text on task cards', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    // Task 1.2 is claimed by agent-02
    const inProgressCol = screen.getByTestId('column-in_progress')
    expect(within(inProgressCol).getByText('agent-02')).toBeInTheDocument()
  })

  it('task cards link to /tasks/:id', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    const cardLinks = screen.getAllByTestId('task-card-link')
    const hrefs = cardLinks.map((el) => el.getAttribute('href'))
    expect(hrefs).toContain('/tasks/101')
    expect(hrefs).toContain('/tasks/102')
    expect(hrefs).toContain('/tasks/103')
  })

  // ── FilterBar ──

  it('renders filter controls', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    expect(screen.getByTestId('filter-priority')).toBeInTheDocument()
    expect(screen.getByTestId('filter-tag')).toBeInTheDocument()
    expect(screen.getByTestId('filter-status')).toBeInTheDocument()
  })

  it('shows clear button only when filters are active', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    // Start with filters in URL
    renderPage('1', '?priority=P0')

    await screen.findByRole('heading', { name: 'Authentication' })
    expect(screen.getByTestId('clear-filters')).toBeInTheDocument()
  })

  it('does not show clear button when no filters are active', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    expect(screen.queryByTestId('clear-filters')).not.toBeInTheDocument()
  })

  // ── URL state sync ──

  it('reads initial filters from URL search params', async () => {
    const spy = vi
      .spyOn(featuresApi, 'getFeatureTasks')
      .mockResolvedValue(mockFeatureTasks)
    renderPage('1', '?priority=P0,P1&tag=core&status=pending')

    await screen.findByRole('heading', { name: 'Authentication' })

    // Should have been called with the filter params from URL
    expect(spy).toHaveBeenCalledWith(
      1,
      expect.objectContaining({
        priority: 'P0,P1',
        tag: 'core',
        status: 'pending',
      }),
    )
  })

  it('passes filter params to API as comma-separated strings', async () => {
    const spy = vi
      .spyOn(featuresApi, 'getFeatureTasks')
      .mockResolvedValue(mockFeatureTasks)
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    // Initial load with no filters
    expect(spy).toHaveBeenCalledWith(1, {})

    // Now select P0 priority
    spy.mockClear()
    await userEvent.click(screen.getByTestId('filter-priority'))
    const priorityDropdown = screen.getByTestId('filter-priority-dropdown')
    await userEvent.click(within(priorityDropdown).getByText('P0'))

    expect(spy).toHaveBeenCalledWith(
      1,
      expect.objectContaining({ priority: 'P0' }),
    )
  })

  it('clears all filters when clear button is clicked', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage('1', '?priority=P0')
    await screen.findByRole('heading', { name: 'Authentication' })

    await userEvent.click(screen.getByTestId('clear-filters'))

    // URL should no longer have search params
    const location = screen.getByTestId('location')
    expect(location.textContent).not.toContain('priority')
  })

  // ── Empty column ──

  it('shows placeholder when a column is empty', async () => {
    const noBlocked: FeatureTasksResponse = {
      featureId: 1,
      featureName: 'Auth',
      tasks: [
        {
          id: 101,
          taskId: '1.1',
          title: 'Setup',
          status: 'pending',
          priority: 'P0',
          tags: [],
          claimedBy: null,
          dependencies: [],
        },
      ],
    }
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(noBlocked)
    renderPage()
    await screen.findByRole('heading', { name: 'Auth' })

    const blockedCol = screen.getByTestId('column-blocked')
    expect(within(blockedCol).getByText('\u2014')).toBeInTheDocument()
  })

  // ── API call ──

  it('calls getFeatureTasks with correct feature id', async () => {
    const spy = vi
      .spyOn(featuresApi, 'getFeatureTasks')
      .mockResolvedValue(mockFeatureTasks)
    renderPage('42')
    await screen.findByRole('heading', { name: 'Authentication' })

    expect(spy).toHaveBeenCalledWith(42, {})
  })

  // ── TagSelect dynamic options ──

  it('aggregates tags from all tasks for TagSelect options', async () => {
    vi.spyOn(featuresApi, 'getFeatureTasks').mockResolvedValue(
      mockFeatureTasks,
    )
    renderPage()
    await screen.findByRole('heading', { name: 'Authentication' })

    // Open the tag filter dropdown
    await userEvent.click(screen.getByTestId('filter-tag'))

    // Should show unique tags from all tasks
    const tagFilter = screen.getByTestId('filter-tag-dropdown')
    const tagOptions = within(tagFilter).getAllByRole('option')
    const tagTexts = tagOptions.map((o) => o.textContent)
    // Tags from mock: core, setup, api, ui, devops, testing
    expect(tagTexts).toContain('core')
    expect(tagTexts).toContain('setup')
    expect(tagTexts).toContain('api')
    expect(tagTexts).toContain('ui')
    expect(tagTexts).toContain('devops')
    expect(tagTexts).toContain('testing')
  })
})
