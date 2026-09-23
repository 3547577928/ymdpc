import { describe, expect, it } from 'vitest'
import { isMarkdownFile, parseMarkdownDocument } from './markdownImport'

describe('markdown import helpers', () => {
  it('accepts markdown extensions and extracts the first h1 title', () => {
    expect(isMarkdownFile({ name: 'note.md', type: '' })).toBe(true)
    expect(isMarkdownFile({ name: 'note.txt', type: 'text/plain' })).toBe(false)
    expect(parseMarkdownDocument('\uFEFF# 导入标题\n\n正文', 'fallback.md')).toEqual({ content: '# 导入标题\n\n正文', title: '导入标题' })
  })

  it('uses the filename when the document has no h1', () => {
    expect(parseMarkdownDocument('正文', 'project-notes.markdown').title).toBe('project-notes')
  })
})
