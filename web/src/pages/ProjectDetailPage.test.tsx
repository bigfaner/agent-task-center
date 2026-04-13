import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import ProjectDetailPage from './ProjectDetailPage'
import * as projectsApi from '@/api/projects'
import type { ProjectDetail } from '@/types'

// Mock child components that are tested separately
vi.mock('@/components/StatusBadge', () => ({
  StatusBadge: ({ status }: { status: string }) => (
    <span data-testid="status-badge" data-status={status}>
      {status}
    </span>
  ),
}))

vi.mock('@/components/CompletionRateBar', () => ({
  CompletionRateBar: ({ rate }: { rate: number }) => (
    <span data-testid="completion-rate">{rate}%</span>
  ),
}))

vi.mock('@/components/RelativeTime', () => ({
  RelativeTime: ({ date }: { date: string }) => (
    <time data-testid="relative-time">{date}</time>
  ),
}))

vi.mock('@/components/TableSkeleton', () => ({
  TableSkeleton: ({ rows }: { rows: number }) => (
    <div data-testid="table-skeleton">Skeleton ({rows} rows)</div>
  ),
}))

vi.mock('@/components/EmptyState', () => ({
  EmptyState: ({ title }: { title: string }) => (
    <div data-testid="empty-state">{title}</div>
  ),
}))

vi.mock('@/components/UploadDialog', () => ({
  UploadDialog: () => <div data-testid="upload-dialog" />,
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

const mockProjectDetail: ProjectDetail = {
  id: 1,
  name: 'my-app',
  proposals: [
    {
      id: 10,
      slug: 'auth-proposal',
      title: 'Auth Proposal',
      createdAt: '2026-01-10T10:00:00Z',
      featureCount: 2,
    },
    {
      id: 11,
      slug: 'api-proposal',
      title: 'API Proposal',
      createdAt: '2026-02-15T12:00:00Z',
      featureCount: 3,
    },
  ],
  features: [
    {
      id: 20,
      slug: 'auth',
      name: 'Authentication',
      status: 'in-progress',
      completionRate: 60.0,
      updatedAt: '2026-04-12T14:30:00Z',
    },
    {
      id: 21,
      slug: 'dashboard',
      name: 'Dashboard',
      status: 'done',
      completionRate: 100,
      updatedAt: '2026-04-11T10:00:00Z',
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

function renderPage(projectId = '1') {
  const qc = createQueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/projects/${projectId}`]}>
        <Routes>
          <Route path="/projects/:id" element={<ProjectDetailPage />} />
          <Route
            path="/features/:id/tasks"
            element={<div data-testid="navigated-feature-tasks" />}
          />
          <Route
            path="/proposals/:id"
            element={<div data-testid="navigated-proposal" />}
          />
          <Route path="/" element={<div data-testid="navigated-home" />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('ProjectDetailPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('shows skeleton on initial load', async () => {
    vi.spyOn(projectsApi, 'getProject').mockReturnValue(
      new Promise(() => {}) as Promise<ProjectDetail>,
    )
    renderPage()
    expect(screen.getByTestId('table-skeleton')).toBeInTheDocument()
  })

  it('renders project name as heading', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    const heading = await screen.findByRole('heading', { name: 'my-app' })
    expect(heading).toBeInTheDocument()
  })

  it('renders back link to home page', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    const backLink = await screen.findByRole('link', { name: /back/i })
    expect(backLink).toHaveAttribute('href', '/')
  })

  it('defaults to Features tab', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })
    // Features tab should show feature rows
    expect(screen.getByText('Authentication')).toBeInTheDocument()
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })

  it('switches to Proposals tab on click', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })

    // Click Proposals tab
    await userEvent.click(screen.getByRole('tab', { name: /proposals/i }))

    // Should show proposals
    expect(screen.getByText('Auth Proposal')).toBeInTheDocument()
    expect(screen.getByText('API Proposal')).toBeInTheDocument()
  })

  it('feature name links to /features/:id/tasks', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    const featureLink = await screen.findByRole('link', {
      name: 'Authentication',
    })
    expect(featureLink).toHaveAttribute('href', '/features/20/tasks')
  })

  it('proposal title links to /proposals/:id', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })

    await userEvent.click(screen.getByRole('tab', { name: /proposals/i }))

    const proposalLink = screen.getByRole('link', { name: 'Auth Proposal' })
    expect(proposalLink).toHaveAttribute('href', '/proposals/10')
  })

  it('shows status badges for features', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })

    const badges = screen.getAllByTestId('status-badge')
    expect(badges.length).toBeGreaterThanOrEqual(2)
    const statuses = badges.map((b) => b.dataset.status)
    expect(statuses).toContain('in-progress')
    expect(statuses).toContain('done')
  })

  it('shows empty state when features list is empty', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue({
      ...mockProjectDetail,
      features: [],
    })
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })

    expect(screen.getByTestId('empty-state')).toHaveTextContent(
      '此项目暂无 Features',
    )
  })

  it('shows empty state when proposals list is empty', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue({
      ...mockProjectDetail,
      proposals: [],
    })
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })

    await userEvent.click(screen.getByRole('tab', { name: /proposals/i }))

    expect(screen.getByTestId('empty-state')).toHaveTextContent(
      '此项目暂无 Proposals',
    )
  })

  it('shows completion rate bars for features', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })

    const rates = screen.getAllByTestId('completion-rate')
    expect(rates.length).toBe(2)
    expect(rates.map((r) => r.textContent)).toEqual(
      expect.arrayContaining(['60%', '100%']),
    )
  })

  it('shows proposal feature count', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })

    await userEvent.click(screen.getByRole('tab', { name: /proposals/i }))

    // Proposal rows should show feature count
    expect(screen.getByText('2')).toBeInTheDocument() // Auth Proposal has 2 features
    expect(screen.getByText('3')).toBeInTheDocument() // API Proposal has 3 features
  })

  it('shows proposal creation date', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })

    await userEvent.click(screen.getByRole('tab', { name: /proposals/i }))

    const times = screen.getAllByTestId('relative-time')
    // Two proposals + two features (from initial render, though features may be hidden)
    const timeTexts = times.map((t) => t.textContent)
    expect(timeTexts).toContain('2026-01-10T10:00:00Z')
    expect(timeTexts).toContain('2026-02-15T12:00:00Z')
  })

  it('calls getProject with correct id', async () => {
    const spy = vi
      .spyOn(projectsApi, 'getProject')
      .mockResolvedValue(mockProjectDetail)
    renderPage('42')
    await screen.findByRole('heading', { name: 'my-app' })
    expect(spy).toHaveBeenCalledWith(42)
  })

  it('renders AppHeader with project name', async () => {
    vi.spyOn(projectsApi, 'getProject').mockResolvedValue(mockProjectDetail)
    renderPage()
    await screen.findByRole('heading', { name: 'my-app' })
    expect(screen.getByTestId('app-header')).toHaveTextContent('my-app')
  })
})
