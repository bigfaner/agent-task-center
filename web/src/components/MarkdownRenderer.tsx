import { type ComponentType, type HTMLAttributes } from 'react'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

interface MarkdownRendererProps {
  content: string
  className?: string
  components?: Record<string, ComponentType<HTMLAttributes<HTMLElement> & { node?: unknown }>>
}

export function MarkdownRenderer({ content, className, components }: MarkdownRendererProps) {
  return (
    <div className={`prose prose-sm dark:prose-invert max-w-none ${className ?? ''}`}>
      <Markdown remarkPlugins={[remarkGfm]} components={components}>{content}</Markdown>
    </div>
  )
}
