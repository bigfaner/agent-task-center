import React from 'react'
import { describe, it, expect, vi, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Route, Routes } from 'react-router-dom'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import DocViewerPage from './DocViewerPage'
import * as docsApi from '@/api/docs'
import type { DocumentContent } from '@/types'

// Mock child components that are tested separately
vi.mock('@/components/AppHeader', () => ({
  AppHeader: ({
    showUpload,
  }: {
    showUpload?: boolean
    onUploadSuccess?: () => void
  }) => (
    <header data-testid="app-header">
      <span data-testid="show-upload">{String(showUpload)}</span>
    </header>
  ),
}))

vi.mock('@/components/MarkdownRenderer', () => ({
  MarkdownRenderer: ({
    content,
    components,
  }: {
    content: string
    components?: Record<string, unknown>
  }) => {
    if (!content) return null

    // Simulate heading ID injection using the components prop
    const h2Comp = components?.h2 as React.ComponentType<
      React.HTMLAttributes<HTMLHeadingElement> & { node?: unknown }
    > | undefined
    const h3Comp = components?.h3 as React.ComponentType<
      React.HTMLAttributes<HTMLHeadingElement> & { node?: unknown }
    > | undefined

    // Parse headings from the markdown content for testing
    const lines = content.split('\n')
    const elements: React.ReactNode[] = []
    let key = 0
    for (const line of lines) {
      if (line.startsWith('## ')) {
        const text = line.replace('## ', '')
        if (h2Comp) {
          elements.push(
            React.createElement(h2Comp, { key: key++, node: {} }, text),
          )
        } else {
          elements.push(React.createElement('h2', { key: key++ }, text))
        }
      } else if (line.startsWith('### ')) {
        const text = line.replace('### ', '')
        if (h3Comp) {
          elements.push(
            React.createElement(h3Comp, { key: key++, node: {} }, text),
          )
        } else {
          elements.push(React.createElement('h3', { key: key++ }, text))
        }
      } else if (line.trim()) {
        elements.push(React.createElement('p', { key: key++ }, line))
      }
    }
    return React.createElement('div', { 'data-testid': 'markdown-renderer' }, ...elements)
  },
}))

vi.mock('@/components/ErrorState', () => ({
  ErrorState: ({
    message,
    onRetry,
  }: {
    message?: string
    onRetry?: () => void
  }) => (
    <div data-testid="error-state">
      <span>{message}</span>
      {onRetry && (
        <button data-testid="retry-btn" onClick={onRetry}>
          Retry
        </button>
      )}
    </div>
  ),
}))

// ── Mock Data ──

const mockDocContent: DocumentContent = {
  title: 'Auth System Proposal',
  content:
    '## Overview\n\nThis is the overview.\n\n## Goals\n\nThe main goals.\n\n### Security\n\nSecurity goals here.\n\n## Design\n\nDesign details.',
  relatedFeatures: [
    { id: 1, name: 'Auth', slug: 'auth' },
    { id: 2, name: 'Dashboard', slug: 'dashboard' },
  ],
  relatedTasks: [],
}

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

function renderPage(proposalId = '1') {
  const qc = createQueryClient()
  return render(
    <QueryClientProvider client={qc}>
      <MemoryRouter initialEntries={[`/proposals/${proposalId}`]}>
        <Routes>
          <Route path="/proposals/:id" element={<DocViewerPage />} />
          <Route
            path="/projects/:id"
            element={<div data-testid="navigated-project" />}
          />
          <Route
            path="/features/:id/tasks"
            element={<div data-testid="navigated-feature" />}
          />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  )
}

