import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { CompletionRateBar } from './CompletionRateBar'

describe('CompletionRateBar', () => {
  it('shows 0.0% when rate is 0', () => {
    render(<CompletionRateBar rate={0} />)
    expect(screen.getByText('0.0%')).toBeInTheDocument()
  })

  it('shows 100.0% when rate is 100', () => {
    render(<CompletionRateBar rate={100} />)
    expect(screen.getByText('100.0%')).toBeInTheDocument()
  })

  it('shows correct percentage for intermediate values', () => {
    render(<CompletionRateBar rate={75.5} />)
    expect(screen.getByText('75.5%')).toBeInTheDocument()
  })

  it('clamps values above 100 to 100', () => {
    render(<CompletionRateBar rate={150} />)
    expect(screen.getByText('100.0%')).toBeInTheDocument()
  })

  it('clamps negative values to 0', () => {
    render(<CompletionRateBar rate={-10} />)
    expect(screen.getByText('0.0%')).toBeInTheDocument()
  })

  it('renders a progress bar element', () => {
    const { container } = render(<CompletionRateBar rate={50} />)
    const bar = container.querySelector('[style*="width"]')
    expect(bar).toBeInTheDocument()
    expect(bar!.getAttribute('style')).toContain('width: 50%')
  })

  it('renders with 100% width for rate 100', () => {
    const { container } = render(<CompletionRateBar rate={100} />)
    const bar = container.querySelector('[style*="width"]')
    expect(bar!.getAttribute('style')).toContain('width: 100%')
  })

  it('renders with 0% width for rate 0', () => {
    const { container } = render(<CompletionRateBar rate={0} />)
    const bar = container.querySelector('[style*="width"]')
    expect(bar!.getAttribute('style')).toContain('width: 0%')
  })

  it('merges custom className', () => {
    const { container } = render(
      <CompletionRateBar rate={50} className="my-class" />,
    )
    expect(container.firstChild).toHaveClass('my-class')
  })
})
