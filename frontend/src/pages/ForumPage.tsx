import { ChevronLeft, ChevronRight, Heart, ImagePlus, MessageCircle, Search, Send, X } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { createForumTopic, likeForumTopic, unlikeForumTopic, uploadImage, type AuthUser, type ForumTopicPage as ForumTopicPageData } from '../services/api'
import { useForumTopics, type ForumTopicsParams } from '../services/queries'
import { SkeletonList } from '../components/Skeleton'
import { useAuth } from '../services/auth'
import { usePageMeta } from '../utils/usePageMeta'
import type { ForumKind, ForumTopic } from '../types'
import { formatDate } from '../utils'

const kindLabels: Record<ForumKind, string> = { discuss: '讨论', share: '分享', help: '求助', rant: '吐槽' }
const kinds: ForumKind[] = ['discuss', 'share', 'help', 'rant']

function TopicItem({ topic, user, onLike }: { topic: ForumTopic; user?: AuthUser; onLike: (topic: ForumTopic) => void }) {
  return <article className="forum-topic-item">
    <header className="forum-topic-author">
      <Link className="forum-avatar" to={`/users/${topic.author.username}`}>{topic.author.avatar ? <img src={topic.author.avatar} alt="" /> : topic.author.nickname.slice(0, 1)}</Link>
      <div><Link to={`/users/${topic.author.username}`}>{topic.author.nickname}</Link><small>@{topic.author.username} · {formatDate(topic.createdAt)}</small></div>
      <span className={`forum-kind is-${topic.kind}`}>{kindLabels[topic.kind]}</span>
    </header>
    <Link className="forum-topic-link" to={`/forum/${topic.id}`}><p>{topic.content}</p></Link>
    {topic.images.length > 0 && <Link className={`forum-image-grid count-${Math.min(topic.images.length, 3)}`} to={`/forum/${topic.id}`}>{topic.images.slice(0, 3).map((image, index) => <span key={image}><img src={image} alt={`帖子图片 ${index + 1}`} loading="lazy" />{index === 2 && topic.images.length > 3 && <b>+{topic.images.length - 3}</b>}</span>)}</Link>}
    <footer className="forum-topic-actions"><button className={topic.liked ? 'is-liked' : ''} onClick={() => user && onLike(topic)}><Heart size={15} fill={topic.liked ? 'currentColor' : 'none'} /> {topic.likesCount}</button><Link to={`/forum/${topic.id}`}><MessageCircle size={15} /> {topic.repliesCount} 回复</Link></footer>
  </article>
}