describe('DocViewerPage', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  // ── Loading state ──

  it('shows loading spinner on initial load', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockReturnValue(
      new Promise(() => {}) as Promise<DocumentContent>,
    )
    renderPage()

    expect(screen.getByTestId('doc-loading-spinner')).toBeInTheDocument()
  })

  // ── Document title ──

  it('renders document title as heading', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    const heading = await screen.findByRole('heading', { level: 1 })
    expect(heading).toHaveTextContent('Auth System Proposal')
  })

  // ── Back link ──

  it('renders back link', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    const backLink = await screen.findByRole('link', { name: /back/i })
    expect(backLink).toBeInTheDocument()
  })

  // ── Markdown content rendering ──

  it('renders markdown content via MarkdownRenderer', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    const md = await screen.findByTestId('markdown-renderer')
    expect(md).toBeInTheDocument()
    expect(md).toHaveTextContent('Overview')
    expect(md).toHaveTextContent('Goals')
  })

  // ── Table of Contents ──

  it('renders table of contents with H2 headings', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    await screen.findByRole('heading', { level: 1 })

    const toc = screen.getByTestId('table-of-contents')
    expect(toc).toBeInTheDocument()
    expect(toc).toHaveTextContent('Overview')
    expect(toc).toHaveTextContent('Goals')
    expect(toc).toHaveTextContent('Design')
  })

  it('renders H3 headings in table of contents', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    await screen.findByRole('heading', { level: 1 })

    const toc = screen.getByTestId('table-of-contents')
    expect(toc).toHaveTextContent('Security')
  })

  it('TOC items are anchor links with correct IDs', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    await screen.findByRole('heading', { level: 1 })

    const tocLinks = screen.getAllByTestId('toc-item')
    expect(tocLinks.length).toBeGreaterThan(0)

    // H2 items
    const overviewLink = tocLinks.find((el) =>
      el.textContent?.includes('Overview'),
    )
    expect(overviewLink).toBeInTheDocument()
    expect(overviewLink!.tagName).toBe('A')
  })

  it('TOC items have correct href anchors', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    await screen.findByRole('heading', { level: 1 })

    const toc = screen.getByTestId('table-of-contents')
    // The anchor hrefs should start with #
    const links = toc.querySelectorAll('a')
    for (const link of links) {
      expect(link.getAttribute('href')).toMatch(/^#/)
    }
  })

  // ── Dual-column layout ──

  it('renders dual-column layout with content and TOC', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    await screen.findByRole('heading', { level: 1 })

    const docLayout = screen.getByTestId('doc-layout')
    expect(docLayout).toBeInTheDocument()

    // Should contain both markdown content and TOC
    expect(screen.getByTestId('markdown-renderer')).toBeInTheDocument()
    expect(screen.getByTestId('table-of-contents')).toBeInTheDocument()
  })

  // ── Related features ──

  it('renders related features section', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    await screen.findByRole('heading', { level: 1 })

    expect(screen.getByTestId('related-section')).toBeInTheDocument()
    expect(screen.getByText('Auth')).toBeInTheDocument()
    expect(screen.getByText('Dashboard')).toBeInTheDocument()
  })

  it('related feature links navigate to /features/:id/tasks', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(mockDocContent)
    renderPage()

    await screen.findByRole('heading', { level: 1 })

    const authLink = screen.getByRole('link', { name: 'Auth' })
    expect(authLink).toHaveAttribute('href', '/features/1/tasks')

    const dashLink = screen.getByRole('link', { name: 'Dashboard' })
    expect(dashLink).toHaveAttribute('href', '/features/2/tasks')
  })

  it('hides related section when no related features or tasks', async () => {
    const docNoRelated: DocumentContent = {
      title: 'Standalone Doc',
      content: '## Intro\n\nSome content.',
      relatedFeatures: [],
      relatedTasks: [],
    }
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(docNoRelated)
    renderPage()

    await screen.findByRole('heading', { level: 1 })

    expect(screen.queryByTestId('related-section')).not.toBeInTheDocument()
  })

  // ── Error state ──

  it('shows error state when document fetch fails', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockRejectedValue(
      new Error('Network error'),
    )
    renderPage()

    const errorState = await screen.findByTestId('error-state')
    expect(errorState).toBeInTheDocument()
    expect(errorState).toHaveTextContent('Failed to load document')
  })

  it('error state includes back link', async () => {
    vi.spyOn(docsApi, 'getProposalContent').mockRejectedValue(
      new Error('Network error'),
    )
    renderPage()

    await screen.findByTestId('error-state')

    const backLink = screen.getByRole('link', { name: /back/i })
    expect(backLink).toBeInTheDocument()
  })

  // ── API calls ──

  it('calls getProposalContent with correct id', async () => {
    const docSpy = vi
      .spyOn(docsApi, 'getProposalContent')
      .mockResolvedValue(mockDocContent)
    renderPage('42')
    await screen.findByRole('heading', { level: 1 })

    expect(docSpy).toHaveBeenCalledWith(42)
  })

  // ── Empty content ──

  it('handles empty content gracefully', async () => {
    const emptyDoc: DocumentContent = {
      title: 'Empty Proposal',
      content: '',
      relatedFeatures: [],
      relatedTasks: [],
    }
    vi.spyOn(docsApi, 'getProposalContent').mockResolvedValue(emptyDoc)
    renderPage()

    const heading = await screen.findByRole('heading', { level: 1 })
    expect(heading).toHaveTextContent('Empty Proposal')

    // TOC should be empty or not render items
    const toc = screen.queryByTestId('table-of-contents')
    if (toc) {
      const tocItems = toc.querySelectorAll('[data-testid="toc-item"]')
      expect(tocItems.length).toBe(0)
    }
  })
})
