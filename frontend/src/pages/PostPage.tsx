import { ArrowLeft, ArrowRight, CalendarDays, Clock3, Eye, Share2 } from 'lucide-react'
import { Link, useParams } from 'react-router-dom'
import { useEffect, useState } from 'react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { getPostBySlug, getPosts } from '../services/api'
import type { Post } from '../types'
import { formatDate } from '../utils'

export function PostPage() {
  const { slug } = useParams()
  const [post, setPost] = useState<Post>()
  const [siblings, setSiblings] = useState<Post[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!slug) return
    let active = true
    setLoading(true)
    setError('')
    Promise.all([getPostBySlug(slug), getPosts({ pageSize: 50 })])
      .then(([detail, page]) => {
        if (!active) return
        setPost(detail)
        setSiblings(page.items)
      })
      .catch((reason: Error) => { if (active) setError(reason.message) })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [slug])

  if (loading) return <div className="container page-state"><span className="eyebrow">Loading article</span><h1>正在读取文章。</h1></div>
  if (error) return <div className="container page-state"><span className="eyebrow">Connection error</span><h1>文章暂时无法加载。</h1><p>{error}</p></div>

  if (!post) return <div className="container empty-state post-not-found"><h1>文章不存在</h1><Link className="text-button" to="/posts">返回文章列表 <ArrowRight size={15} /></Link></div>

  const index = siblings.findIndex((item) => item.slug === post.slug)
  const previous = siblings[index + 1]
  const next = siblings[index - 1]

  return (
    <article className="article-page">
      <header className="article-header container">
        <Link className="back-link" to="/posts"><ArrowLeft size={15} /> 返回文章列表</Link>
        <div className="article-kicker"><span className="eyebrow">{post.tags[0] ?? 'Note'}</span><span className="meta-dot" /><span>{formatDate(post.publishedAt)}</span></div>
        <h1>{post.title}</h1>
        <p className="article-summary">{post.summary}</p>
        <div className="article-meta"><span><CalendarDays size={15} /> {formatDate(post.publishedAt)}</span><span><Clock3 size={15} /> {post.readingTime} min read</span><span><Eye size={15} /> {post.views.toLocaleString()} views</span><button className="icon-text-button" aria-label="分享文章" onClick={() => navigator.clipboard?.writeText(window.location.href)}><Share2 size={15} /> 分享</button></div>
      </header>
      <div className="article-hero container"><img src={post.coverImage} alt="" /></div>
      <div className="article-body-wrap container">
        <aside className="article-aside"><div className="aside-label">On this page</div><a href="#article-content">正文内容</a><a href="#article-end">继续阅读</a></aside>
        <div className="article-content" id="article-content"><ReactMarkdown remarkPlugins={[remarkGfm]}>{post.content}</ReactMarkdown><div id="article-end" /></div>
      </div>
      <div className="container article-nav">
        {previous ? <Link to={`/posts/${previous.slug}`} className="article-nav-item"><span><ArrowLeft size={15} /> 上一篇</span><strong>{previous.title}</strong></Link> : <span />}
        {next ? <Link to={`/posts/${next.slug}`} className="article-nav-item align-right"><span>下一篇 <ArrowRight size={15} /></span><strong>{next.title}</strong></Link> : <span />}
      </div>
    </article>
  )
}
