import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { SearchInput } from './SearchInput'

describe('SearchInput', () => {
  it('renders an input with placeholder text', () => {
    render(<SearchInput value="" onChange={() => {}} />)
    expect(screen.getByPlaceholderText('Search projects...')).toBeInTheDocument()
  })

  it('displays the current value', () => {
    render(<SearchInput value="my-app" onChange={() => {}} />)
    const input = screen.getByPlaceholderText('Search projects...') as HTMLInputElement
    expect(input.value).toBe('my-app')
  })

  it('calls onChange when the user types', async () => {
    const onChange = vi.fn()
    render(<SearchInput value="" onChange={onChange} />)

    await userEvent.type(screen.getByPlaceholderText('Search projects...'), 'test')
    expect(onChange).toHaveBeenCalledTimes(4)
    expect(onChange).toHaveBeenLastCalledWith('t')
    // Each call passes the latest character since we simulate individual keypresses
    // The parent component should handle debouncing via useDebounce
  })

  it('calls onChange with empty string when cleared', async () => {
    const onChange = vi.fn()
    render(<SearchInput value="test" onChange={onChange} />)

    const input = screen.getByPlaceholderText('Search projects...')
    await userEvent.clear(input)
    expect(onChange).toHaveBeenCalledWith('')
  })
})
