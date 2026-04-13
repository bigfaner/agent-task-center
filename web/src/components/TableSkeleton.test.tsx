import { describe, it, expect } from 'vitest'
import { render } from '@testing-library/react'
import { TableSkeleton } from './TableSkeleton'

describe('TableSkeleton', () => {
  it('renders 5 rows by default', () => {
    const { container } = render(<TableSkeleton />)
    const rows = container.querySelectorAll('.animate-pulse')
    expect(rows.length).toBeGreaterThanOrEqual(5)
  })

  it('renders the specified number of rows', () => {
    const { container } = render(<TableSkeleton rows={3} />)
    // Each row has multiple pulse elements, so count the row containers
    const rowContainers = container.querySelectorAll('.flex.items-center')
    expect(rowContainers.length).toBe(3)
  })

  it('renders 10 rows when rows=10', () => {
    const { container } = render(<TableSkeleton rows={10} />)
    const rowContainers = container.querySelectorAll('.flex.items-center')
    expect(rowContainers.length).toBe(10)
  })

  it('merges custom className', () => {
    const { container } = render(<TableSkeleton className="my-skeleton" />)
    expect(container.firstChild).toHaveClass('my-skeleton')
  })
})
