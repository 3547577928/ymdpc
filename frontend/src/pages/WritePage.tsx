import { ArrowLeft, CalendarClock, Eye, FileClock, History, PenLine, Plus, RotateCcw, Save, Send } from 'lucide-react'
import type { FormEvent } from 'react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { MarkdownContent } from '../components/MarkdownContent'
import { ImageUploadField } from '../components/ImageUploadField'
import { MarkdownImportButton } from '../components/MarkdownImportButton'
import { CategoryField } from '../components/CategoryField'
import { MarkdownImageUploadButton } from '../components/MarkdownImageUploadButton'
import { ConfirmDialog } from '../components/Dialog'
import { createSeries, createUserPost, getAuthorSeries, getCategories, getPostById, getPostRevisions, restorePostRevision, updateUserPost, type PostInput, type SeriesListItem } from '../services/api'
import { useAuth } from '../services/auth'
import { usePageMeta } from '../utils/usePageMeta'
import type { Category, Post, PostRevision } from '../types'
import { clearPostDraft, hasDraftContent, loadPostDraft, savePostDraft, type StoredPostDraft } from '../utils/draftStorage'

const initialDraft: PostInput = { title: '', slug: '', summary: '', content: '', coverImage: '', tags: [], status: 'draft', featured: false, seriesId: null, scheduledAt: null }

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
  usePageMeta(id ? '编辑文章' : '写文章')
  const [draft, setDraft] = useState<PostInput>(initialDraft)
  const [categories, setCategories] = useState<Category[]>([])
  const [seriesList, setSeriesList] = useState<SeriesListItem[]>([])
  const [newSeriesTitle, setNewSeriesTitle] = useState('')
  const [seriesBusy, setSeriesBusy] = useState(false)
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
  const contentRef = useRef<HTMLTextAreaElement>(null)

  useEffect(() => {
    getCategories().then(setCategories).catch(() => undefined)
  }, [])

  // 登录态由 AuthProvider 提供：探测中保持 loading，确认未登录再跳登录页，
  // 避免已登录用户因 me 请求尚未返回而被误踢
  const { user, loading: authLoading } = useAuth()
  useEffect(() => {
    if (authLoading) return
    if (!user) navigate(`/login?from=${encodeURIComponent(window.location.pathname)}`)
    else setChecking(false)
  }, [user, authLoading, navigate])
  // 作者的系列列表（选择或新建后挂载文章）
  useEffect(() => {
    if (user) getAuthorSeries(user.username).then(setSeriesList).catch(() => undefined)
  }, [user])

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
        const serverDraft: PostInput = { title: post.title, slug: post.slug, summary: post.summary, content: post.content, coverImage: post.coverImage, tags: post.tags, status: post.status, featured: post.featured, categoryId: post.category?.id ?? null, seriesId: post.seriesId ?? null, scheduledAt: post.scheduledAt ?? null }
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
    setDraft({ title: post.title, slug: post.slug, summary: post.summary, content: post.content, coverImage: post.coverImage, tags: post.tags, status: post.status, featured: post.featured, categoryId: post.category?.id ?? null, seriesId: post.seriesId ?? null, scheduledAt: post.scheduledAt ?? null })
  }

  const importMarkdown = (document: { content: string; title: string; summary: string }) => {
    setDraft((current) => ({ ...current, content: document.content, title: current.title.trim() ? current.title : document.title, summary: current.summary.trim() ? current.summary : document.summary }))
    setMode('preview')
    setAutosaveText('已导入 Markdown')
    setError('')
  }

  const insertMarkdownImage = (markdown: string) => {
    setDraft((current) => {
      const textarea = contentRef.current
      const start = textarea?.selectionStart ?? current.content.length
      const end = textarea?.selectionEnd ?? start
      const before = current.content.slice(0, start)
      const after = current.content.slice(end)
      const prefix = before && !before.endsWith('\n') ? '\n' : ''
      const suffix = after && !after.startsWith('\n') ? '\n' : ''
      return { ...current, content: `${before}${prefix}${markdown}${suffix}${after}` }
    })
    setMode('write')
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
      <div className="editor-tools"><MarkdownImportButton onImport={importMarkdown} onError={setError} /><div className="editor-mode" role="tablist" aria-label="编辑器模式">
        <button type="button" className={mode === 'write' ? 'is-active' : ''} onClick={() => setMode('write')}><PenLine size={14} /> 编辑</button>
        <button type="button" className={mode === 'preview' ? 'is-active' : ''} onClick={() => setMode('preview')}><Eye size={14} /> 预览</button>
      </div></div>
    </div>
    {recovery && <div className="draft-recovery"><div><RotateCcw size={17} /><span><strong>发现本地草稿</strong><small>{new Date(recovery.savedAt).toLocaleString()} 自动保存</small></span></div><div><button type="button" className="button button-light" onClick={discardLocalDraft}>忽略</button><button type="button" className="button button-dark" onClick={restoreLocalDraft}>恢复草稿</button></div></div>}
    <form className="write-layout" onSubmit={submit}>
      <main className="write-main">
        <input className="write-title" value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} placeholder="写下标题" required />
        <textarea className="write-summary" value={draft.summary} onChange={(event) => setDraft({ ...draft, summary: event.target.value })} placeholder="用一句话说明这篇文章" />
        {mode === 'write'
          ? <><div className="editor-content-tools write-content-tools"><MarkdownImageUploadButton onInsert={insertMarkdownImage} onError={setError} /><span>图片会插入当前光标位置</span></div><textarea ref={contentRef} className="write-content" value={draft.content} onChange={(event) => setDraft({ ...draft, content: event.target.value })} placeholder="支持 Markdown，写下你的想法。" required /></>
          : <div className="write-preview"><MarkdownContent className="article-content" content={draft.content || '*这里会显示 Markdown 预览。*'} /></div>}
      </main>
      <aside className="write-sidebar">
        <div className="write-panel"><div className="eyebrow">Publish</div><p>公开文章会出现在最新和热门信息流中。</p><div className="write-actions"><button type="button" className="button button-light" onClick={() => void save('draft')} disabled={saving}><Save size={16} /> {editingID ? '存为草稿' : '保存草稿'}</button><button type="button" className="button button-light" onClick={() => void save('scheduled')} disabled={saving || !draft.scheduledAt}><CalendarClock size={16} /> 定时发布</button><button type="submit" className="button button-dark" disabled={saving}><Send size={16} /> {editingID ? '保存并发布' : '发布文章'}</button></div></div>
        <div className="editor-counts"><span>{draft.content.trim() ? draft.content.trim().split(/\s+/).length : 0} 词</span><span>{draft.content.length} 字符</span></div>
        <label>链接标识<input value={draft.slug} onChange={(event) => setDraft({ ...draft, slug: event.target.value })} placeholder="留空自动生成" /></label>
        <label>标签<input value={draft.tags.join(', ')} onChange={(event) => setDraft({ ...draft, tags: event.target.value.split(',').map((tag) => tag.trim()).filter(Boolean) })} placeholder="设计, 工程" /></label>
        <div className="write-field"><span>分类</span><CategoryField categories={categories} value={draft.categoryId} onChange={(categoryId) => setDraft({ ...draft, categoryId })} onError={setError} /></div>
        <div className="write-field"><span>系列</span>
          <select value={draft.seriesId ?? ''} onChange={(event) => setDraft({ ...draft, seriesId: event.target.value ? Number(event.target.value) : null })}>
            <option value="">不属于系列</option>
            {seriesList.map((series) => <option key={series.id} value={series.id}>{series.title}</option>)}
          </select>
          <div className="category-create-row">
            <input value={newSeriesTitle} onChange={(event) => setNewSeriesTitle(event.target.value)} placeholder="新建系列" />
            <button type="button" className="icon-button" disabled={seriesBusy || !newSeriesTitle.trim()} aria-label="创建系列" onClick={async () => {
              setSeriesBusy(true)
              setError('')
              try {
                const created = await createSeries({ title: newSeriesTitle.trim(), description: '' })
                setSeriesList((current) => [created, ...current])
                setDraft((current) => ({ ...current, seriesId: created.id }))
                setNewSeriesTitle('')
              } catch (reason) { setError(reason instanceof Error ? reason.message : '创建系列失败') } finally { setSeriesBusy(false) }
            }}><Plus size={16} /></button>
          </div>
        </div>
        <label>计划发布时间<input type="datetime-local" min={dateTimeLocalValue(new Date().toISOString())} value={dateTimeLocalValue(draft.scheduledAt)} onChange={(event) => setDraft({ ...draft, scheduledAt: event.target.value ? new Date(event.target.value).toISOString() : null })} /></label>
        <label>封面图片<ImageUploadField value={draft.coverImage} onChange={(coverImage) => setDraft({ ...draft, coverImage })} /></label>
        {editingID && <div className="revision-panel"><div className="revision-heading"><span><History size={14} /> 版本历史</span><small>最近 {revisions.length} 个版本</small></div>{revisions.length ? <div className="revision-list">{revisions.map((revision) => <button type="button" key={revision.id} onClick={() => setRevisionToRestore(revision)}><strong>{revision.title}</strong><small>{new Date(revision.createdAt).toLocaleString()}</small></button>)}</div> : <p>保存修改后会在这里生成历史版本。</p>}</div>}
        {error && <div className="form-error">{error}</div>}
      </aside>
    </form>
    <ConfirmDialog open={Boolean(revisionToRestore)} title="恢复历史版本" message={revisionToRestore ? `恢复到 ${new Date(revisionToRestore.createdAt).toLocaleString()} 的内容？当前内容会自动保存为新版本。` : undefined} onCancel={() => setRevisionToRestore(undefined)} onConfirm={restoreRevision} confirmLabel="恢复版本" />
  </section>
}
