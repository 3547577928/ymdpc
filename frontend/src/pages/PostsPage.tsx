import { ChevronLeft, ChevronRight, Heart, MessageCircle, Search, SlidersHorizontal, Trophy, X } from 'lucide-react'
import { Link } from 'react-router-dom'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { PostCard } from '../components/PostCard'
import { getFeed, getTags } from '../services/api'
import type { PostSummary } from '../types'
import { formatDate } from '../utils'

export function PostsPage() {
  const pageSize = 12
  const currentYear = new Date().getFullYear()
  const [params, setParams] = useSearchParams()
  const query = params.get('q') ?? ''
  const page = Math.max(1, Number(params.get('page')) || 1)
  const mode = params.get('mode') ?? 'latest'
  const [queryInput, setQueryInput] = useState(query)
  const [posts, setPosts] = useState<PostSummary[]>([])
  const [rankingPosts, setRankingPosts] = useState<PostSummary[]>([])
  const [allTags, setAllTags] = useState<string[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const selectedTag = params.get('tag') ?? '全部'

  useEffect(() => {
    setQueryInput(query)
  }, [query])

  useEffect(() => {
    if (queryInput === query) return
    const timer = window.setTimeout(() => {
      const next = new URLSearchParams(params)
      if (queryInput) next.set('q', queryInput)
      else next.delete('q')
      next.delete('page')
      setParams(next, { replace: true })
    }, 300)
    return () => window.clearTimeout(timer)
  }, [params, query, queryInput, setParams])

  useEffect(() => {
    getTags().then((tags) => setAllTags(tags.map((tag) => tag.name))).catch((reason: Error) => setError(reason.message))
    getFeed({ mode: 'hot', pageSize: 5 }).then((data) => setRankingPosts(data.items)).catch(() => setRankingPosts([]))
  }, [])

  useEffect(() => {
    let active = true
    setLoading(true)
    setError('')
    getFeed({ mode, q: query, tag: selectedTag === '全部' ? undefined : selectedTag, page, pageSize }).then((postPage) => {
      if (!active) return
      setPosts(postPage.items)
      setTotal(postPage.total)
    }).catch((reason: Error) => {
      if (active) setError(reason.message)
    }).finally(() => {
      if (active) setLoading(false)
    })
    return () => { active = false }
  }, [mode, page, query, selectedTag])

  const changeTag = (tag: string) => {
    const next = new URLSearchParams(params)
    if (tag === '全部') next.delete('tag')
    else next.set('tag', tag)
    next.delete('page')
    setParams(next)
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  const changePage = (nextPage: number) => {
    const next = new URLSearchParams(params)
    if (nextPage <= 1) next.delete('page')
    else next.set('page', String(nextPage))
    setParams(next)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  const feedContent = loading ? <div className="empty-state"><h2>正在加载文章</h2><p>正在从社区读取公开内容。</p></div> : error ? <div className="empty-state"><h2>文章加载失败</h2><p>{error}</p></div> : posts.length ? <>
    {mode === 'following' ? <div className="following-feed">{posts.map((post) => <div className="following-update" key={post.id}>
      <Link className="following-author" to={`/users/${post.author.username}`}>
        <span className="following-avatar">{post.author.avatar ? <img src={post.author.avatar} alt="" /> : post.author.nickname.slice(0, 1)}</span>
        <span><strong>{post.author.nickname}</strong><small>发布了新文章 · {formatDate(post.publishedAt)}</small></span>
      </Link>
      <PostCard post={post} showAuthor={false} />
    </div>)}</div> : <div className="archive-grid">{posts.map((post) => <PostCard key={post.id} post={post} />)}</div>}
    {totalPages > 1 && <div className="archive-pagination"><button className="icon-button" disabled={page <= 1} onClick={() => changePage(page - 1)} aria-label="上一页"><ChevronLeft size={17} /></button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page >= totalPages} onClick={() => changePage(page + 1)} aria-label="下一页"><ChevronRight size={17} /></button></div>}
  </> : <div className="empty-state"><h2>{mode === 'following' ? '还没有关注更新' : '没有找到相关内容'}</h2><p>{mode === 'following' ? '关注感兴趣的作者后，他们发布的新文章会按时间出现在这里。' : '换一个关键词或清除筛选条件。'}</p></div>

  return (
    <section className="container archive-page">
      <div className="archive-header">
        <div><div className="eyebrow">Community / {currentYear}</div><h1>{mode === 'hot' ? '热门讨论' : mode === 'following' ? '关注动态' : '社区文章'}</h1><p>{mode === 'following' ? '只看你关注的作者，按发布时间展示最新更新。' : '阅读、写作和讨论都从这里开始。'}</p></div>
        <div className="archive-count"><strong>{String(total).padStart(2, '0')}</strong><span>{mode === 'following' ? '篇关注更新' : '篇公开文章'}</span></div>
      </div>

      <div className="archive-toolbar">
        <label className="search-box"><Search size={17} /><input value={queryInput} onChange={(event) => setQueryInput(event.target.value)} placeholder="搜索标题或关键词" /></label>
        <div className="feed-tabs"><Link className={mode === 'latest' ? 'is-active' : ''} to={`/posts?${new URLSearchParams({ ...Object.fromEntries(params), mode: 'latest' })}`}>最新</Link><Link className={mode === 'hot' ? 'is-active' : ''} to={`/posts?${new URLSearchParams({ ...Object.fromEntries(params), mode: 'hot' })}`}>热门</Link>{mode === 'following' && <Link className="is-active" to="/posts?mode=following">关注</Link>}</div>
      </div>
      <div className="toolbar-label"><SlidersHorizontal size={15} /> 筛选主题</div>
      <div className="filter-row">
        {['全部', ...allTags].map((tag) => <button key={tag} className={`filter-chip ${selectedTag === tag ? 'is-active' : ''}`} onClick={() => changeTag(tag)}>{tag}</button>)}
        {(queryInput || selectedTag !== '全部') && <button className="clear-filter" onClick={() => { setQueryInput(''); setParams({}) }}><X size={14} /> 清除筛选</button>}
      </div>

      <div className="community-content">
        <div className="feed-content">{feedContent}</div>
        <aside className="ranking-panel">
          <div className="ranking-heading"><span><Trophy size={16} /> 热门榜</span><small>实时热度</small></div>
          <div className="ranking-list">{rankingPosts.length ? rankingPosts.map((post, index) => <Link className="ranking-item" to={`/posts/${post.slug}`} key={post.id}>
            <span className="ranking-number">{String(index + 1).padStart(2, '0')}</span>
            <span className="ranking-copy"><strong>{post.title}</strong><small>@{post.author.nickname}</small></span>
            <span className="ranking-stats"><span><Heart size={12} />{post.likesCount}</span><span><MessageCircle size={12} />{post.commentsCount}</span></span>
          </Link>) : <p className="ranking-empty">暂时还没有足够的热度数据。</p>}</div>
        </aside>
      </div>
    </section>
  )
}
