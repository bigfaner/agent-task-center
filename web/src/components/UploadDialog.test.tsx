import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { UploadDialog } from './UploadDialog'

// Mock the upload API
vi.mock('@/api/upload', () => ({
  uploadFile: vi.fn().mockResolvedValue({
    filename: 'test.json',
    created: 3,
    updated: 1,
    skipped: 0,
    message: 'Created 3, updated 1',
  }),
}))

describe('UploadDialog', () => {
  it('renders nothing when closed', () => {
    render(<UploadDialog open={false} onOpenChange={vi.fn()} />)
    expect(screen.queryByText('Upload File')).not.toBeInTheDocument()
  })

  it('renders the dialog when open', () => {
    render(<UploadDialog open={true} onOpenChange={vi.fn()} />)
    expect(screen.getByText('Upload File')).toBeInTheDocument()
  })

  it('renders Cancel and Upload buttons', () => {
    render(<UploadDialog open={true} onOpenChange={vi.fn()} />)
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Upload' })).toBeInTheDocument()
  })

  it('Upload button is disabled when no file is selected', () => {
    render(
      <UploadDialog open={true} onOpenChange={vi.fn()} projectName="test" />,
    )
    expect(screen.getByRole('button', { name: 'Upload' })).toBeDisabled()
  })

  it('calls onOpenChange(false) when Cancel is clicked', async () => {
    const onOpenChange = vi.fn()
    render(<UploadDialog open={true} onOpenChange={onOpenChange} />)
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  it('renders file input with correct accept types', () => {
    render(<UploadDialog open={true} onOpenChange={vi.fn()} />)
    const input = document.querySelector('input[type="file"]')
    expect(input).toBeInTheDocument()
    expect(input!.getAttribute('accept')).toBe('.json,.md')
  })
})
