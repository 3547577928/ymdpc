export type ImportedMarkdown = {
  content: string
  title: string
  summary: string
}

const MAX_MARKDOWN_SIZE = 2 * 1024 * 1024

export function isMarkdownFile(file: Pick<File, 'name' | 'type'>) {
  const name = file.name.toLowerCase()
  return name.endsWith('.md') || name.endsWith('.markdown') || file.type === 'text/markdown'
}

export function parseMarkdownDocument(content: string, filename: string): ImportedMarkdown {
  const normalized = content.replace(/^\uFEFF/, '')
  const lines = normalized.split('\n')
  const heading = lines.find((line) => /^#\s+\S/.test(line.trim()))
  const fallback = filename.replace(/\.(?:md|markdown)$/i, '').trim()
  const summary = lines
    .map((line) => line.trim())
    .find((line) => line && !line.startsWith('#') && !line.startsWith('```') && !line.startsWith('>') && !line.startsWith('- ') && !line.startsWith('* ') && !line.startsWith('!['))
    ?.replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/[*_`~]/g, '')
    .trim()
    .slice(0, 180) ?? ''
  return {
    content: normalized,
    title: heading ? heading.trim().replace(/^#\s+/, '').replace(/\s+#+\s*$/, '').trim() : fallback,
    summary,
  }
}

export async function readMarkdownFile(file: File) {
  if (!isMarkdownFile(file)) throw new Error('请选择 .md 或 .markdown 文件')
  if (file.size <= 0) throw new Error('Markdown 文件不能为空')
  if (file.size > MAX_MARKDOWN_SIZE) throw new Error('Markdown 文件不能超过 2MB')
  return parseMarkdownDocument(await file.text(), file.name)
}
