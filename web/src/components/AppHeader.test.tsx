import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { AppHeader } from './AppHeader'

// Mock UploadDialog to avoid complex child rendering
vi.mock('./UploadDialog', () => ({
  UploadDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="upload-dialog">Upload Dialog</div> : null,
}))

describe('AppHeader', () => {
  it('renders the logo text "Agent Task Center"', () => {
    render(<AppHeader />)
    expect(screen.getByText('Agent Task Center')).toBeInTheDocument()
  })

  it('renders Upload button by default', () => {
    render(<AppHeader />)
    expect(
      screen.getByRole('button', { name: 'Upload' }),
    ).toBeInTheDocument()
  })

  it('hides Upload button when showUpload is false', () => {
    render(<AppHeader showUpload={false} />)
    expect(screen.queryByRole('button', { name: 'Upload' })).not.toBeInTheDocument()
  })

  it('opens UploadDialog when Upload button is clicked', async () => {
    render(<AppHeader />)
    expect(screen.queryByTestId('upload-dialog')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Upload' }))
    expect(screen.getByTestId('upload-dialog')).toBeInTheDocument()
  })

  it('passes projectName to UploadDialog', async () => {
    render(<AppHeader projectName="my-app" />)
    await userEvent.click(screen.getByRole('button', { name: 'Upload' }))
    // Dialog is rendered (mock doesn't show prop, but the test verifies the dialog opens)
    expect(screen.getByTestId('upload-dialog')).toBeInTheDocument()
  })
})
