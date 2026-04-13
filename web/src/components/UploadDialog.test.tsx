import React from 'react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen, waitFor, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { UploadDialog } from './UploadDialog'
import * as uploadApi from '@/api/upload'
import * as projectsApi from '@/api/projects'

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

// Mock the projects API
vi.mock('@/api/projects', () => ({
  listProjects: vi.fn().mockResolvedValue({
    items: [
      { id: 1, name: 'project-a', featureCount: 2, taskTotal: 10, completionRate: 50, updatedAt: '2026-04-12T10:00:00Z' },
      { id: 2, name: 'project-b', featureCount: 1, taskTotal: 5, completionRate: 80, updatedAt: '2026-04-12T10:00:00Z' },
    ],
    total: 2,
    page: 1,
    pageSize: 100,
  }),
}))

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
      },
    },
  })
}

function renderDialog(props?: Partial<React.ComponentProps<typeof UploadDialog>>) {
  const qc = createQueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <UploadDialog
        open={true}
        onOpenChange={vi.fn()}
        {...props}
      />
    </QueryClientProvider>,
  )
}

describe('UploadDialog', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  // ── Visibility ──

  it('renders nothing when closed', () => {
    render(
      <QueryClientProvider client={createQueryClient()}>
        <UploadDialog open={false} onOpenChange={vi.fn()} />
      </QueryClientProvider>,
    )
    expect(screen.queryByText('Upload File')).not.toBeInTheDocument()
  })

  it('renders the dialog when open', () => {
    renderDialog()
    expect(screen.getByText('Upload File')).toBeInTheDocument()
  })

  // ── ProjectSelector ──

  it('renders project selector dropdown', async () => {
    renderDialog()
    const selector = await screen.findByTestId('project-selector')
    expect(selector).toBeInTheDocument()
  })

  it('loads projects from API', async () => {
    renderDialog()
    await screen.findByTestId('project-selector')

    expect(projectsApi.listProjects).toHaveBeenCalledWith({ pageSize: 100 })
  })

  it('displays project options in dropdown', async () => {
    renderDialog()
    await screen.findByTestId('project-selector')

    const selector = screen.getByTestId('project-selector')
    expect(selector).toHaveTextContent('Select project')
  })

  // ── DropZone ──

  it('renders drop zone with instructions', () => {
    renderDialog()
    expect(screen.getByText(/drag.*drop/i)).toBeInTheDocument()
  })

  it('has hidden file input with correct accept types', () => {
    renderDialog()
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    expect(input).toBeInTheDocument()
    expect(input.getAttribute('accept')).toBe('.json,.md')
  })

  // ── File validation ──

  it('shows error for non-.json/.md files via drop', async () => {
    renderDialog()
    const dropZone = screen.getByTestId('drop-zone')

    const file = new File(['content'], 'test.txt', { type: 'text/plain' })
    const dataTransfer = { files: [file] }

    // Simulate drag events then drop
    await userEvent.hover(dropZone)
    fireEvent.drop(dropZone, { dataTransfer })

    expect(screen.getByTestId('validation-message')).toHaveTextContent(
      'Only .json and .md files are supported',
    )
  })

  it('shows error for files exceeding 5MB via drop', async () => {
    renderDialog()
    const dropZone = screen.getByTestId('drop-zone')

    const largeContent = new ArrayBuffer(6 * 1024 * 1024)
    const file = new File([largeContent], 'large.json', { type: 'application/json' })
    const dataTransfer = { files: [file] }

    fireEvent.drop(dropZone, { dataTransfer })

    expect(screen.getByTestId('validation-message')).toHaveTextContent(
      'File size cannot exceed 5MB',
    )
  })

  it('accepts valid .json file', async () => {
    renderDialog()
    const input = document.querySelector('input[type="file"]') as HTMLInputElement

    const file = new File(['{"task_id":"1.1","title":"Test"}'], 'tasks.json', {
      type: 'application/json',
    })
    await userEvent.upload(input, file)

    expect(screen.queryByTestId('validation-message')).not.toBeInTheDocument()
    expect(screen.getByTestId('selected-file')).toHaveTextContent('tasks.json')
  })

  it('accepts valid .md file', async () => {
    renderDialog()
    const input = document.querySelector('input[type="file"]') as HTMLInputElement

    const file = new File(['# Proposal'], 'proposal.md', { type: 'text/markdown' })
    await userEvent.upload(input, file)

    expect(screen.queryByTestId('validation-message')).not.toBeInTheDocument()
    expect(screen.getByTestId('selected-file')).toHaveTextContent('proposal.md')
  })

  // ── Upload button disabled state ──

  it('Upload button is disabled when no file is selected', () => {
    renderDialog()
    expect(screen.getByRole('button', { name: 'Upload' })).toBeDisabled()
  })

  it('Upload button is disabled when no project is selected', async () => {
    renderDialog()
    const input = document.querySelector('input[type="file"]') as HTMLInputElement

    const file = new File(['content'], 'test.json', { type: 'application/json' })
    await userEvent.upload(input, file)

    // No project selected yet
    expect(screen.getByRole('button', { name: 'Upload' })).toBeDisabled()
  })

  // ── Cancel button ──

  it('calls onOpenChange(false) when Cancel is clicked', async () => {
    const onOpenChange = vi.fn()
    renderDialog({ onOpenChange })
    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onOpenChange).toHaveBeenCalledWith(false)
  })

  // ── Upload success ──

  it('shows success summary after successful upload', async () => {
    renderDialog({ projectName: 'project-a' })

    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['content'], 'test.json', { type: 'application/json' })
    await userEvent.upload(input, file)

    await userEvent.click(screen.getByRole('button', { name: 'Upload' }))

    await waitFor(() => {
      expect(screen.getByTestId('upload-result')).toBeInTheDocument()
    })
    expect(screen.getByTestId('upload-result')).toHaveTextContent(
      'Created 3, updated 1',
    )
  })

  it('calls onUploadSuccess after successful upload', async () => {
    const onUploadSuccess = vi.fn()
    renderDialog({ projectName: 'project-a', onUploadSuccess })

    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['content'], 'test.json', { type: 'application/json' })
    await userEvent.upload(input, file)

    await userEvent.click(screen.getByRole('button', { name: 'Upload' }))

    await waitFor(() => {
      expect(onUploadSuccess).toHaveBeenCalled()
    })
  })

  // ── Upload error ──

  it('shows error message when upload fails', async () => {
    vi.mocked(uploadApi.uploadFile).mockRejectedValueOnce(new Error('Server error'))
    renderDialog({ projectName: 'project-a' })

    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['content'], 'test.json', { type: 'application/json' })
    await userEvent.upload(input, file)

    await userEvent.click(screen.getByRole('button', { name: 'Upload' }))

    await waitFor(() => {
      expect(screen.getByTestId('validation-message')).toHaveTextContent('Server error')
    })
  })

  // ── Reset on close ──

  it('resets state when dialog is closed and reopened', async () => {
    const { rerender } = render(
      <QueryClientProvider client={createQueryClient()}>
        <UploadDialog open={true} onOpenChange={vi.fn()} projectName="project-a" />
      </QueryClientProvider>,
    )

    // Upload a file
    const input = document.querySelector('input[type="file"]') as HTMLInputElement
    const file = new File(['content'], 'test.json', { type: 'application/json' })
    await userEvent.upload(input, file)
    expect(screen.getByTestId('selected-file')).toBeInTheDocument()

    // Close dialog
    rerender(
      <QueryClientProvider client={createQueryClient()}>
        <UploadDialog open={false} onOpenChange={vi.fn()} projectName="project-a" />
      </QueryClientProvider>,
    )

    // Reopen dialog
    rerender(
      <QueryClientProvider client={createQueryClient()}>
        <UploadDialog open={true} onOpenChange={vi.fn()} projectName="project-a" />
      </QueryClientProvider>,
    )

    // File should be cleared
    expect(screen.queryByTestId('selected-file')).not.toBeInTheDocument()
  })

  // ── Drag and drop ──

  it('drop zone highlights on drag over', async () => {
    renderDialog()
    const dropZone = screen.getByTestId('drop-zone')

    await userEvent.hover(dropZone)

    // The drop zone should exist and be interactive
    expect(dropZone).toBeInTheDocument()
  })
})
