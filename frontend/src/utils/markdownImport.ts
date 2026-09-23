export type ImportedMarkdown = {
  content: string
  title: string
}

const MAX_MARKDOWN_SIZE = 2 * 1024 * 1024

export function isMarkdownFile(file: Pick<File, 'name' | 'type'>) {
  const name = file.name.toLowerCase()
  return name.endsWith('.md') || name.endsWith('.markdown') || file.type === 'text/markdown'
}

export function parseMarkdownDocument(content: string, filename: string): ImportedMarkdown {
  const normalized = content.replace(/^\uFEFF/, '')
  const heading = normalized.split('\n').find((line) => /^#\s+\S/.test(line.trim()))
  const fallback = filename.replace(/\.(?:md|markdown)$/i, '').trim()
  return {
    content: normalized,
    title: heading ? heading.trim().replace(/^#\s+/, '').replace(/\s+#+\s*$/, '').trim() : fallback,
  }
}

export async function readMarkdownFile(file: File) {
  if (!isMarkdownFile(file)) throw new Error('请选择 .md 或 .markdown 文件')
  if (file.size <= 0) throw new Error('Markdown 文件不能为空')
  if (file.size > MAX_MARKDOWN_SIZE) throw new Error('Markdown 文件不能超过 2MB')
  return parseMarkdownDocument(await file.text(), file.name)
}
