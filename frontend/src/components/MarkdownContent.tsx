import { Children, isValidElement, type ReactNode } from 'react'
import ReactMarkdown, { type Components } from 'react-markdown'
import rehypeSanitize, { defaultSchema } from 'rehype-sanitize'
import remarkGfm from 'remark-gfm'
import { createHeadingSlugger } from '../utils/markdown'

function nodeText(node: ReactNode): string {
  return Children.toArray(node).map((child) => {
    if (typeof child === 'string' || typeof child === 'number') return String(child)
    if (isValidElement<{ children?: ReactNode }>(child)) return nodeText(child.props.children)
    return ''
  }).join('')
}

export function MarkdownContent({ content, className }: { content: string; className?: string }) {
  const slug = createHeadingSlugger()
  const heading = (level: 1 | 2 | 3) => ({ children, ...props }: React.HTMLAttributes<HTMLHeadingElement>) => {
    const id = slug(nodeText(children))
    const Tag = `h${level}` as const
    return <Tag id={id} {...props}>{children}</Tag>
  }
  const components: Components = { h1: heading(1), h2: heading(2), h3: heading(3) }

  // 显式 sanitize：不能永远依赖「没人启用 rehype-raw」这一约定来防 XSS，
  // 默认 schema 已覆盖 Markdown 可生成的全部元素；clobberPrefix 关闭以保留空 id
  const schema = { ...defaultSchema, clobber: [], attributes: { ...defaultSchema.attributes, '*': [...(defaultSchema.attributes?.['*'] ?? []), 'id'] } }

  return <div className={className}><ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[[rehypeSanitize, schema]]} components={components}>{content}</ReactMarkdown></div>
}
