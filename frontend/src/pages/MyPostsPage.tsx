import { ArrowLeft, FileText, Pencil, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { deleteUserPost } from '../services/api'
import { useMyPosts } from '../services/queries'
import { ConfirmDialog } from '../components/Dialog'
import { SkeletonList } from '../components/Skeleton'
import type { PostStatus, PostSummary } from '../types'
import { formatDate } from '../utils'
import { usePageMeta } from '../utils/usePageMeta'

const tabs: { key: string; label: string }[] = [
  { key: '', label: '全部' },
  { key: 'published', label: '已发布' },
  { key: 'scheduled', label: '待发布' },
  { key: 'draft', label: '草稿' },
  { key: 'archived', label: '归档' },
]

const statusLabels: Record<PostStatus, string> = { draft: '草稿', scheduled: '待发布', published: '已发布', archived: '已归档' }

// 我的文章与草稿管理页，作者可以编辑或删除自己的任何状态文章
export function MyPostsPage() {
  usePageMeta('我的文章')
  // 状态筛选与 URL 同步：/me/posts?status=draft 直接进入草稿箱；
  // 用 useSearchParams 响应式读取，导航菜单在页面已挂载时切换草稿箱也能生效
  const [searchParams, setSearchParams] = useSearchParams()
  const status = searchParams.get('status') ?? ''
  const setStatus = (next: string) => setSearchParams(next ? { status: next } : {})
  const [page, setPage] = useState(1)
  const [error, setError] = useState('')
  const [deleteRequest, setDeleteRequest] = useState<PostSummary>()
  const pageSize = 20
  const queryClient = useQueryClient()
  const postsQuery = useMyPosts(status, page, pageSize)
  const posts = postsQuery.data?.items ?? []
  const total = postsQuery.data?.total ?? 0
  const loading = postsQuery.isLoading
  if (postsQuery.error) setError(postsQuery.error.message)

  const handleDelete = (post: PostSummary) => {
    setDeleteRequest(post)
  }

  const deleteMutation = useMutation({
    mutationFn: deleteUserPost,
    // 删除成功后失效我的文章缓存，重新拉取当前页
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ['my-posts'] }),
    onError: (reason) => setError(reason.message),
  })

  const confirmDelete = async () => {
    if (!deleteRequest) return
    // 普通作者走用户端点 /posts/id/:id，admin 端点仅限后台使用
    if (posts.length === 1 && page > 1) setPage((current) => current - 1)
    deleteMutation.mutate(deleteRequest.id)
    setDeleteRequest(undefined)
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return <section className="container my-posts-page">
    <div className="write-topbar"><Link className="back-link" to="/"><ArrowLeft size={15} /> 返回社区</Link><span className="eyebrow">My posts</span></div>
    <h1>我的文章</h1>
    <div className="feed-tabs archive-toolbar-tabs">
      {tabs.map((tab) => <button key={tab.key} className={status === tab.key ? 'is-active' : ''} onClick={() => { setStatus(tab.key); setPage(1) }}>{tab.label}</button>)}
    </div>
    {error && <div className="form-error">{error}</div>}
    {loading ? <SkeletonList count={6} /> : posts.length ? <div className="admin-table">
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
    <ConfirmDialog open={Boolean(deleteRequest)} title="删除文章" message={deleteRequest ? `确定删除「${deleteRequest.title}」吗？` : undefined} onCancel={() => setDeleteRequest(undefined)} onConfirm={confirmDelete} />
  </section>
}
