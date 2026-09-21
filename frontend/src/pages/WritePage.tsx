import { ArrowLeft, Save, Send } from 'lucide-react'
import { FormEvent, useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { createUserPost, getCategories, getCurrentUser, getPostById, updateUserPost, type PostInput } from '../services/api'
import type { Category } from '../types'

const initialDraft: PostInput = { title: '', slug: '', summary: '', content: '', coverImage: '', tags: [], status: 'draft', featured: false }

export function WritePage() {
  const navigate = useNavigate()
  const { id } = useParams()
  // 携带 :id 时进入编辑模式，加载自己的文章
  const editingID = id ? Number(id) : undefined
  const [draft, setDraft] = useState<PostInput>(initialDraft)
  const [categories, setCategories] = useState<Category[]>([])
  const [saving, setSaving] = useState(false)
  const [loading, setLoading] = useState(Boolean(editingID))
  const [error, setError] = useState('')
  const [checking, setChecking] = useState(true)

  useEffect(() => {
    getCategories().then(setCategories).catch(() => undefined)
  }, [])

  useEffect(() => {
    getCurrentUser().catch(() => navigate('/login')).finally(() => setChecking(false))
  }, [navigate])

  useEffect(() => {
    if (!editingID) return
    getPostById(editingID)
      .then((post) => setDraft({ title: post.title, slug: post.slug, summary: post.summary, content: post.content, coverImage: post.coverImage, tags: post.tags, status: post.status, featured: post.featured, categoryId: post.category?.id ?? null }))
      .catch((reason: Error) => setError(reason.message))
      .finally(() => setLoading(false))
  }, [editingID])

  const submit = async (event: FormEvent, status: 'draft' | 'published') => {
    event.preventDefault()
    setError('')
    setSaving(true)
    try {
      const post = editingID ? await updateUserPost(editingID, { ...draft, status }) : await createUserPost({ ...draft, status })
      navigate(`/posts/${post.slug}`)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '保存文章失败')
    } finally {
      setSaving(false)
    }
  }

  if (checking || loading) return <div className="container page-state"><span className="eyebrow">{editingID ? 'Edit post' : 'New post'}</span><h1>正在准备编辑器。</h1></div>
  return <section className="write-page container">
    <div className="write-topbar"><Link className="back-link" to="/"><ArrowLeft size={15} /> 返回社区</Link><span className="eyebrow">{editingID ? 'Edit post' : 'New post'}</span></div>
    <form className="write-layout" onSubmit={(event) => void submit(event, 'published')}>
      <main className="write-main">
        <input className="write-title" value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} placeholder="写下标题" required />
        <textarea className="write-summary" value={draft.summary} onChange={(event) => setDraft({ ...draft, summary: event.target.value })} placeholder="用一句话说明这篇文章" />
        <textarea className="write-content" value={draft.content} onChange={(event) => setDraft({ ...draft, content: event.target.value })} placeholder="支持 Markdown，写下你的想法。" required />
      </main>
      <aside className="write-sidebar">
        <div className="write-panel"><div className="eyebrow">Publish</div><p>公开文章会出现在最新和热门信息流中。</p><div className="write-actions"><button type="button" className="button button-light" onClick={(event) => void submit(event, 'draft')} disabled={saving}><Save size={16} /> {editingID ? '存为草稿' : '保存草稿'}</button><button type="submit" className="button button-dark" disabled={saving}><Send size={16} /> {editingID ? '保存修改' : '发布文章'}</button></div></div>
        <label>链接标识<input value={draft.slug} onChange={(event) => setDraft({ ...draft, slug: event.target.value })} placeholder="留空自动生成" /></label>
        <label>标签<input value={draft.tags.join(', ')} onChange={(event) => setDraft({ ...draft, tags: event.target.value.split(',').map((tag) => tag.trim()).filter(Boolean) })} placeholder="设计, 工程" /></label>
        <label>分类<select value={draft.categoryId ?? ''} onChange={(event) => setDraft({ ...draft, categoryId: event.target.value ? Number(event.target.value) : null })}><option value="">不设分类</option>{categories.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}</select></label>
        <label>封面地址<input value={draft.coverImage} onChange={(event) => setDraft({ ...draft, coverImage: event.target.value })} placeholder="https://..." /></label>
        {error && <div className="form-error">{error}</div>}
      </aside>
    </form>
  </section>
}