export function ForumPage() {
  usePageMeta('论坛')
  const pageSize = 15
  const [params, setParams] = useSearchParams()
  const page = Math.max(1, Number(params.get('page')) || 1)
  const mode = params.get('mode') === 'hot' ? 'hot' : 'latest'
  const kind = (params.get('kind') ?? 'all') as ForumKind | 'all'
  const query = params.get('q') ?? ''
  const [queryInput, setQueryInput] = useState(query)
  const { user } = useAuth()
  const [error, setError] = useState('')
  const [content, setContent] = useState('')
  const [composerKind, setComposerKind] = useState<ForumKind>('discuss')
  const [images, setImages] = useState<string[]>([])
  const [publishing, setPublishing] = useState(false)
  const [uploading, setUploading] = useState(false)
  // 列表数据由 React Query 管理：参数即缓存键，发布/点赞通过失效或就地更新同步
  const topicsParams = useMemo<ForumTopicsParams>(() => ({ mode, kind, q: query, page, pageSize }), [mode, kind, query, page])
  const queryClient = useQueryClient()
  const topicsQuery = useForumTopics(topicsParams)
  const topics = topicsQuery.data?.items ?? []
  const total = topicsQuery.data?.total ?? 0
  const loading = topicsQuery.isLoading
  const queryError = topicsQuery.error?.message ?? ''

  useEffect(() => { setQueryInput(query) }, [query])
  useEffect(() => {
    if (queryInput === query) return
    const timer = window.setTimeout(() => {
      const next = new URLSearchParams(params)
      if (queryInput.trim()) next.set('q', queryInput.trim()); else next.delete('q')
      next.delete('page')
      setParams(next, { replace: true })
    }, 300)
    return () => window.clearTimeout(timer)
  }, [params, query, queryInput, setParams])

  // 论坛新帖实时流：仅在「最新」第一页、无搜索、分类匹配时插入，
  // 避免破坏分页与筛选结果
  useEffect(() => {
    const source = new EventSource('/api/forum/stream')
    source.addEventListener('topic', (event) => {
      try {
        const topic = JSON.parse((event as MessageEvent).data) as ForumTopic
        if (mode !== 'latest' || page !== 1 || query) return
        if (kind !== 'all' && kind !== topic.kind) return
        queryClient.setQueryData<ForumTopicPageData>(['forum-topics', topicsParams], (current) => current && !current.items.some((item) => item.id === topic.id) ? { ...current, total: current.total + 1, items: [topic, ...current.items].slice(0, pageSize) } : current)
      } catch { /* 无法解析的事件忽略 */ }
    })
    return () => source.close()
  }, [mode, page, query, kind, queryClient, topicsParams])

  const changeParam = (key: string, value?: string) => {
    const next = new URLSearchParams(params)
    if (value && value !== 'all') next.set(key, value); else next.delete(key)
    if (key !== 'page') next.delete('page')
    setParams(next)
  }

  const publish = async () => {
    if (!content.trim() || !user) return
    setPublishing(true)
    setError('')
    try {
      await createForumTopic({ content: content.trim(), kind: composerKind, images })
      setContent('')
      setImages([])
      setParams({})
      // 失效所有帖子列表缓存，回到第一页时拿到包含新帖的数据
      void queryClient.invalidateQueries({ queryKey: ['forum-topics'] })
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '发布帖子失败')
    } finally {
      setPublishing(false)
    }
  }

  const selectImages = async (files: FileList | null) => {
    if (!files || images.length >= 9) return
    const selected = Array.from(files).slice(0, 9 - images.length)
    setUploading(true)
    setError('')
    try {
      const uploaded = await Promise.all(selected.map((file) => uploadImage(file)))
      setImages((current) => [...current, ...uploaded.map((item) => item.url)].slice(0, 9))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '上传图片失败')
    } finally {
      setUploading(false)
    }
  }

  const toggleLike = async (topic: ForumTopic) => {
    try {
      const result = topic.liked ? await unlikeForumTopic(topic.id) : await likeForumTopic(topic.id)
      // 就地更新当前页缓存，无需整页重拉
      queryClient.setQueryData<ForumTopicPageData>(['forum-topics', topicsParams], (current) => current ? { ...current, items: current.items.map((item) => item.id === topic.id ? { ...item, ...result } : item) } : current)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '点赞失败')
    }
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))
  return <section className="container forum-page">
    <header className="forum-header"><div><span className="eyebrow">Community / Forum</span><h1>论坛</h1><p>讨论、分享、求助，或者只是吐槽一下。</p></div><div className="archive-count"><strong>{total.toLocaleString()}</strong><span>条社区帖子</span></div></header>
    {user ? <div className="forum-composer"><div className="forum-composer-head"><span className="forum-avatar">{user.avatar ? <img src={user.avatar} alt="" /> : user.nickname.slice(0, 1)}</span><strong>{user.nickname}</strong></div><textarea value={content} onChange={(event) => setContent(event.target.value)} maxLength={2000} rows={4} placeholder="有什么想和社区聊聊的？" />{images.length > 0 && <div className="forum-composer-images">{images.map((image) => <span key={image}><img src={image} alt="待发布图片" /><button onClick={() => setImages((current) => current.filter((item) => item !== image))} aria-label="移除图片"><X size={14} /></button></span>)}</div>}<div className="forum-composer-footer"><div className="forum-kind-selector">{kinds.map((item) => <button key={item} className={composerKind === item ? 'is-active' : ''} onClick={() => setComposerKind(item)}>{kindLabels[item]}</button>)}</div><div className="forum-publish-actions"><label className="icon-button forum-upload-button" aria-label="上传图片"><ImagePlus size={17} /><input type="file" accept="image/jpeg,image/png,image/gif,image/webp" multiple disabled={uploading || images.length >= 9} onChange={(event) => { void selectImages(event.target.files); event.target.value = '' }} /></label><span>{content.length}/2000</span><button className="button button-dark" onClick={() => void publish()} disabled={publishing || uploading || !content.trim()}><Send size={15} />{publishing ? '发布中' : '发布'}</button></div></div></div> : <div className="forum-login-prompt"><span>登录后可以发布帖子、回复和参与讨论。</span><Link className="button button-dark" to="/login">登录参与</Link></div>}
    {(error || queryError) && <div className="form-error forum-error">{error || queryError}</div>}
    <div className="forum-toolbar"><label className="search-box"><Search size={16} /><input value={queryInput} onChange={(event) => setQueryInput(event.target.value)} placeholder="搜索帖子内容" /></label><div className="feed-tabs"><button className={mode === 'latest' ? 'is-active' : ''} onClick={() => changeParam('mode', 'latest')}>最新</button><button className={mode === 'hot' ? 'is-active' : ''} onClick={() => changeParam('mode', 'hot')}>热门</button></div></div>
    <div className="forum-kind-filter"><button className={kind === 'all' ? 'is-active' : ''} onClick={() => changeParam('kind')}>全部</button>{kinds.map((item) => <button key={item} className={kind === item ? 'is-active' : ''} onClick={() => changeParam('kind', item)}>{kindLabels[item]}</button>)}</div>
    {loading ? <SkeletonList count={6} /> : topics.length ? <div className="forum-feed">{topics.map((topic) => <TopicItem key={topic.id} topic={topic} user={user} onLike={(item) => void toggleLike(item)} />)}</div> : <div className="empty-state"><h2>还没有相关帖子</h2><p>换个分类，或者发布第一条讨论。</p></div>}
    {totalPages > 1 && <div className="archive-pagination"><button className="icon-button" disabled={page <= 1} onClick={() => changeParam('page', String(page - 1))} aria-label="上一页"><ChevronLeft size={17} /></button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page >= totalPages} onClick={() => changeParam('page', String(page + 1))} aria-label="下一页"><ChevronRight size={17} /></button></div>}
  </section>
}
