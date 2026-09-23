import { ArrowLeft, CalendarClock, Eye, FileClock, History, PenLine, RotateCcw, Save, Send } from 'lucide-react'
import type { FormEvent } from 'react'
import { useEffect, useMemo, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { MarkdownContent } from '../components/MarkdownContent'
import { ImageUploadField } from '../components/ImageUploadField'
import { ConfirmDialog } from '../components/Dialog'
import { createUserPost, getCategories, getCurrentUser, getPostById, getPostRevisions, restorePostRevision, updateUserPost, type PostInput } from '../services/api'
import type { Category, Post, PostRevision } from '../types'
import { clearPostDraft, hasDraftContent, loadPostDraft, savePostDraft, type StoredPostDraft } from '../utils/draftStorage'

const initialDraft: PostInput = { title: '', slug: '', summary: '', content: '', coverImage: '', tags: [], status: 'draft', featured: false, scheduledAt: null }

function dateTimeLocalValue(value?: string | null) {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Date(date.getTime() - date.getTimezoneOffset() * 60_000).toISOString().slice(0, 16)
}

export function WritePage() {
  const navigate = useNavigate()
  const { id } = useParams()
  const editingID = id ? Number(id) : undefined
  const storageKey = useMemo(() => `qs:editor-draft:${editingID ?? 'new'}`, [editingID])
  const [draft, setDraft] = useState<PostInput>(initialDraft)
  const [categories, setCategories] = useState<Category[]>([])
  const [saving, setSaving] = useState(false)
  const [loading, setLoading] = useState(Boolean(editingID))
  const [error, setError] = useState('')
  const [checking, setChecking] = useState(true)
  const [mode, setMode] = useState<'write' | 'preview'>('write')
  const [autosaveReady, setAutosaveReady] = useState(false)
  const [autosaveText, setAutosaveText] = useState('尚未自动保存')
  const [recovery, setRecovery] = useState<StoredPostDraft>()
  const [revisions, setRevisions] = useState<PostRevision[]>([])
  const [revisionToRestore, setRevisionToRestore] = useState<PostRevision>()

  useEffect(() => {
    getCategories().then(setCategories).catch(() => undefined)
  }, [])

  useEffect(() => {
    getCurrentUser().catch(() => navigate('/login')).finally(() => setChecking(false))
  }, [navigate])

  useEffect(() => {
    let active = true
    const local = loadPostDraft(storageKey)
    if (!editingID) {
      if (local && hasDraftContent(local.draft)) setRecovery(local)
      setAutosaveReady(true)
      setLoading(false)
      return () => { active = false }
    }
    getPostById(editingID)
      .then((post) => {
        if (!active) return
        const serverDraft: PostInput = { title: post.title, slug: post.slug, summary: post.summary, content: post.content, coverImage: post.coverImage, tags: post.tags, status: post.status, featured: post.featured, categoryId: post.category?.id ?? null, scheduledAt: post.scheduledAt ?? null }
        setDraft(serverDraft)
        const serverUpdatedAt = post.updatedAt ? new Date(post.updatedAt).getTime() : 0
        if (local && local.savedAt > serverUpdatedAt && hasDraftContent(local.draft)) setRecovery(local)
        void getPostRevisions(editingID).then(setRevisions).catch(() => undefined)
      })
      .catch((reason: Error) => setError(reason.message))
      .finally(() => {
        if (active) {
          setAutosaveReady(true)
          setLoading(false)
        }
      })
    return () => { active = false }
  }, [editingID, storageKey])

  useEffect(() => {
    if (!autosaveReady) return
    setAutosaveText('有未保存改动')
    const timer = window.setTimeout(() => {
      if (hasDraftContent(draft)) {
        savePostDraft(storageKey, draft)
        setAutosaveText(`已自动保存 ${new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}`)
      } else {
        clearPostDraft(storageKey)
        setAutosaveText('尚未自动保存')
      }
    }, 800)
    return () => window.clearTimeout(timer)
  }, [autosaveReady, draft, storageKey])

  const save = async (status: 'draft' | 'scheduled' | 'published') => {
    setError('')
    setSaving(true)
    try {
      const post = editingID ? await updateUserPost(editingID, { ...draft, status }) : await createUserPost({ ...draft, status })
      clearPostDraft(storageKey)
      navigate(status === 'published' ? `/posts/${post.slug}` : `/me/posts?status=${status}`)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '保存文章失败')
    } finally {
      setSaving(false)
    }
  }

  const submit = (event: FormEvent) => {
    event.preventDefault()
    void save('published')
  }

  const restoreLocalDraft = () => {
    if (!recovery) return
    setDraft(recovery.draft)
    setRecovery(undefined)
    setAutosaveText('已恢复本地草稿')
  }

  const discardLocalDraft = () => {
    clearPostDraft(storageKey)
    setRecovery(undefined)
  }

  const applyPost = (post: Post) => {
    setDraft({ title: post.title, slug: post.slug, summary: post.summary, content: post.content, coverImage: post.coverImage, tags: post.tags, status: post.status, featured: post.featured, categoryId: post.category?.id ?? null, scheduledAt: post.scheduledAt ?? null })
  }

  const restoreRevision = async () => {
    if (!editingID || !revisionToRestore) return
    setSaving(true)
    setError('')
    try {
      applyPost(await restorePostRevision(editingID, revisionToRestore.id))
      setRevisions(await getPostRevisions(editingID))
      setRevisionToRestore(undefined)
      setAutosaveText('已恢复历史版本')
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '恢复文章版本失败')
    } finally {
      setSaving(false)
    }
  }

  if (checking || loading) return <div className="container page-state"><span className="eyebrow">{editingID ? 'Edit post' : 'New post'}</span><h1>正在准备编辑器。</h1></div>
  return <section className="write-page container">
    <div className="write-topbar">
      <Link className="back-link" to="/"><ArrowLeft size={15} /> 返回社区</Link>
      <div className="editor-status"><FileClock size={14} /><span>{autosaveText}</span></div>
      <div className="editor-mode" role="tablist" aria-label="编辑器模式">
        <button type="button" className={mode === 'write' ? 'is-active' : ''} onClick={() => setMode('write')}><PenLine size={14} /> 编辑</button>
        <button type="button" className={mode === 'preview' ? 'is-active' : ''} onClick={() => setMode('preview')}><Eye size={14} /> 预览</button>
      </div>
    </div>
    {recovery && <div className="draft-recovery"><div><RotateCcw size={17} /><span><strong>发现本地草稿</strong><small>{new Date(recovery.savedAt).toLocaleString()} 自动保存</small></span></div><div><button type="button" className="button button-light" onClick={discardLocalDraft}>忽略</button><button type="button" className="button button-dark" onClick={restoreLocalDraft}>恢复草稿</button></div></div>}
    <form className="write-layout" onSubmit={submit}>
      <main className="write-main">
        <input className="write-title" value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} placeholder="写下标题" required />
        <textarea className="write-summary" value={draft.summary} onChange={(event) => setDraft({ ...draft, summary: event.target.value })} placeholder="用一句话说明这篇文章" />
        {mode === 'write'
          ? <textarea className="write-content" value={draft.content} onChange={(event) => setDraft({ ...draft, content: event.target.value })} placeholder="支持 Markdown，写下你的想法。" required />
          : <div className="write-preview"><MarkdownContent className="article-content" content={draft.content || '*这里会显示 Markdown 预览。*'} /></div>}
      </main>
      <aside className="write-sidebar">
        <div className="write-panel"><div className="eyebrow">Publish</div><p>公开文章会出现在最新和热门信息流中。</p><div className="write-actions"><button type="button" className="button button-light" onClick={() => void save('draft')} disabled={saving}><Save size={16} /> {editingID ? '存为草稿' : '保存草稿'}</button><button type="button" className="button button-light" onClick={() => void save('scheduled')} disabled={saving || !draft.scheduledAt}><CalendarClock size={16} /> 定时发布</button><button type="submit" className="button button-dark" disabled={saving}><Send size={16} /> {editingID ? '保存并发布' : '发布文章'}</button></div></div>
        <div className="editor-counts"><span>{draft.content.trim() ? draft.content.trim().split(/\s+/).length : 0} 词</span><span>{draft.content.length} 字符</span></div>
        <label>链接标识<input value={draft.slug} onChange={(event) => setDraft({ ...draft, slug: event.target.value })} placeholder="留空自动生成" /></label>
        <label>标签<input value={draft.tags.join(', ')} onChange={(event) => setDraft({ ...draft, tags: event.target.value.split(',').map((tag) => tag.trim()).filter(Boolean) })} placeholder="设计, 工程" /></label>
        <label>分类<select value={draft.categoryId ?? ''} onChange={(event) => setDraft({ ...draft, categoryId: event.target.value ? Number(event.target.value) : null })}><option value="">不设分类</option>{categories.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}</select></label>
        <label>计划发布时间<input type="datetime-local" min={dateTimeLocalValue(new Date().toISOString())} value={dateTimeLocalValue(draft.scheduledAt)} onChange={(event) => setDraft({ ...draft, scheduledAt: event.target.value ? new Date(event.target.value).toISOString() : null })} /></label>
        <label>封面图片<ImageUploadField value={draft.coverImage} onChange={(coverImage) => setDraft({ ...draft, coverImage })} /></label>
        {editingID && <div className="revision-panel"><div className="revision-heading"><span><History size={14} /> 版本历史</span><small>最近 {revisions.length} 个版本</small></div>{revisions.length ? <div className="revision-list">{revisions.map((revision) => <button type="button" key={revision.id} onClick={() => setRevisionToRestore(revision)}><strong>{revision.title}</strong><small>{new Date(revision.createdAt).toLocaleString()}</small></button>)}</div> : <p>保存修改后会在这里生成历史版本。</p>}</div>}
        {error && <div className="form-error">{error}</div>}
      </aside>
    </form>
    <ConfirmDialog open={Boolean(revisionToRestore)} title="恢复历史版本" message={revisionToRestore ? `恢复到 ${new Date(revisionToRestore.createdAt).toLocaleString()} 的内容？当前内容会自动保存为新版本。` : undefined} onCancel={() => setRevisionToRestore(undefined)} onConfirm={restoreRevision} confirmLabel="恢复版本" />
  </section>
}
