import { ArrowLeft } from 'lucide-react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMyFavorites } from '../services/queries'
import { PostCard } from '../components/PostCard'
import { SkeletonPostGrid } from '../components/Skeleton'
import { usePageMeta } from '../utils/usePageMeta'

// 我的收藏页，展示收藏的文章卡片
export function FavoritesPage() {
  usePageMeta('我的收藏')
  const [page, setPage] = useState(1)
  const pageSize = 12
  const favoritesQuery = useMyFavorites(page, pageSize)
  const posts = favoritesQuery.data?.items ?? []
  const total = favoritesQuery.data?.total ?? 0
  const loading = favoritesQuery.isLoading
  const error = favoritesQuery.error?.message ?? ''

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return <section className="container my-posts-page">
    <div className="write-topbar"><Link className="back-link" to="/"><ArrowLeft size={15} /> 返回社区</Link><span className="eyebrow">Favorites</span></div>
    <h1>我的收藏</h1>
    {error && <div className="form-error">{error}</div>}
    {loading ? <SkeletonPostGrid count={6} /> : posts.length ? <div className="post-grid favorites-grid">{posts.map((post) => <PostCard key={post.id} post={post} />)}</div> : <div className="empty-state comments-empty"><h2>还没有收藏</h2><p>在文章页点击收藏，稍后回来阅读。</p></div>}
    {totalPages > 1 && <div className="admin-pagination"><button className="icon-button" disabled={page === 1} onClick={() => setPage((current) => current - 1)}>上一页</button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page >= totalPages} onClick={() => setPage((current) => current + 1)}>下一页</button></div>}
  </section>
}
