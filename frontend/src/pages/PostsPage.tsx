import { Search, SlidersHorizontal, X } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { PostCard } from '../components/PostCard'
import { getPosts, getTags } from '../services/api'
import type { Post } from '../types'

export function PostsPage() {
  const [params, setParams] = useSearchParams()
  const [query, setQuery] = useState(params.get('q') ?? '')
  const [posts, setPosts] = useState<Post[]>([])
  const [allTags, setAllTags] = useState<string[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const selectedTag = params.get('tag') ?? '全部'

  useEffect(() => {
    let active = true
    setLoading(true)
    setError('')
    Promise.all([
      getPosts({ q: query, tag: selectedTag === '全部' ? undefined : selectedTag, pageSize: 50 }),
      getTags(),
    ]).then(([postPage, tags]) => {
      if (!active) return
      setPosts(postPage.items)
      setTotal(postPage.total)
      setAllTags(tags.map((tag) => tag.name))
    }).catch((reason: Error) => {
      if (active) setError(reason.message)
    }).finally(() => {
      if (active) setLoading(false)
    })
    return () => { active = false }
  }, [query, selectedTag])

  const changeTag = (tag: string) => {
    const next = new URLSearchParams(params)
    if (tag === '全部') next.delete('tag')
    else next.set('tag', tag)
    setParams(next)
  }

  const updateQuery = (value: string) => {
    setQuery(value)
    const next = new URLSearchParams(params)
    if (value) next.set('q', value)
    else next.delete('q')
    setParams(next, { replace: true })
  }

  return (
    <section className="container archive-page">
      <div className="archive-header">
        <div><div className="eyebrow">Archive / 2026</div><h1>所有文章</h1><p>关于设计、工程、阅读，以及把事情做得更好的过程记录。</p></div>
        <div className="archive-count"><strong>{String(total).padStart(2, '0')}</strong><span>篇公开文章</span></div>
      </div>

      <div className="archive-toolbar">
        <label className="search-box"><Search size={17} /><input value={query} onChange={(event) => updateQuery(event.target.value)} placeholder="搜索标题或关键词" /><kbd>⌘ K</kbd></label>
        <div className="toolbar-label"><SlidersHorizontal size={15} /> 筛选主题</div>
      </div>
      <div className="filter-row">
        {['全部', ...allTags].map((tag) => <button key={tag} className={`filter-chip ${selectedTag === tag ? 'is-active' : ''}`} onClick={() => changeTag(tag)}>{tag}</button>)}
        {(query || selectedTag !== '全部') && <button className="clear-filter" onClick={() => { setQuery(''); setParams({}) }}><X size={14} /> 清除筛选</button>}
      </div>

      {loading ? <div className="empty-state"><h2>正在加载文章</h2><p>正在从 SQLite 读取公开内容。</p></div> : error ? <div className="empty-state"><h2>文章加载失败</h2><p>{error}</p></div> : posts.length ? <div className="archive-grid">{posts.map((post) => <PostCard key={post.id} post={post} />)}</div> : <div className="empty-state"><h2>没有找到相关内容</h2><p>换一个关键词或清除筛选条件。</p></div>}
    </section>
  )
}
