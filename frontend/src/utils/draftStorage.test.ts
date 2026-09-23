import { beforeEach, describe, expect, it } from 'vitest'
import { clearPostDraft, hasDraftContent, loadPostDraft, savePostDraft } from './draftStorage'

describe('draft storage', () => {
  beforeEach(() => localStorage.clear())

  it('round-trips local drafts and detects meaningful content', () => {
    const draft = { title: '标题', slug: '', summary: '', content: '# 正文', coverImage: '', tags: [], status: 'draft' as const, featured: false }
    expect(hasDraftContent(draft)).toBe(true)
    savePostDraft('draft-key', draft, 123)
    expect(loadPostDraft('draft-key')).toEqual({ draft, savedAt: 123 })
    clearPostDraft('draft-key')
    expect(loadPostDraft('draft-key')).toBeUndefined()
  })
})
