import { ArrowLeft } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { getMyFavorites } from '../services/api'
import { PostCard } from '../components/PostCard'
import type { PostSummary } from '../types'

// 我的收藏页，展示收藏的文章卡片
export function FavoritesPage() {
  const [posts, setPosts] = useState<PostSummary[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const pageSize = 12

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getMyFavorites({ page, pageSize })
      setPosts(data.items)
      setTotal(data.total)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取收藏失败')
    } finally {
      setLoading(false)
    }
  }, [page])

  useEffect(() => {
    void load()
  }, [load])

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return <section className="container my-posts-page">
    <div className="write-topbar"><Link className="back-link" to="/"><ArrowLeft size={15} /> 返回社区</Link><span className="eyebrow">Favorites</span></div>
    <h1>我的收藏</h1>
    {error && <div className="form-error">{error}</div>}
    {loading ? <div className="page-state"><h1>正在读取。</h1></div> : posts.length ? <div className="post-grid favorites-grid">{posts.map((post) => <PostCard key={post.id} post={post} />)}</div> : <div className="empty-state comments-empty"><h2>还没有收藏</h2><p>在文章页点击收藏，稍后回来阅读。</p></div>}
    {totalPages > 1 && <div className="admin-pagination"><button className="icon-button" disabled={page === 1} onClick={() => setPage((current) => current - 1)}>上一页</button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page >= totalPages} onClick={() => setPage((current) => current + 1)}>下一页</button></div>}
  </section>
}
