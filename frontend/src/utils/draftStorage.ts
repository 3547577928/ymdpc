import type { PostInput } from '../services/api'

export type StoredPostDraft = { draft: PostInput; savedAt: number }

export function loadPostDraft(key: string): StoredPostDraft | undefined {
  try {
    const raw = window.localStorage.getItem(key)
    if (!raw) return undefined
    const stored = JSON.parse(raw) as StoredPostDraft
    if (!stored?.draft || typeof stored.savedAt !== 'number') return undefined
    return stored
  } catch {
    return undefined
  }
}

export function savePostDraft(key: string, draft: PostInput, savedAt = Date.now()) {
  try {
    window.localStorage.setItem(key, JSON.stringify({ draft, savedAt } satisfies StoredPostDraft))
  } catch {
    // localStorage 配额满（~5 MB）或隐私模式不可用；存不了但丢弃总比崩溃好
  }
}

export function clearPostDraft(key: string) {
  window.localStorage.removeItem(key)
}

export function hasDraftContent(draft: PostInput) {
  return Boolean(draft.title.trim() || draft.summary.trim() || draft.content.trim())
}
