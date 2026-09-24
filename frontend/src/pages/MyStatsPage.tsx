import { ArrowLeft, Eye, FileText, Heart, MessageCircle, Users } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useAuth } from '../services/auth'
import { useMyStats } from '../services/queries'
import { SkeletonList } from '../components/Skeleton'
import { usePageMeta } from '../utils/usePageMeta'

// 作者数据看板：自己文章的总量指标与近 14 天互动趋势
export function MyStatsPage() {
  usePageMeta('数据看板')
  const { user } = useAuth()
  const statsQuery = useMyStats(Boolean(user))
  const stats = statsQuery.data

  if (statsQuery.isLoading) return <section className="container my-posts-page"><SkeletonList count={5} /></section>
  if (!stats) return <div className="container page-state"><span className="eyebrow">Stats</span><h1>统计数据暂时无法加载。</h1></div>

  const cards = [
    { icon: FileText, label: '文章', value: stats.posts, hint: `${stats.published} 已发布 · ${stats.drafts} 草稿` },
    { icon: Eye, label: '总阅读', value: stats.views, hint: '全部文章累计' },
    { icon: Heart, label: '获赞', value: stats.likes, hint: '全部文章累计' },
    { icon: MessageCircle, label: '被评论', value: stats.comments, hint: '全部文章累计' },
    { icon: Users, label: '粉丝', value: stats.followers, hint: '关注你的读者' },
  ]
  const maxDaily = Math.max(1, ...stats.dailyMetrics.map((day) => day.posts + day.likes + day.comments))

  return (
    <section className="container my-posts-page">
      <div className="write-topbar"><Link className="back-link" to="/me/posts"><ArrowLeft size={15} /> 我的文章</Link><span className="eyebrow">Stats</span></div>
      <h1>数据看板</h1>
      <div className="stats-cards">{cards.map((card) => <div className="stats-card" key={card.label}><card.icon size={17} /><strong>{card.value.toLocaleString()}</strong><span>{card.label}</span><small>{card.hint}</small></div>)}</div>

      <div className="admin-panel"><div className="admin-panel-heading"><div><div className="eyebrow">Last 14 days</div><h2>近 14 天互动趋势</h2></div><small>发布 / 获赞 / 被评论</small></div>
        <div className="admin-chart">{stats.dailyMetrics.map((day) => (
          <div className="admin-chart-day" key={day.date} title={`${day.date}：发布 ${day.posts} · 获赞 ${day.likes} · 评论 ${day.comments}`}>
            <div className="admin-chart-bars">
              <i className="chart-posts" style={{ height: `${Math.max(3, (day.posts / maxDaily) * 100)}%` }} />
              <i className="chart-users" style={{ height: `${Math.max(3, (day.likes / maxDaily) * 100)}%` }} />
              <i className="chart-comments" style={{ height: `${Math.max(3, (day.comments / maxDaily) * 100)}%` }} />
            </div>
            <small>{day.date.slice(5)}</small>
          </div>
        ))}</div>
      </div>

      <div className="admin-panel"><div className="admin-panel-heading"><div><div className="eyebrow">Top posts</div><h2>最受欢迎的文章</h2></div></div>
        <div className="admin-top-posts">{stats.topPosts.length ? stats.topPosts.map((post, index) => (
          <Link className="admin-top-post" to={`/posts/${post.slug}`} key={post.id}>
            <strong>{String(index + 1).padStart(2, '0')}</strong>
            <span><b>{post.title}</b><small>{post.views} 阅读 · {post.likesCount} 赞 · {post.commentsCount} 评论</small></span>
          </Link>
        )) : <p className="muted-copy">还没有公开文章，发布第一篇后这里会出现排名。</p>}</div>
      </div>
    </section>
  )
}
