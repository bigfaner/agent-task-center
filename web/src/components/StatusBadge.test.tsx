import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { StatusBadge } from './StatusBadge'

describe('StatusBadge', () => {
  const featureStatuses = [
    { status: 'prd' as const, label: 'PRD', color: 'blue' },
    { status: 'design' as const, label: 'Design', color: 'purple' },
    { status: 'tasks' as const, label: 'Tasks', color: 'yellow' },
    { status: 'in-progress' as const, label: 'In Progress', color: 'orange' },
    { status: 'done' as const, label: 'Done', color: 'green' },
  ]

  const taskStatuses = [
    { status: 'pending' as const, label: 'Pending', color: 'gray' },
    { status: 'in_progress' as const, label: 'In Progress', color: 'orange' },
    { status: 'completed' as const, label: 'Completed', color: 'green' },
    { status: 'blocked' as const, label: 'Blocked', color: 'red' },
  ]

  it('renders feature statuses with correct labels', () => {
    for (const { status, label } of featureStatuses) {
      const { unmount } = render(<StatusBadge status={status} />)
      expect(screen.getByText(label)).toBeInTheDocument()
      unmount()
    }
  })

  it('renders task statuses with correct labels', () => {
    for (const { status, label } of taskStatuses) {
      const { unmount } = render(<StatusBadge status={status} />)
      expect(screen.getByText(label)).toBeInTheDocument()
      unmount()
    }
  })

  it('applies correct color classes for feature statuses', () => {
    for (const { status, color } of featureStatuses) {
      const { unmount } = render(<StatusBadge status={status} />)
      const badge = screen.getByText(
        status === 'prd'
          ? 'PRD'
          : status === 'design'
            ? 'Design'
            : status === 'tasks'
              ? 'Tasks'
              : status === 'in-progress'
                ? 'In Progress'
                : 'Done',
      )
      expect(badge.className).toContain(`bg-${color}-100`)
      expect(badge.className).toContain(`text-${color}-700`)
      unmount()
    }
  })

  it('applies correct color classes for task statuses', () => {
    for (const { status, color } of taskStatuses) {
      const { unmount } = render(<StatusBadge status={status} />)
      const label =
        status === 'pending'
          ? 'Pending'
          : status === 'in_progress'
            ? 'In Progress'
            : status === 'completed'
              ? 'Completed'
              : 'Blocked'
      const badge = screen.getByText(label)
      expect(badge.className).toContain(`bg-${color}-100`)
      expect(badge.className).toContain(`text-${color}-700`)
      unmount()
    }
  })

  it('applies rounded-full and badge classes', () => {
    render(<StatusBadge status="pending" />)
    const badge = screen.getByText('Pending')
    expect(badge.className).toContain('rounded-full')
    expect(badge.className).toContain('text-xs')
    expect(badge.className).toContain('font-medium')
  })

  it('falls back to gray for unknown status', () => {
    render(<StatusBadge status={'unknown' as never} />)
    const badge = screen.getByText('unknown')
    expect(badge.className).toContain('bg-gray-100')
    expect(badge.className).toContain('text-gray-700')
  })

  it('merges custom className', () => {
    render(<StatusBadge status="pending" className="extra-class" />)
    const badge = screen.getByText('Pending')
    expect(badge.className).toContain('extra-class')
  })
})
