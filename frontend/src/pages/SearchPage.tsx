import { CalendarDays, ChevronLeft, ChevronRight, Eye, Heart, MessageCircle, Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useSearch } from '../services/queries'
import { SkeletonList } from '../components/Skeleton'
import { formatDate } from '../utils'
import { usePageMeta } from '../utils/usePageMeta'

// 高亮标记渲染：后端用 \u0001/\u0002 标出命中片段，切分后奇数段包 <mark>。
// 全程走 React 文本渲染，不接触 innerHTML
export function MarkedText({ text }: { text: string }) {
  // eslint-disable-next-line no-control-regex -- 后端用控制符标出命中片段，见 search_page.go
  const parts = text.split(/[\u0001\u0002]/)
  return <>{parts.map((part, index) => index % 2 === 1 ? <mark key={index}>{part}</mark> : part)}</>
}

const pageSize = 12

export function SearchPage() {
  usePageMeta('搜索')
  const [params, setParams] = useSearchParams()
  const q = params.get('q') ?? ''
  const page = Math.max(1, Number(params.get('page')) || 1)
  const [input, setInput] = useState(q)

  useEffect(() => { setInput(q) }, [q])
  // 输入防抖写入 URL，URL 是查询的唯一事实来源
  useEffect(() => {
    if (input.trim() === q) return
    const timer = window.setTimeout(() => {
      const next = new URLSearchParams(params)
      if (input.trim()) next.set('q', input.trim())
      else next.delete('q')
      next.delete('page')
      setParams(next, { replace: true })
    }, 300)
    return () => window.clearTimeout(timer)
  }, [input, params, q, setParams])

  const searchQuery = useSearch(q, page, pageSize)
  const items = searchQuery.data?.items ?? []
  const total = searchQuery.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const changePage = (nextPage: number) => {
    const next = new URLSearchParams(params)
    if (nextPage <= 1) next.delete('page')
    else next.set('page', String(nextPage))
    setParams(next)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  return (
    <section className="container search-page">
      <header className="forum-header"><div><span className="eyebrow">Search</span><h1>搜索</h1><p>全文检索标题、摘要与正文。</p></div>{q && <div className="archive-count"><strong>{total.toLocaleString()}</strong><span>条结果</span></div>}</header>
      <label className="search-box search-page-box"><Search size={17} /><input value={input} onChange={(event) => setInput(event.target.value)} placeholder="至少输入 3 个字符，试试「界面」之外的更长关键词" autoFocus /></label>
      {searchQuery.data && !searchQuery.data.fts && q && <p className="muted-copy">当前为兼容模式搜索（关键词过短或索引不可用），不提供高亮。</p>}
      {!q ? <div className="empty-state"><h2>输入关键词开始搜索</h2><p>支持中文与英文，按相关度排序。</p></div>
        : searchQuery.isLoading ? <SkeletonList count={6} />
        : items.length === 0 && !searchQuery.data?.topics.length && !searchQuery.data?.users.length ? <div className="empty-state"><h2>没有找到「{q}」相关内容</h2><p>换个关键词，或去 <Link to="/posts">文章列表</Link> 逛逛。</p></div>
        : <>
        {(searchQuery.data?.users.length ?? 0) > 0 && <div className="search-section"><div className="eyebrow">用户</div><div className="search-users">{searchQuery.data!.users.map((item) => (
          <Link className="search-user" to={`/users/${item.username}`} key={item.id}><span className="comment-avatar">{item.avatar ? <img src={item.avatar} alt="" /> : item.nickname.slice(0, 1)}</span><span><strong>{item.nickname}</strong><small>@{item.username}</small></span></Link>
        ))}</div></div>}
        {(searchQuery.data?.topics.length ?? 0) > 0 && <div className="search-section"><div className="eyebrow">论坛帖子</div><div>{searchQuery.data!.topics.map((topic) => (
          <Link className="search-topic" to={`/forum/${topic.id}`} key={topic.id}><span className={`forum-kind is-${topic.kind}`}>{topic.kind}</span><p>{topic.content.length > 120 ? topic.content.slice(0, 120) + '…' : topic.content}</p><small>@{topic.author.nickname} · {topic.repliesCount} 回复</small></Link>
        ))}</div></div>}
        {items.length > 0 && <div className="search-section"><div className="eyebrow">文章</div></div>}
        <div className="search-results">
          {items.map((item) => (
            <article className="search-result" key={item.id}>
              <h3><Link to={`/posts/${item.slug}`}><MarkedText text={item.titleMarked || item.title} /></Link></h3>
              <p className="search-result-snippet">{item.snippet ? <MarkedText text={item.snippet} /> : item.summary}</p>
              <div className="post-meta">
                <Link to={`/users/${item.author.username}`}>@{item.author.nickname}</Link>
                <span className="meta-dot" />
                <span><CalendarDays size={13} /> {formatDate(item.publishedAt)}</span>
                <span className="inline-meta"><Eye size={13} /> {item.views}</span>
                <span className="inline-meta"><Heart size={13} /> {item.likesCount}</span>
                <span className="inline-meta"><MessageCircle size={13} /> {item.commentsCount}</span>
              </div>
            </article>
          ))}
        </div></>}
      {totalPages > 1 && <div className="archive-pagination"><button className="icon-button" disabled={page <= 1} onClick={() => changePage(page - 1)} aria-label="上一页"><ChevronLeft size={17} /></button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page >= totalPages} onClick={() => changePage(page + 1)} aria-label="下一页"><ChevronRight size={17} /></button></div>}
    </section>
  )
}
