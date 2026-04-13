import { useMemo, useCallback, type ComponentType, type HTMLAttributes, type ReactNode } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { AppHeader } from '@/components/AppHeader'
import { MarkdownRenderer } from '@/components/MarkdownRenderer'
import { ErrorState } from '@/components/ErrorState'
import { getProposalContent } from '@/api/docs'

// ── Types ──

interface TocItem {
  id: string
  text: string
  level: 2 | 3
}

// ── Helpers ──

function slugify(text: string): string {
  return text
    .toLowerCase()
    .replace(/[^\w\s-]/g, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .trim()
}

function extractTocItems(content: string): TocItem[] {
  const items: TocItem[] = []
  const headingRegex = /^(#{2,3})\s+(.+)$/gm
  let match
  while ((match = headingRegex.exec(content)) !== null) {
    const level = match[1].length as 2 | 3
    const text = match[2].trim()
    const id = slugify(text)
    items.push({ id, text, level })
  }
  return items
}

// ── Custom Markdown components with heading IDs ──

function createHeadingComponents(): Record<string, ComponentType<HTMLAttributes<HTMLElement> & { node?: unknown }>> {
  const makeHeading = (Tag: 'h2' | 'h3') => {
    const Heading = ({
      children,
      node,
      ...rest
    }: HTMLAttributes<HTMLHeadingElement> & { node?: unknown }) => {
      const text = extractTextFromChildren(children)
      const id = slugify(text)
      return (
        <Tag
          id={id}
          {...rest}
        >
          {children}
        </Tag>
      )
    }
    Heading.displayName = Tag === 'h2' ? 'Heading2' : 'Heading3'
    return Heading
  }

  return {
    h2: makeHeading('h2'),
    h3: makeHeading('h3'),
  }
}

function extractTextFromChildren(children: ReactNode): string {
  if (typeof children === 'string') return children
  if (Array.isArray(children)) return children.map(extractTextFromChildren).join('')
  if (children && typeof children === 'object' && 'props' in children) {
    return extractTextFromChildren((children as { props: { children: ReactNode } }).props.children)
  }
  return ''
}

// ── TableOfContents ──

function TableOfContents({ items }: { items: TocItem[] }) {
  const handleClick = useCallback((e: React.MouseEvent<HTMLAnchorElement>, id: string) => {
    e.preventDefault()
    const el = document.getElementById(id)
    if (el) {
      el.scrollIntoView({ behavior: 'smooth' })
    }
  }, [])

  if (items.length === 0) return null

  return (
    <nav data-testid="table-of-contents" className="sticky top-6">
      <h4 className="mb-2 text-sm font-semibold text-muted-foreground">Table of Contents</h4>
      <ul className="space-y-1 text-sm">
        {items.map((item) => (
          <li
            key={item.id}
            className={item.level === 3 ? 'pl-4' : ''}
          >
            <a
              data-testid="toc-item"
              href={`#${item.id}`}
              onClick={(e) => handleClick(e, item.id)}
              className="text-muted-foreground hover:text-foreground hover:underline"
            >
              {item.text}
            </a>
          </li>
        ))}
      </ul>
    </nav>
  )
}

// ── RelatedSection ──

function RelatedSection({
  features,
  tasks,
}: {
  features: { id: number; name: string; slug: string }[]
  tasks: { id: number; taskId: string; title: string }[]
}) {
  if (features.length === 0 && tasks.length === 0) return null

  return (
    <div data-testid="related-section" className="mt-8 border-t pt-6">
      <h3 className="mb-3 text-lg font-semibold">Related</h3>
      {features.length > 0 && (
        <div className="mb-3">
          <h4 className="mb-1 text-sm font-medium text-muted-foreground">Features</h4>
          <div className="flex flex-wrap gap-2">
            {features.map((f) => (
              <Link
                key={f.id}
                to={`/features/${f.id}/tasks`}
                className="rounded-md bg-secondary px-2.5 py-1 text-sm text-secondary-foreground hover:bg-secondary/80"
              >
                {f.name}
              </Link>
            ))}
          </div>
        </div>
      )}
      {tasks.length > 0 && (
        <div>
          <h4 className="mb-1 text-sm font-medium text-muted-foreground">Tasks</h4>
          <div className="flex flex-wrap gap-2">
            {tasks.map((t) => (
              <Link
                key={t.id}
                to={`/tasks/${t.id}`}
                className="rounded-md bg-secondary px-2.5 py-1 text-sm text-secondary-foreground hover:bg-secondary/80"
              >
                {t.taskId} {t.title}
              </Link>
            ))}
          </div>
        </div>
      )}
    </div>
  )
}

// ── Loading Spinner ──

function LoadingSpinner() {
  return (
    <div
      data-testid="doc-loading-spinner"
      className="flex items-center justify-center py-24"
    >
      <div className="h-8 w-8 animate-spin rounded-full border-4 border-muted border-t-primary" />
    </div>
  )
}

// ── Main Page ──

export default function DocViewerPage() {
  const { id } = useParams<{ id: string }>()
  const numericId = Number(id)

  const {
    data: doc,
    isLoading,
    isError,
    refetch,
  } = useQuery({
    queryKey: ['proposal-content', numericId],
    queryFn: () => getProposalContent(numericId),
  })

  const tocItems = useMemo(() => {
    if (!doc?.content) return []
    return extractTocItems(doc.content)
  }, [doc?.content])

  const headingComponents = useMemo(() => createHeadingComponents(), [])

  if (isLoading) {
    return (
      <div className="flex min-h-screen flex-col">
        <AppHeader showUpload={false} />
        <main className="flex-1 p-6">
          <LoadingSpinner />
        </main>
      </div>
    )
  }

  if (isError) {
    return (
      <div className="flex min-h-screen flex-col">
        <AppHeader showUpload={false} />
        <main className="flex-1 p-6">
          <Link
            to="/"
            className="text-sm text-muted-foreground hover:underline"
          >
            &larr; Back
          </Link>
          <ErrorState
            message="Failed to load document"
            onRetry={() => refetch()}
          />
        </main>
      </div>
    )
  }

  return (
    <div className="flex min-h-screen flex-col">
      <AppHeader showUpload={false} />
      <main className="flex-1 p-6">
        <Link
          to="/"
          className="text-sm text-muted-foreground hover:underline"
        >
          &larr; Back
        </Link>

        {doc && (
          <>
            <h1 className="mt-4 text-2xl font-bold">{doc.title}</h1>

            {/* Dual-column layout on desktop, single on mobile */}
            <div
              data-testid="doc-layout"
              className="mt-6 grid grid-cols-1 gap-8 lg:grid-cols-[1fr_250px]"
            >
              {/* Main content */}
              <div>
                <MarkdownRenderer
                  content={doc.content}
                  components={headingComponents}
                />
              </div>

              {/* TOC sidebar */}
              <aside className="hidden lg:block">
                <TableOfContents items={tocItems} />
              </aside>
            </div>

            {/* TOC on mobile (shown below content) */}
            <div className="mt-6 lg:hidden">
              <h4 className="mb-2 text-sm font-semibold text-muted-foreground">Table of Contents</h4>
              <ul className="space-y-1 text-sm">
                {tocItems.map((item) => (
                  <li
                    key={item.id}
                    className={item.level === 3 ? 'pl-4' : ''}
                  >
                    <a
                      data-testid="toc-item-mobile"
                      href={`#${item.id}`}
                      onClick={(e) => {
                        e.preventDefault()
                        const el = document.getElementById(item.id)
                        if (el) el.scrollIntoView({ behavior: 'smooth' })
                      }}
                      className="text-muted-foreground hover:text-foreground hover:underline"
                    >
                      {item.text}
                    </a>
                  </li>
                ))}
              </ul>
            </div>

            {/* Related section */}
            <RelatedSection
              features={doc.relatedFeatures}
              tasks={doc.relatedTasks}
            />
          </>
        )}
      </main>
    </div>
  )
}
