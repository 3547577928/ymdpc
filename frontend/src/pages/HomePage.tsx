import { ArrowDownRight, ArrowUpRight, Command } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useEffect, useMemo, useState } from 'react'
import { PostCard } from '../components/PostCard'
import { SectionHeading } from '../components/SectionHeading'
import { CoverImage } from '../components/CoverImage'
import { getFeed, getTrendingTags } from '../services/api'
import type { PostSummary } from '../types'
import { formatDate } from '../utils'

export function HomePage() {
  const currentYear = new Date().getFullYear()
  const [posts, setPosts] = useState<PostSummary[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [trendingTags, setTrendingTags] = useState<string[]>([])

  useEffect(() => {
    Promise.all([getFeed({ mode: 'latest', pageSize: 12 }), getTrendingTags({ limit: 8 })])
      .then(([data, tags]) => { setPosts(data.items); setTrendingTags(tags.map((tag) => tag.name)) })
      .catch((reason: Error) => setError(reason.message))
      .finally(() => setLoading(false))
  }, [])

  const featured = posts.find((post) => post.featured) ?? posts[0]
  const allTags = useMemo(() => trendingTags.length ? trendingTags : Array.from(new Set(posts.flatMap((post) => post.tags))), [posts, trendingTags])

  if (loading) return <div className="container page-state"><span className="eyebrow">Loading notes</span><h1>正在读取文章。</h1></div>
  if (error) return <div className="container page-state"><span className="eyebrow">Connection error</span><h1>文章暂时无法加载。</h1><p>{error}</p></div>
  if (!featured) return <div className="container page-state"><span className="eyebrow">No published posts</span><h1>还没有公开文章。</h1></div>

  return (
    <>
      <section className="home-intro container">
        <div className="intro-copy">
          <div className="eyebrow">界面 / 系统 / 代码实践 / 社区讨论</div>
          <h1>把复杂的事，<br /><em>写得清楚一点。</em></h1>
          <p className="intro-lead">Quiet Signal 是一个安静的写作社区，记录界面、系统和生活里那些值得慢慢想清楚的部分。</p>
          <div className="intro-actions">
            <Link className="button button-dark" to="/posts">进入社区文章 <ArrowUpRight size={16} /></Link>
            <Link className="text-button" to="/about">了解社区 <ArrowUpRight size={15} /></Link>
          </div>
        </div>
        <div className="intro-signal" aria-label="社区视觉标识">
          <div className="signal-frame">
            <div className="signal-topline"><span>QUIET SIGNAL</span><span>{currentYear}</span></div>
            <div className="signal-core">
              <div className="signal-letter">Q</div>
              <div className="signal-cross" />
              <div className="signal-letter signal-letter-right">S</div>
            </div>
            <div className="signal-bottomline"><span>QUIET SIGNAL</span><span>BUILD / WRITE / OBSERVE</span></div>
          </div>
          <div className="signal-note"><Command size={15} /> <span>少一点噪音，多一点判断</span></div>
        </div>
      </section>

      <section className="container featured-section">
        <SectionHeading eyebrow="Selected note" title="精选文章" action={<Link className="section-link" to={`/posts/${featured.slug}`}>阅读全文 <ArrowUpRight size={15} /></Link>} />
        <div className="featured-layout">
          <div className="featured-visual">
            <CoverImage src={featured.coverImage} title={featured.title} />
            <span className="featured-index">01</span>
          </div>
          <div className="featured-copy">
            <div className="post-meta"><span>{formatDate(featured.publishedAt)}</span><span className="meta-dot" /><span>{featured.readingTime} min read</span></div>
            <h3><Link to={`/posts/${featured.slug}`}>{featured.title}</Link></h3>
            <p>{featured.summary}</p>
            <div className="tag-row">{featured.tags.map((tag) => <span className="tag" key={tag}>{tag}</span>)}</div>
          </div>
        </div>
      </section>

      <section className="recent-section">
        <div className="container">
          <SectionHeading eyebrow="Latest notes" title="最近写了什么" action={<Link className="section-link" to="/posts">查看全部 <ArrowUpRight size={15} /></Link>} />
          <div className="post-grid">{posts.filter((post) => post.id !== featured.id).slice(0, 6).map((post) => <PostCard key={post.id} post={post} />)}</div>
        </div>
      </section>

      <section className="container topics-section">
        <div className="topics-copy"><div className="eyebrow">Browse by topic</div><h2>从一个关键词开始。</h2></div>
        <div className="topics-list">{allTags.map((tag, index) => <Link to={`/posts?tag=${encodeURIComponent(tag)}`} key={tag}><span>0{index + 1}</span>{tag}<ArrowDownRight size={16} /></Link>)}</div>
      </section>
    </>
  )
}
