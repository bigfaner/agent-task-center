import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { BrowserRouter } from 'react-router-dom'
import { ProjectTable } from './ProjectTable'
import type { ProjectSummary } from '@/types'

// Mock RelativeTime to avoid date-related test flakiness
vi.mock('./RelativeTime', () => ({
  RelativeTime: ({ date }: { date: string }) => (
    <time data-testid="relative-time">{date}</time>
  ),
}))

// Mock CompletionRateBar
vi.mock('./CompletionRateBar', () => ({
  CompletionRateBar: ({ rate }: { rate: number }) => (
    <div data-testid="completion-rate">{rate}%</div>
  ),
}))

const mockProjects: ProjectSummary[] = [
  {
    id: 1,
    name: 'my-app',
    featureCount: 3,
    taskTotal: 24,
    completionRate: 75.0,
    updatedAt: '2026-04-12T14:30:00Z',
  },
  {
    id: 2,
    name: 'backend',
    featureCount: 1,
    taskTotal: 8,
    completionRate: 100,
    updatedAt: '2026-04-11T10:00:00Z',
  },
]

function renderWithRouter(ui: React.ReactElement) {
  return render(<BrowserRouter>{ui}</BrowserRouter>)
}

describe('ProjectTable', () => {
  it('renders table headers', () => {
    renderWithRouter(<ProjectTable projects={mockProjects} />)
    expect(screen.getByText('Project Name')).toBeInTheDocument()
    expect(screen.getByText('Features')).toBeInTheDocument()
    expect(screen.getByText('Tasks')).toBeInTheDocument()
    expect(screen.getByText('Completion')).toBeInTheDocument()
    expect(screen.getByText('Updated')).toBeInTheDocument()
  })

  it('renders a row for each project', () => {
    renderWithRouter(<ProjectTable projects={mockProjects} />)
    expect(screen.getByText('my-app')).toBeInTheDocument()
    expect(screen.getByText('backend')).toBeInTheDocument()
  })

  it('renders project name as a link', () => {
    renderWithRouter(<ProjectTable projects={mockProjects} />)
    const link = screen.getByRole('link', { name: 'my-app' })
    expect(link).toHaveAttribute('href', '/projects/1')
  })

  it('renders feature count and task total', () => {
    renderWithRouter(<ProjectTable projects={mockProjects} />)
    // First project: 3 features, 24 tasks
    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('24')).toBeInTheDocument()
  })

  it('renders completion rate bar', () => {
    renderWithRouter(<ProjectTable projects={mockProjects} />)
    const bars = screen.getAllByTestId('completion-rate')
    expect(bars).toHaveLength(2)
    expect(bars[0]).toHaveTextContent('75%')
    expect(bars[1]).toHaveTextContent('100%')
  })

  it('renders relative time', () => {
    renderWithRouter(<ProjectTable projects={mockProjects} />)
    const times = screen.getAllByTestId('relative-time')
    expect(times).toHaveLength(2)
    expect(times[0]).toHaveTextContent('2026-04-12T14:30:00Z')
  })

  it('renders empty table body when no projects', () => {
    renderWithRouter(<ProjectTable projects={[]} />)
    // Headers should still be there
    expect(screen.getByText('Project Name')).toBeInTheDocument()
    // But no project names
    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })
})
