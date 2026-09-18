import { ArrowDownRight, ArrowUpRight, Command, MoveUpRight } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useEffect, useMemo, useState } from 'react'
import { PostCard } from '../components/PostCard'
import { SectionHeading } from '../components/SectionHeading'
import { getPosts } from '../services/api'
import type { Post } from '../types'
import { formatDate } from '../utils'

export function HomePage() {
  const [posts, setPosts] = useState<Post[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    getPosts({ pageSize: 12 })
      .then((data) => setPosts(data.items))
      .catch((reason: Error) => setError(reason.message))
      .finally(() => setLoading(false))
  }, [])

  const featured = posts.find((post) => post.featured) ?? posts[0]
  const allTags = useMemo(() => Array.from(new Set(posts.flatMap((post) => post.tags))), [posts])

  if (loading) return <div className="container page-state"><span className="eyebrow">Loading notes</span><h1>正在读取文章。</h1></div>
  if (error) return <div className="container page-state"><span className="eyebrow">Connection error</span><h1>文章暂时无法加载。</h1><p>{error}</p></div>
  if (!featured) return <div className="container page-state"><span className="eyebrow">No published posts</span><h1>还没有公开文章。</h1></div>

  return (
    <>
      <section className="home-intro container">
        <div className="intro-copy">
          <div className="eyebrow">独立开发者 / 产品设计 / 代码实践</div>
          <h1>把复杂的事，<br /><em>写得清楚一点。</em></h1>
          <p className="intro-lead">你好，我是 Yiming。这里是我的个人博客，记录界面、系统和生活里那些值得慢慢想清楚的部分。</p>
          <div className="intro-actions">
            <Link className="button button-dark" to="/posts">浏览全部文章 <ArrowUpRight size={16} /></Link>
            <Link className="text-button" to="/about">了解我 <ArrowUpRight size={15} /></Link>
          </div>
        </div>
        <div className="intro-signal" aria-label="个人博客视觉标识">
          <div className="signal-frame">
            <div className="signal-topline"><span>QS / 2026</span><span>01—04</span></div>
            <div className="signal-core">
              <div className="signal-letter">Y</div>
              <div className="signal-cross" />
              <div className="signal-letter signal-letter-right">M</div>
            </div>
            <div className="signal-bottomline"><span>QUIET SIGNAL</span><span>BUILD / WRITE / OBSERVE</span></div>
          </div>
          <div className="signal-note"><Command size={15} /> <span>少一点噪音，多一点判断</span></div>
        </div>
      </section>

      <section className="container featured-section">
        <SectionHeading eyebrow="Selected note" title="精选文章" action={<Link className="section-link" to={`/posts/${featured.slug}`}>打开精选 <ArrowUpRight size={15} /></Link>} />
        <div className="featured-layout">
          <div className="featured-visual">
            <img src={featured.coverImage} alt="" />
            <span className="featured-index">01</span>
          </div>
          <div className="featured-copy">
            <div className="post-meta"><span>{formatDate(featured.publishedAt)}</span><span className="meta-dot" /><span>{featured.readingTime} min read</span></div>
            <h3><Link to={`/posts/${featured.slug}`}>{featured.title}</Link></h3>
            <p>{featured.summary}</p>
            <div className="tag-row">{featured.tags.map((tag) => <span className="tag" key={tag}>{tag}</span>)}</div>
            <Link className="round-link" to={`/posts/${featured.slug}`} aria-label="阅读精选文章"><MoveUpRight size={19} /></Link>
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
