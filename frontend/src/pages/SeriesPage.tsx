import { ArrowLeft, Layers } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { getSeries } from '../services/api'
import { SkeletonList } from '../components/Skeleton'
import { usePageMeta } from '../utils/usePageMeta'

// 系列目录页：按发布时间正序展示一组有序文章
export function SeriesPage() {
  const { slug = '' } = useParams()
  const seriesQuery = useQuery({ queryKey: ['series', slug], queryFn: () => getSeries(slug) })
  const series = seriesQuery.data
  usePageMeta(series ? `系列 · ${series.title}` : '系列', series?.description || undefined)

  if (seriesQuery.isLoading) return <section className="container my-posts-page"><SkeletonList count={4} /></section>
  if (!series) return <div className="container page-state"><span className="eyebrow">Series</span><h1>系列不存在。</h1><p><Link className="text-button" to="/posts">回到文章列表</Link></p></div>

  return (
    <section className="container series-page">
      <div className="write-topbar"><Link className="back-link" to={`/users/${series.author.username}`}><ArrowLeft size={15} /> {series.author.nickname} 的主页</Link><span className="eyebrow">Series</span></div>
      <header className="series-page-header">
        <div className="eyebrow"><Layers size={13} /> 系列 / 共 {series.posts.length} 篇</div>
        <h1>{series.title}</h1>
        {series.description && <p>{series.description}</p>}
      </header>
      {series.posts.length ? <ol className="series-catalog">{series.posts.map((post, index) => (
        <li key={post.slug}><Link to={`/posts/${post.slug}`}><span>{String(index + 1).padStart(2, '0')}</span><strong>{post.title}</strong></Link></li>
      ))}</ol> : <div className="empty-state"><h2>系列下还没有公开文章</h2><p>作者在写作页把文章挂进系列后，会按顺序出现在这里。</p></div>}
    </section>
  )
}
