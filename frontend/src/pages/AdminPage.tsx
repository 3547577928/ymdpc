import { FileText, LayoutDashboard, LogOut, Plus, Settings, Tag, Trash2, UploadCloud, X } from 'lucide-react'
import { FormEvent, useEffect, useState } from 'react'
import { createPost, deletePost, getCurrentUser, getPosts, login, logout, updatePostStatus, type AuthUser, type PostInput } from '../services/api'
import type { Post } from '../types'
import { formatDate } from '../utils'

const emptyDraft: PostInput = { title: '', summary: '', content: '', coverImage: '', tags: [], status: 'draft', featured: false }

export function AdminPage() {
  const [user, setUser] = useState<AuthUser>()
  const [active, setActive] = useState('文章')
  const [posts, setPosts] = useState<Post[]>([])
  const [showEditor, setShowEditor] = useState(false)
  const [draft, setDraft] = useState<PostInput>(emptyDraft)
  const [loginForm, setLoginForm] = useState({ username: 'admin', password: 'admin123' })
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    getCurrentUser().then(setUser).catch(() => undefined).finally(() => setLoading(false))
  }, [])

  useEffect(() => {
    if (!user) return
    getPosts({ pageSize: 50 }, true).then((data) => setPosts(data.items)).catch((reason: Error) => setError(reason.message))
  }, [user])

  const handleLogin = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    try {
      setUser(await login(loginForm.username, loginForm.password))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '登录失败')
    }
  }

  const submitDraft = async (status: Post['status']) => {
    setSaving(true)
    setError('')
    try {
      const created = await createPost({ ...draft, status })
      setPosts((current) => [created, ...current])
      setDraft(emptyDraft)
      setShowEditor(false)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '创建文章失败')
    } finally {
      setSaving(false)
    }
  }

  const handleCreate = (event: FormEvent) => {
    event.preventDefault()
    void submitDraft('published')
  }

  const handlePublish = async (post: Post) => {
    try {
      await updatePostStatus(post.id, post.status === 'published' ? 'draft' : 'published')
      setPosts((current) => current.map((item) => item.id === post.id ? { ...item, status: item.status === 'published' ? 'draft' : 'published' } : item))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '更新状态失败')
    }
  }

  const handleDelete = async (post: Post) => {
    if (!window.confirm(`确定删除「${post.title}」吗？`)) return
    try {
      await deletePost(post.id)
      setPosts((current) => current.filter((item) => item.id !== post.id))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '删除文章失败')
    }
  }

  if (loading) return <div className="container page-state"><span className="eyebrow">Admin</span><h1>正在验证登录状态。</h1></div>
  if (!user) return <section className="admin-login-page"><form className="admin-login-card" onSubmit={handleLogin}><div className="admin-logo">QS <span>CONTROL</span></div><div className="eyebrow">Private workspace</div><h1>登录管理后台</h1><p>管理文章、草稿和公开内容。</p><label>用户名<input value={loginForm.username} onChange={(event) => setLoginForm({ ...loginForm, username: event.target.value })} /></label><label>密码<input type="password" value={loginForm.password} onChange={(event) => setLoginForm({ ...loginForm, password: event.target.value })} /></label>{error && <div className="form-error">{error}</div>}<button className="button button-dark" type="submit">登录后台</button></form></section>

  return (
    <section className="admin-page">
      <div className="admin-sidebar"><div className="admin-logo">QS <span>CONTROL</span></div><div className="admin-nav">{[['概览', LayoutDashboard], ['文章', FileText], ['标签', Tag], ['设置', Settings]].map(([label, Icon]) => <button key={String(label)} className={active === label ? 'is-active' : ''} onClick={() => setActive(String(label))}><Icon size={16} /> {String(label)}</button>)}</div><button className="admin-logout" onClick={async () => { await logout(); setUser(undefined) }}><LogOut size={16} /> 退出登录</button></div>
      <div className="admin-main"><header className="admin-topbar"><div><div className="eyebrow">Admin / {active}</div><h1>{active}</h1></div>{active === '文章' && <button className="button button-dark" onClick={() => setShowEditor(true)}><Plus size={16} /> 写新文章</button>}</header>
        {error && <div className="form-error admin-error">{error}</div>}
        {active === '文章' ? <><div className="admin-stats"><div><span>文章总数</span><strong>{String(posts.length).padStart(2, '0')}</strong><small>来自 SQLite</small></div><div><span>已发布</span><strong>{String(posts.filter((post) => post.status === 'published').length).padStart(2, '0')}</strong><small>公开内容</small></div><div><span>总阅读</span><strong>{posts.reduce((sum, post) => sum + post.views, 0).toLocaleString()}</strong><small>累计访问</small></div></div><div className="admin-table"><div className="table-head"><span>文章标题</span><span>状态</span><span>更新时间</span><span>操作</span></div>{posts.map((post) => <div className="table-row" key={post.id}><div className="table-title"><FileText size={16} /><span>{post.title}</span></div><button className="status-dot status-button" onClick={() => handlePublish(post)}><i className={post.status === 'published' ? '' : 'is-draft'} /> {post.status === 'published' ? '已发布' : '草稿'}</button><span>{formatDate(post.updatedAt ?? post.createdAt ?? post.publishedAt)}</span><button className="icon-button" onClick={() => handleDelete(post)} aria-label={`删除 ${post.title}`}><Trash2 size={15} /></button></div>)}</div></> : <div className="admin-placeholder"><Settings size={22} /><h2>{active}模块</h2><p>标签管理和系统设置会在下一步接入真实数据。</p></div>}
      </div>
      {showEditor && <div className="modal-backdrop" onClick={() => setShowEditor(false)}><form className="editor-modal" onSubmit={handleCreate} onClick={(event) => event.stopPropagation()}><div className="modal-head"><div><div className="eyebrow">New post</div><h2>写一篇新文章</h2></div><button type="button" className="icon-button" onClick={() => setShowEditor(false)} aria-label="关闭"><X size={17} /></button></div><label>标题<input required value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} placeholder="输入文章标题" /></label><label>摘要<textarea value={draft.summary} onChange={(event) => setDraft({ ...draft, summary: event.target.value })} placeholder="用一句话说明这篇文章" /></label><label>标签<input value={draft.tags.join(', ')} onChange={(event) => setDraft({ ...draft, tags: event.target.value.split(',').map((tag) => tag.trim()).filter(Boolean) })} placeholder="React, Go" /></label><label>正文<textarea required className="editor-textarea" value={draft.content} onChange={(event) => setDraft({ ...draft, content: event.target.value })} placeholder="支持 Markdown" /></label><div className="editor-actions"><button type="button" className="button button-light" onClick={() => void submitDraft('draft')} disabled={saving}>保存草稿</button><button type="submit" className="button button-dark" disabled={saving}><UploadCloud size={16} /> {saving ? '保存中' : '发布文章'}</button></div></form></div>}
    </section>
  )
}
