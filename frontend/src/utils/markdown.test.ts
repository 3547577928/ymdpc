import { describe, expect, it } from 'vitest'
import { extractMarkdownHeadings, headingBaseId } from './markdown'

describe('markdown helpers', () => {
  it('creates stable ids for headings and de-duplicates repeats', () => {
    expect(headingBaseId('设计与工程')).toBe('设计与工程')
    expect(extractMarkdownHeadings('# 设计与工程\n\n```md\n# ignored\n```\n## 设计与工程')).toEqual([
      { level: 1, text: '设计与工程', id: '设计与工程' },
      { level: 2, text: '设计与工程', id: '设计与工程-2' },
    ])
  })
})
