import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { PriorityBadge } from './PriorityBadge'

describe('PriorityBadge', () => {
  it('renders P0 with red color', () => {
    render(<PriorityBadge priority="P0" />)
    const badge = screen.getByText('P0')
    expect(badge).toBeInTheDocument()
    expect(badge.className).toContain('bg-red-100')
    expect(badge.className).toContain('text-red-700')
  })

  it('renders P1 with orange color', () => {
    render(<PriorityBadge priority="P1" />)
    const badge = screen.getByText('P1')
    expect(badge).toBeInTheDocument()
    expect(badge.className).toContain('bg-orange-100')
    expect(badge.className).toContain('text-orange-700')
  })

  it('renders P2 with blue color', () => {
    render(<PriorityBadge priority="P2" />)
    const badge = screen.getByText('P2')
    expect(badge).toBeInTheDocument()
    expect(badge.className).toContain('bg-blue-100')
    expect(badge.className).toContain('text-blue-700')
  })

  it('applies badge styling classes', () => {
    render(<PriorityBadge priority="P0" />)
    const badge = screen.getByText('P0')
    expect(badge.className).toContain('rounded-full')
    expect(badge.className).toContain('text-xs')
    expect(badge.className).toContain('font-medium')
  })

  it('merges custom className', () => {
    render(<PriorityBadge priority="P1" className="custom-cls" />)
    const badge = screen.getByText('P1')
    expect(badge.className).toContain('custom-cls')
  })
})
