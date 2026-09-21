import { ArrowLeft, FileText, Pencil, Trash2 } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { deletePost, getMyPosts } from '../services/api'
import type { PostStatus, PostSummary } from '../types'
import { formatDate } from '../utils'

const tabs: { key: string; label: string }[] = [
  { key: '', label: '全部' },
  { key: 'published', label: '已发布' },
  { key: 'draft', label: '草稿' },
  { key: 'archived', label: '归档' },
]

const statusLabels: Record<PostStatus, string> = { draft: '草稿', published: '已发布', archived: '已归档' }

// 我的文章与草稿管理页，作者可以编辑或删除自己的任何状态文章
export function MyPostsPage() {
  // 初始状态从 URL 读取，支持 /me/posts?status=draft 直接进入草稿箱
  const [status, setStatus] = useState(() => new URLSearchParams(window.location.search).get('status') ?? '')
  const [posts, setPosts] = useState<PostSummary[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const pageSize = 20

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getMyPosts({ status, page, pageSize })
      setPosts(data.items)
      setTotal(data.total)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取文章失败')
    } finally {
      setLoading(false)
    }
  }, [status, page])

  useEffect(() => {
    void load()
  }, [load])

  const handleDelete = async (post: PostSummary) => {
    if (!window.confirm(`确定删除「${post.title}」吗？`)) return
    try {
      await deletePost(post.id)
      if (posts.length === 1 && page > 1) setPage((current) => current - 1)
      else void load()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '删除文章失败')
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return <section className="container my-posts-page">
    <div className="write-topbar"><Link className="back-link" to="/"><ArrowLeft size={15} /> 返回社区</Link><span className="eyebrow">My posts</span></div>
    <h1>我的文章</h1>
    <div className="feed-tabs archive-toolbar-tabs">
      {tabs.map((tab) => <button key={tab.key} className={status === tab.key ? 'is-active' : ''} onClick={() => { setStatus(tab.key); setPage(1) }}>{tab.label}</button>)}
    </div>
    {error && <div className="form-error">{error}</div>}
    {loading ? <div className="page-state"><h1>正在读取。</h1></div> : posts.length ? <div className="admin-table">
      <div className="table-head"><span>标题</span><span>状态</span><span>更新时间</span><span>操作</span></div>
      {posts.map((post) => <div className="table-row" key={post.id}>
        <div className="table-title"><FileText size={16} /><span><Link to={`/posts/${post.slug}`}>{post.title}</Link><small>{post.category ? post.category.name : post.tags[0] ?? ''}</small></span></div>
        <span className="status-dot"><i className={`is-${post.status}`} /> {statusLabels[post.status]}</span>
        <span>{formatDate(post.updatedAt ?? post.createdAt ?? post.publishedAt)}</span>
        <div className="table-actions">
          <Link className="icon-button" to={`/posts/${post.id}/edit`} aria-label={`编辑 ${post.title}`}><Pencil size={15} /></Link>
          <button className="icon-button" onClick={() => void handleDelete(post)} aria-label={`删除 ${post.title}`}><Trash2 size={15} /></button>
        </div>
      </div>)}
    </div> : <div className="empty-state comments-empty"><h2>这里还空着</h2><p>去写下第一篇文章吧。</p><Link className="text-button" to="/write">写文章</Link></div>}
    {totalPages > 1 && <div className="admin-pagination"><button className="icon-button" disabled={page === 1} onClick={() => setPage((current) => current - 1)}>上一页</button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page >= totalPages} onClick={() => setPage((current) => current + 1)}>下一页</button></div>}
  </section>
}
