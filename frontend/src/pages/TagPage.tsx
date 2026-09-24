import { Bell, BellRing, ChevronLeft, ChevronRight } from 'lucide-react'
import { useState } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { PostCard } from '../components/PostCard'
import { SkeletonPostGrid } from '../components/Skeleton'
import { subscribeTag, unsubscribeTag } from '../services/api'
import { useAuth } from '../services/auth'
import { useFeed, useTag } from '../services/queries'
import { usePageMeta } from '../utils/usePageMeta'

const pageSize = 12

// 标签落地页：聚合该标签下的公开文章，登录用户可订阅（新文章站内通知）
export function TagPage() {
  const { slug = '' } = useParams()
  const [params, setParams] = useSearchParams()
  const page = Math.max(1, Number(params.get('page')) || 1)
  const { user } = useAuth()
  const queryClient = useQueryClient()
  const tagQuery = useTag(slug)
  const tag = tagQuery.data
  usePageMeta(tag ? `标签 · ${tag.name}` : '标签')
  const feedQuery = useFeed({ tag: slug, page, pageSize })
  const [busy, setBusy] = useState(false)

  const posts = feedQuery.data?.items ?? []
  const total = feedQuery.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const toggleSubscribe = async () => {
    if (!tag || busy) return
    setBusy(true)
    try {
      const result = tag.subscribed ? await unsubscribeTag(tag.id) : await subscribeTag(tag.id)
      queryClient.setQueryData(['tag', slug], { ...tag, subscribed: result.subscribed })
    } catch { /* 失败保持原状态 */ } finally {
      setBusy(false)
    }
  }

  const changePage = (nextPage: number) => {
    const next = new URLSearchParams(params)
    if (nextPage <= 1) next.delete('page')
    else next.set('page', String(nextPage))
    setParams(next)
    window.scrollTo({ top: 0, behavior: 'smooth' })
  }

  if (tagQuery.isLoading) return <div className="container" style={{ paddingTop: 40 }}><SkeletonPostGrid count={3} /></div>
  if (!tag) return <div className="container page-state"><span className="eyebrow">Tag</span><h1>标签不存在。</h1><p><Link className="text-button" to="/posts">回到文章列表</Link></p></div>

  return (
    <section className="container tag-page">
      <header className="forum-header">
        <div><span className="eyebrow">Tag</span><h1># {tag.name}</h1><p>该标签下共 {tag.postCount} 篇公开文章。</p></div>
        {user
          ? <button className="button button-light" onClick={() => void toggleSubscribe()} disabled={busy}>{tag.subscribed ? <><BellRing size={15} /> 已订阅</> : <><Bell size={15} /> 订阅标签</>}</button>
          : <Link className="button button-light" to={`/login?from=${encodeURIComponent(`/tags/${tag.slug}`)}`}><Bell size={15} /> 登录后订阅</Link>}
      </header>
      {feedQuery.isLoading ? <SkeletonPostGrid count={6} />
        : posts.length ? <div className="archive-grid">{posts.map((post) => <PostCard key={post.id} post={post} />)}</div>
        : <div className="empty-state"><h2>该标签下还没有文章</h2><p>写一篇文章，成为第一个使用这个标签的人。</p></div>}
      {totalPages > 1 && <div className="archive-pagination"><button className="icon-button" disabled={page <= 1} onClick={() => changePage(page - 1)} aria-label="上一页"><ChevronLeft size={17} /></button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page >= totalPages} onClick={() => changePage(page + 1)} aria-label="下一页"><ChevronRight size={17} /></button></div>}
    </section>
  )
}
