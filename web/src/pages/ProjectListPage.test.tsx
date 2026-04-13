import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { BrowserRouter } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import ProjectListPage from './ProjectListPage'
import * as projectsApi from '@/api/projects'
import type { ListProjectsResponse } from '@/types'

// Mock all child components that are tested separately
vi.mock('@/components/ProjectTable', () => ({
  ProjectTable: ({ projects }: { projects: unknown[] }) => (
    <div data-testid="project-table" data-count={projects.length}>
      ProjectTable ({projects.length} projects)
    </div>
  ),
}))

vi.mock('@/components/SearchInput', () => ({
  SearchInput: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
    <input
      data-testid="search-input"
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder="Search projects..."
    />
  ),
}))

vi.mock('@/components/TableSkeleton', () => ({
  TableSkeleton: ({ rows }: { rows: number }) => (
    <div data-testid="table-skeleton">Skeleton ({rows} rows)</div>
  ),
}))

vi.mock('@/components/EmptyState', () => ({
  EmptyState: ({ title, description, action }: { title: string; description?: string; action?: { label: string; onClick: () => void } }) => (
    <div data-testid="empty-state">
      <span>{title}</span>
      {description && <span>{description}</span>}
      {action && <button data-testid="empty-action" onClick={action.onClick}>{action.label}</button>}
    </div>
  ),
}))

vi.mock('@/components/ErrorState', () => ({
  ErrorState: ({ message, onRetry }: { message?: string; onRetry?: () => void }) => (
    <div data-testid="error-state">
      <span>{message ?? 'Error'}</span>
      {onRetry && <button data-testid="retry-btn" onClick={onRetry}>Retry</button>}
    </div>
  ),
}))

const mockResponse: ListProjectsResponse = {
  items: [
    {
      id: 1,
      name: 'my-app',
      featureCount: 3,
      taskTotal: 24,
      completionRate: 75.0,
      updatedAt: '2026-04-12T14:30:00Z',
    },
  ],
  total: 1,
  page: 1,
  pageSize: 20,
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

function renderPage() {
  const qc = createQueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <BrowserRouter>
        <ProjectListPage />
      </BrowserRouter>
    </QueryClientProvider>,
  )
}

describe('ProjectListPage', () => {
  beforeEach(() => {
    vi.useFakeTimers({ shouldAdvanceTime: true })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('shows skeleton on initial load', async () => {
    // Return a promise that never resolves to keep loading state
    vi.spyOn(projectsApi, 'listProjects').mockReturnValue(
      new Promise(() => {}) as Promise<ListProjectsResponse>,
    )
    renderPage()
    expect(screen.getByTestId('table-skeleton')).toBeInTheDocument()
  })

  it('renders project table when data loads', async () => {
    vi.spyOn(projectsApi, 'listProjects').mockResolvedValue(mockResponse)
    renderPage()

    // Wait for loading to finish
    await screen.findByTestId('project-table')
    expect(screen.getByTestId('project-table')).toHaveTextContent('1 projects')
  })

  it('shows empty state when no projects exist', async () => {
    vi.spyOn(projectsApi, 'listProjects').mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      pageSize: 20,
    })
    renderPage()

    await screen.findByTestId('empty-state')
    expect(screen.getByTestId('empty-state')).toHaveTextContent(
      '暂无项目，点击上传开始',
    )
  })

  it('shows filtered empty state when search yields no results', async () => {
    const spy = vi.spyOn(projectsApi, 'listProjects')
    // First call: return data (or empty without search)
    spy.mockResolvedValueOnce(mockResponse)
    // Second call (after search): return empty
    spy.mockResolvedValueOnce({
      items: [],
      total: 0,
      page: 1,
      pageSize: 20,
    })
    renderPage()

    await screen.findByTestId('project-table')

    // Type in search
    const searchInput = screen.getByTestId('search-input')
    await userEvent.type(searchInput, 'nonexistent')

    // Wait for debounce (300ms)
    vi.advanceTimersByTime(400)

    await screen.findByTestId('empty-state')
    expect(screen.getByTestId('empty-state')).toHaveTextContent(
      '未找到匹配项目',
    )
  })

  it('shows error state when API fails', async () => {
    vi.spyOn(projectsApi, 'listProjects').mockRejectedValue(
      new Error('Network error'),
    )
    renderPage()

    await screen.findByTestId('error-state')
    expect(screen.getByTestId('error-state')).toHaveTextContent(
      'Failed to load projects',
    )
  })

  it('shows retry button on error that refetches', async () => {
    const spy = vi.spyOn(projectsApi, 'listProjects')
    spy.mockRejectedValueOnce(new Error('Network error'))
    spy.mockResolvedValueOnce(mockResponse)
    renderPage()

    await screen.findByTestId('error-state')

    // Click retry
    await userEvent.click(screen.getByTestId('retry-btn'))

    // Should now show data
    await screen.findByTestId('project-table')
    expect(spy).toHaveBeenCalledTimes(2)
  })

  it('renders the heading "Projects"', async () => {
    vi.spyOn(projectsApi, 'listProjects').mockResolvedValue(mockResponse)
    renderPage()
    await screen.findByTestId('project-table')
    expect(screen.getByText('Projects')).toBeInTheDocument()
  })

  it('renders SearchInput', async () => {
    vi.spyOn(projectsApi, 'listProjects').mockResolvedValue(mockResponse)
    renderPage()
    await screen.findByTestId('project-table')
    expect(screen.getByTestId('search-input')).toBeInTheDocument()
  })

  it('passes search term to API after debounce', async () => {
    const spy = vi.spyOn(projectsApi, 'listProjects').mockResolvedValue(mockResponse)
    renderPage()

    // Initial load call
    await screen.findByTestId('project-table')
    expect(spy).toHaveBeenLastCalledWith({ search: '', page: 1, pageSize: 20 })

    // Type search
    const searchInput = screen.getByTestId('search-input')
    await userEvent.type(searchInput, 'my-app')

    // Advance past debounce
    vi.advanceTimersByTime(400)

    // Wait for refetch
    await screen.findByTestId('project-table')

    // Should have been called with search term
    expect(spy).toHaveBeenCalledWith({
      search: 'my-app',
      page: 1,
      pageSize: 20,
    })
  })
})
