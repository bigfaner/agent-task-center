import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { RelativeTime } from './RelativeTime'

describe('RelativeTime', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date('2026-04-14T12:00:00Z'))
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows "just now" for times within 60 seconds', () => {
    render(<RelativeTime date="2026-04-14T11:59:30Z" />)
    expect(screen.getByText('just now')).toBeInTheDocument()
  })

  it('shows minutes for times within 60 minutes', () => {
    render(<RelativeTime date="2026-04-14T11:30:00Z" />)
    expect(screen.getByText('30m ago')).toBeInTheDocument()
  })

  it('shows hours for times within 24 hours', () => {
    render(<RelativeTime date="2026-04-14T10:00:00Z" />)
    expect(screen.getByText('2h ago')).toBeInTheDocument()
  })

  it('shows days for times within 30 days', () => {
    render(<RelativeTime date="2026-04-12T12:00:00Z" />)
    expect(screen.getByText('2d ago')).toBeInTheDocument()
  })

  it('shows months for times beyond 30 days', () => {
    render(<RelativeTime date="2026-02-14T12:00:00Z" />)
    // Feb 14 to Apr 14 = 59 days = 1 month (floor(59/30) = 1)
    expect(screen.getByText('1mo ago')).toBeInTheDocument()
  })

  it('sets the dateTime attribute', () => {
    render(<RelativeTime date="2026-04-14T10:00:00Z" />)
    const timeEl = screen.getByText('2h ago')
    expect(timeEl.tagName).toBe('TIME')
    expect(timeEl.getAttribute('datetime')).toBe('2026-04-14T10:00:00Z')
  })

  it('shows full date in title attribute', () => {
    render(<RelativeTime date="2026-04-14T10:00:00Z" />)
    const timeEl = screen.getByText('2h ago')
    expect(timeEl.getAttribute('title')).toBeTruthy()
  })

  it('merges custom className', () => {
    render(<RelativeTime date="2026-04-14T10:00:00Z" className="custom" />)
    const timeEl = screen.getByText('2h ago')
    expect(timeEl.className).toContain('custom')
  })
})
