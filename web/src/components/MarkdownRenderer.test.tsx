import { describe, it, expect } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MarkdownRenderer } from './MarkdownRenderer'

describe('MarkdownRenderer', () => {
  it('renders plain text markdown', () => {
    render(<MarkdownRenderer content="Hello world" />)
    expect(screen.getByText('Hello world')).toBeInTheDocument()
  })

  it('renders headings', () => {
    render(<MarkdownRenderer content="# Title" />)
    const heading = screen.getByRole('heading', { level: 1 })
    expect(heading).toHaveTextContent('Title')
  })

  it('renders bold text', () => {
    render(<MarkdownRenderer content="This is **bold** text" />)
    const strong = screen.getByText('bold')
    expect(strong.tagName).toBe('STRONG')
  })

  it('renders italic text', () => {
    render(<MarkdownRenderer content="This is *italic* text" />)
    const em = screen.getByText('italic')
    expect(em.tagName).toBe('EM')
  })

  it('renders code blocks', () => {
    render(<MarkdownRenderer content="```js\nconsole.log('hi')\n```" />)
    const code = screen.getByText(/console\.log/)
    expect(code).toBeInTheDocument()
  })

  it('renders inline code', () => {
    render(<MarkdownRenderer content="Use `npm install` to setup" />)
    const code = screen.getByText('npm install')
    expect(code.tagName).toBe('CODE')
  })

  it('renders GFM tables', () => {
    const tableMd = `| Name | Age |
|------|-----|
| Alice | 30 |
| Bob | 25 |`
    render(<MarkdownRenderer content={tableMd} />)
    expect(screen.getByText('Alice')).toBeInTheDocument()
    expect(screen.getByText('Bob')).toBeInTheDocument()
    expect(screen.getByText('Name')).toBeInTheDocument()
    expect(screen.getByText('Age')).toBeInTheDocument()
  })

  it('renders links', () => {
    render(<MarkdownRenderer content="[Go to Google](https://google.com)" />)
    const link = screen.getByRole('link', { name: 'Go to Google' })
    expect(link).toHaveAttribute('href', 'https://google.com')
  })

  it('renders unordered lists', () => {
    const { container } = render(
      <MarkdownRenderer
        content={`- item 1
- item 2
- item 3`}
      />,
    )
    const ul = container.querySelector('ul')
    expect(ul).toBeInTheDocument()
    const listItems = ul!.querySelectorAll('li')
    expect(listItems.length).toBe(3)
    expect(listItems[0]).toHaveTextContent('item 1')
    expect(listItems[1]).toHaveTextContent('item 2')
    expect(listItems[2]).toHaveTextContent('item 3')
  })

  it('applies prose class for styling', () => {
    const { container } = render(
      <MarkdownRenderer content="Hello" />,
    )
    expect(container.firstChild).toHaveClass('prose')
  })

  it('applies custom className', () => {
    const { container } = render(
      <MarkdownRenderer content="Hello" className="my-class" />,
    )
    expect(container.firstChild).toHaveClass('my-class')
  })
})
