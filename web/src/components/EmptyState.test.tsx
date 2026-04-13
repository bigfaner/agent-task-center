import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { EmptyState } from './EmptyState'

describe('EmptyState', () => {
  it('renders the title', () => {
    render(<EmptyState title="No items found" />)
    expect(screen.getByText('No items found')).toBeInTheDocument()
  })

  it('renders description when provided', () => {
    render(
      <EmptyState title="No items" description="Try adjusting your search" />,
    )
    expect(screen.getByText('Try adjusting your search')).toBeInTheDocument()
  })

  it('does not render description when not provided', () => {
    render(<EmptyState title="No items" />)
    expect(screen.queryByText('Try adjusting')).not.toBeInTheDocument()
  })

  it('renders action button when provided', () => {
    render(
      <EmptyState
        title="No items"
        action={{ label: 'Upload', onClick: vi.fn() }}
      />,
    )
    expect(
      screen.getByRole('button', { name: 'Upload' }),
    ).toBeInTheDocument()
  })

  it('does not render action button when not provided', () => {
    render(<EmptyState title="No items" />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('calls action.onClick when button is clicked', async () => {
    const onClick = vi.fn()
    render(<EmptyState title="No items" action={{ label: 'Upload', onClick }} />)
    await userEvent.click(screen.getByRole('button', { name: 'Upload' }))
    expect(onClick).toHaveBeenCalledOnce()
  })
})
