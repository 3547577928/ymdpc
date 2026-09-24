import { ArrowLeft, ArrowRight, Bookmark, CalendarDays, ChevronDown, ChevronRight, Clock3, Eye, Flag, Heart, MessageCircle, Pencil, Pin, Share2, UserPlus, UserRoundCheck, Trash2 } from 'lucide-react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useEffect, useMemo, useRef, useState } from 'react'
import { createComment, createReport, deleteComment, favoritePost, followUser, getComments, getPostBySlug, getUserProfile, likeComment, likePost, pinComment, recordPostView, sendCommentTyping, unfavoritePost, unfollowUser, unlikeComment, unlikePost, updateComment, type AdjacentPost, type SeriesDetail } from '../services/api'
import { useAuth } from '../services/auth'
import { ReadingProgress } from '../components/ReadingProgress'
import { CoverImage } from '../components/CoverImage'
import { ConfirmDialog, NoticeDialog, PromptDialog } from '../components/Dialog'
import { MarkdownContent } from '../components/MarkdownContent'
import type { PostComment, Post } from '../types'
import { formatDate } from '../utils'
import { extractMarkdownHeadings } from '../utils/markdown'
import { usePageMeta } from '../utils/usePageMeta'

// 评论树节点：顶层评论 + 挂在其下的回复（方案要求评论最多两层）
type CommentNode = { comment: PostComment; replies: PostComment[] }
type ReportRequest = { targetType: 'post' | 'comment'; targetId: number; title: string }

// 后端返回按时间排序的平铺列表，这里重建为两层结构：
// 回复的回复归到根评论下；父评论已删除的回复提升为顶层，避免丢失
function buildCommentTree(flat: PostComment[]): CommentNode[] {
  const byId = new Map(flat.map((item) => [item.id, item]))
  const nodes = new Map<number, CommentNode>(flat.map((item) => [item.id, { comment: item, replies: [] as PostComment[] }]))
  const roots: CommentNode[] = []
  for (const item of flat) {
    if (!item.parentId) {
      roots.push(nodes.get(item.id)!)
      continue
    }
    const parent = byId.get(item.parentId)
    // 回复的回复挂到根评论；根不存在时提升为顶层
    const root = parent?.parentId ? byId.get(parent.parentId) : parent
    const target = root ? nodes.get(root.id) : undefined
    if (target && target.comment.id !== item.id) target.replies.push(item)
    else roots.push(nodes.get(item.id)!)
  }
  return roots
}

export function PostPage() {
  const { slug } = useParams()
  const navigate = useNavigate()
  const [post, setPost] = useState<Post>()
  const [previous, setPrevious] = useState<AdjacentPost | null>(null)
  const [next, setNext] = useState<AdjacentPost | null>(null)
  const [series, setSeries] = useState<SeriesDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const { user } = useAuth()
  usePageMeta(post?.title, post?.summary)
  const [comments, setComments] = useState<PostComment[]>([])
  const [commentPage, setCommentPage] = useState(1)
  const [commentsTotal, setCommentsTotal] = useState(0)
  const [commentsLoading, setCommentsLoading] = useState(false)
  const [commentInput, setCommentInput] = useState('')
  const [typingUser, setTypingUser] = useState<string>()
  const typingSentAt = useRef(0)
  const [replyTo, setReplyTo] = useState<PostComment>()
  const [commentSaving, setCommentSaving] = useState(false)
  const [following, setFollowing] = useState(false)
  const [followBusy, setFollowBusy] = useState(false)
  const [reportRequest, setReportRequest] = useState<ReportRequest>()
  const [reportReason, setReportReason] = useState('')
  const [notice, setNotice] = useState<{ title: string; message: string }>()
  const [deleteRequest, setDeleteRequest] = useState<PostComment>()
  const [editingComment, setEditingComment] = useState<number>()
  const [editCommentInput, setEditCommentInput] = useState('')
  const [collapsedThreads, setCollapsedThreads] = useState<Set<number>>(new Set())
  const headings = useMemo(() => extractMarkdownHeadings(post?.content ?? ''), [post?.content])

  // 顶层评论分页加载：第一页替换，后续页追加；回复由后端随父评论一并返回
  const loadComments = async (postSlug: string, page: number) => {
    setCommentsLoading(true)
    try {
      const data = await getComments(postSlug, { page, pageSize: 20 })
      setComments((current) => page === 1 ? data.items : [...current, ...data.items])
      setCommentsTotal(data.total)
      setCommentPage(data.page)
    } catch {
      // 评论加载失败不阻塞正文阅读，保留已加载部分
    } finally {
      setCommentsLoading(false)
    }
  }

  // 实时评论流：新评论/删除/「正在输入」经 SSE 即时出现在评论区；
  // 自己的评论提交也会触发事件，按 id 去重
  const postSlug = post?.slug
  useEffect(() => {
    if (!postSlug) return
    const source = new EventSource(`/api/posts/${encodeURIComponent(postSlug)}/comments/stream`)
    source.addEventListener('comment', (event) => {
      try {
        const comment = JSON.parse((event as MessageEvent).data) as PostComment
        setComments((current) => current.some((item) => item.id === comment.id) ? current : [...current, comment])
        if (!comment.parentId) setCommentsTotal((total) => total + 1)
        setPost((current) => current ? { ...current, commentsCount: current.commentsCount + 1 } : current)
      } catch { /* 无法解析的事件忽略 */ }
    })
    source.addEventListener('comment_edited', (event) => {
      try {
        const updated = JSON.parse((event as MessageEvent).data) as PostComment
        setComments((current) => current.map((item) => item.id === updated.id ? { ...item, ...updated, liked: item.liked } : item))
      } catch { /* 无法解析的事件忽略 */ }
    })
    source.addEventListener('comment_deleted', (event) => {
      try {
        const { ids } = JSON.parse((event as MessageEvent).data) as { ids: number[] }
        setComments((current) => {
          const removedTopLevel = current.filter((item) => ids.includes(item.id) && !item.parentId).length
          setCommentsTotal((total) => Math.max(0, total - removedTopLevel))
          return current.filter((item) => !ids.includes(item.id))
        })
        setPost((current) => current ? { ...current, commentsCount: Math.max(0, current.commentsCount - ids.length) } : current)
      } catch { /* 无法解析的事件忽略 */ }
    })
    source.addEventListener('typing', (event) => {
      try {
        const { nickname } = JSON.parse((event as MessageEvent).data) as { nickname: string }
        setTypingUser(nickname)
      } catch { /* 无法解析的事件忽略 */ }
    })
    return () => source.close()
  }, [postSlug])

  // 「正在输入」提示 4 秒未续期即消失
  useEffect(() => {
    if (!typingUser) return
    const timer = window.setTimeout(() => setTypingUser(undefined), 4000)
    return () => window.clearTimeout(timer)
  }, [typingUser])

  useEffect(() => {
    if (!slug) return
    let active = true
    setLoading(true)
    setError('')
    getPostBySlug(slug)
      .then((postDetail) => {
        if (!active) return
        setPost(postDetail.post)
        setPrevious(postDetail.previous)
        setNext(postDetail.next)
        setSeries(postDetail.series ?? null)
        // 旧 slug 经 301 拿到文章后，地址栏同步为当前 slug（分享/书签才是有效链接）
        if (postDetail.post.slug !== slug) navigate(`/posts/${postDetail.post.slug}`, { replace: true })
        void loadComments(postDetail.post.slug, 1)
        void getUserProfile(postDetail.post.author.username).then((profile) => { if (active) setFollowing(profile.followingMe) }).catch(() => undefined)
        const viewKey = `qs:viewed:${postDetail.post.slug}`
        let shouldRecord = true
        try {
          shouldRecord = sessionStorage.getItem(viewKey) !== '1'
          if (shouldRecord) sessionStorage.setItem(viewKey, '1')
        } catch {
          // Browsers may disable session storage; counting still remains best-effort.
        }
        if (shouldRecord) {
          void recordPostView(postDetail.post.slug)
            .then(({ views }) => { if (active) setPost((current) => current ? { ...current, views } : current) })
            .catch(() => undefined)
        }
      })
      .catch((reason: Error) => { if (active) setError(reason.message) })
      .finally(() => { if (active) setLoading(false) })
    return () => { active = false }
  }, [slug, navigate])

  const toggleLike = async () => {
    if (!post || !user) return
    try {
      const result = post.liked ? await unlikePost(post.slug) : await likePost(post.slug)
      setPost({ ...post, liked: result.liked, likesCount: result.likesCount })
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '点赞失败')
    }
  }

  const toggleFollow = async () => {
    if (!post || !user || post.author.id === user.id) return
    setFollowBusy(true)
    try {
      const result = following ? await unfollowUser(post.author.id) : await followUser(post.author.id)
      setFollowing(result.following)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '关注失败')
    } finally {
      setFollowBusy(false)
    }
  }

  // 收藏或取消收藏文章
  const toggleFavorite = async () => {
    if (!post || !user) return
    try {
      const result = post.favorited ? await unfavoritePost(post.slug) : await favoritePost(post.slug)
      setPost({ ...post, favorited: result.favorited, favoriteCount: result.favoriteCount })
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '收藏操作失败')
    }
  }

  // 举报文章，打开统一输入弹窗
  const reportPost = () => {
    if (!post || !user) return
    setReportReason('')
    setReportRequest({ targetType: 'post', targetId: post.id, title: '举报文章' })
  }

  const submitReport = async () => {
    if (!reportRequest || !reportReason.trim()) return
    try {
      await createReport({ targetType: reportRequest.targetType, targetId: reportRequest.targetId, reason: reportReason.trim() })
      setReportRequest(undefined)
      setReportReason('')
      setNotice({ title: '举报已提交', message: '感谢你的反馈，管理员会在审核后处理。' })
    } catch (reason2) {
      setError(reason2 instanceof Error ? reason2.message : '举报提交失败')
    }
  }

  // 举报评论
  const reportComment = (comment: PostComment) => {
    if (!user) return
    setReportReason('')
    setReportRequest({ targetType: 'comment', targetId: comment.id, title: '举报评论' })
  }

  // 点赞或取消点赞评论，原地更新列表
  const toggleCommentLike = async (comment: PostComment) => {
    if (!user) return
    try {
      const result = comment.liked ? await unlikeComment(comment.id) : await likeComment(comment.id)
      setComments(comments.map((item) => item.id === comment.id ? { ...item, liked: result.liked, likesCount: result.likesCount } : item))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '点赞失败')
    }
  }

  const submitComment = async () => {
    if (!post || !commentInput.trim() || !user) return
    setCommentSaving(true)
    try {
      // 回复一条回复时，parentId 统一指向其根评论，保持两层结构
      const parentId = replyTo ? replyTo.parentId ?? replyTo.id : undefined
      const comment = await createComment(post.slug, { content: commentInput.trim(), parentId, replyToUserId: replyTo?.author.id })
      setComments([...comments, comment])
      if (!comment.parentId) setCommentsTotal((total) => total + 1)
      setCommentInput('')
      setReplyTo(undefined)
      setPost({ ...post, commentsCount: post.commentsCount + 1 })
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '发表评论失败')
    } finally {
      setCommentSaving(false)
    }
  }

  const removeComment = (comment: PostComment) => {
    setDeleteRequest(comment)
  }

  const confirmRemoveComment = async () => {
    if (!post || !deleteRequest) return
    const comment = deleteRequest
    setDeleteRequest(undefined)
    try {
      await deleteComment(comment.id)
      // 后端会把该评论下的回复一并删除，前端同步移除，避免刷新前残留孤儿评论
      const removedReplies = comments.filter((item) => item.parentId === comment.id)
      setComments(comments.filter((item) => item.id !== comment.id && item.parentId !== comment.id))
      if (!comment.parentId) setCommentsTotal((total) => Math.max(0, total - 1))
      setPost({ ...post, commentsCount: Math.max(0, post.commentsCount - 1 - removedReplies.length) })
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '删除评论失败')
    }
  }

  const togglePinned = async (comment: PostComment) => {
    try {
      const result = await pinComment(comment.id, !comment.pinned)
      setComments((current) => current.map((item) => item.id === comment.id ? { ...item, pinned: result.pinned } : item))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '更新置顶状态失败')
    }
  }

  const toggleThread = (commentID: number) => {
    setCollapsedThreads((current) => {
      const next = new Set(current)
      if (next.has(commentID)) next.delete(commentID)
      else next.add(commentID)
      return next
    })
  }

  if (loading) return <div className="container page-state"><span className="eyebrow">Loading article</span><h1>正在读取文章。</h1></div>
  if (error) return <div className="container page-state"><span className="eyebrow">Connection error</span><h1>文章暂时无法加载。</h1><p>{error}</p></div>

  if (!post) return <div className="container empty-state post-not-found"><h1>文章不存在</h1><Link className="text-button" to="/posts">返回文章列表 <ArrowRight size={15} /></Link></div>

  // 平铺评论重建为两层树后再渲染，保证回复显示在被回复的评论下面
  const commentTree = buildCommentTree(comments)
  const saveEditComment = async (comment: PostComment) => {
    if (!post || !editCommentInput.trim()) return
    try {
      const updated = await updateComment(comment.id, editCommentInput.trim(), post.slug)
      setComments((current) => current.map((item) => item.id === comment.id ? { ...item, ...updated, liked: item.liked } : item))
      setEditingComment(undefined)
    } catch (reason) {
      setNotice({ title: '编辑失败', message: reason instanceof Error ? reason.message : '编辑评论失败' })
    }
  }

  const renderComment = (comment: PostComment, isReply: boolean) => (
    <div className={`comment-item ${isReply ? 'is-reply' : ''}`} key={comment.id}>
      <div className="comment-avatar">{comment.author.avatar ? <img src={comment.author.avatar} alt="" /> : comment.author.nickname.slice(0, 1)}</div>
      <div className="comment-body">
        <div className="comment-meta"><Link to={`/users/${comment.author.username}`}>{comment.author.nickname}</Link><span>{formatDate(comment.createdAt)}</span>{comment.editedAt && <span className="comment-edited">已编辑</span>}{comment.pinned && <span className="comment-pinned"><Pin size={11} /> 置顶</span>}</div>
        {editingComment === comment.id
          ? <div className="comment-edit"><textarea value={editCommentInput} onChange={(event) => setEditCommentInput(event.target.value)} rows={3} maxLength={2000} autoFocus /><div className="comment-edit-actions"><button onClick={() => setEditingComment(undefined)}>取消</button><button className="button button-dark" onClick={() => void saveEditComment(comment)} disabled={!editCommentInput.trim()}>保存</button></div></div>
          : <p>{comment.content}</p>}
        <div className="comment-actions"><button onClick={() => setReplyTo(comment)}>回复</button><button className={comment.liked ? 'is-liked' : ''} onClick={() => user ? void toggleCommentLike(comment) : undefined}><Heart size={13} fill={comment.liked ? 'currentColor' : 'none'} /> {comment.likesCount}</button>{!isReply && user && (user.id === post.author.id || user.role === 'admin') && <button onClick={() => void togglePinned(comment)}><Pin size={13} /> {comment.pinned ? '取消置顶' : '置顶'}</button>}{user && user.id !== comment.author.id && <button onClick={() => void reportComment(comment)}><Flag size={13} /> 举报</button>}{user?.id === comment.author.id && <button onClick={() => { setEditingComment(comment.id); setEditCommentInput(comment.content) }}><Pencil size={13} /> 编辑</button>}{user?.id === comment.author.id && <button onClick={() => void removeComment(comment)}><Trash2 size={13} /> 删除</button>}</div>
      </div>
    </div>
  )

  return (
    <>
      <ReadingProgress />
      <article className="article-page">
      <header className="article-header container">
        <Link className="back-link" to="/posts"><ArrowLeft size={15} /> 返回文章列表</Link>
        <div className="article-kicker"><span className="eyebrow">{post.tags[0] ?? 'Note'}</span><span className="meta-dot" /><span>{formatDate(post.publishedAt)}</span></div>
        <h1>{post.title}</h1>
        <p className="article-summary">{post.summary}</p>
        <div className="article-meta"><span><CalendarDays size={15} /> {formatDate(post.publishedAt)}</span><span><Clock3 size={15} /> {post.readingTime} min read</span><span><Eye size={15} /> {post.views.toLocaleString()} views</span><button className={`icon-text-button ${post.liked ? 'is-liked' : ''}`} onClick={() => user ? void toggleLike() : undefined}><Heart size={15} fill={post.liked ? 'currentColor' : 'none'} /> {post.likesCount} 赞</button><button className={`icon-text-button ${post.favorited ? 'is-liked' : ''}`} onClick={() => user ? void toggleFavorite() : undefined}><Bookmark size={15} fill={post.favorited ? 'currentColor' : 'none'} /> {post.favoriteCount} 收藏</button><a className="icon-text-button" href="#comments"><MessageCircle size={15} /> {post.commentsCount} 评论</a><button className="icon-text-button" aria-label="分享文章" onClick={() => navigator.clipboard?.writeText(window.location.href)}><Share2 size={15} /> 分享</button>{user && (user.id === post.author.id || user.role === 'admin') && <Link className="icon-text-button" to={`/posts/${post.id}/edit`}><Pencil size={15} /> 编辑</Link>}{user && user.id !== post.author.id && <button className="icon-text-button" onClick={() => void reportPost()}><Flag size={15} /> 举报</button>}</div>
        <div className="article-author"><Link to={`/users/${post.author.username}`}><span className="mini-avatar">{post.author.avatar ? <img src={post.author.avatar} alt="" /> : post.author.nickname.slice(0, 1)}</span><span><strong>{post.author.nickname}</strong><small>@{post.author.username}</small></span></Link>{user && user.id !== post.author.id && <button className="button button-light follow-button" onClick={() => void toggleFollow()} disabled={followBusy}>{following ? <UserRoundCheck size={14} /> : <UserPlus size={14} />} {following ? '已关注作者' : '关注作者'}</button>}</div>
      </header>
      <div className="article-hero container"><CoverImage src={post.coverImage} title={post.title} /></div>
      <div className="article-body-wrap container">
        <aside className="article-aside"><div className="aside-label">On this page</div>{headings.length ? headings.map((heading) => <a className={`toc-level-${heading.level}`} href={`#${heading.id}`} key={heading.id}>{heading.text}</a>) : <a href="#article-content">正文内容</a>}</aside>
        <div id="article-content"><MarkdownContent className="article-content" content={post.content} /><div id="article-end" /></div>
      </div>
      {series && <div className="container series-box">
        <div className="series-box-head"><span className="eyebrow">Series</span><Link to={`/series/${series.slug}`}><strong>{series.title}</strong></Link>{series.description && <p>{series.description}</p>}</div>
        <ol>{series.posts.map((item, index) => <li key={item.slug} className={item.slug === post.slug ? 'is-current' : ''}><Link to={`/posts/${item.slug}`}><span>{String(index + 1).padStart(2, '0')}</span>{item.title}</Link></li>)}</ol>
      </div>}
      <div className="container article-nav">
        {previous ? <Link to={`/posts/${previous.slug}`} className="article-nav-item"><span><ArrowLeft size={15} /> 上一篇</span><strong>{previous.title}</strong></Link> : <span />}
        {next ? <Link to={`/posts/${next.slug}`} className="article-nav-item align-right"><span>下一篇 <ArrowRight size={15} /></span><strong>{next.title}</strong></Link> : <span />}
      </div>
      <section className="container comments-section" id="comments"><div className="comments-heading"><div><div className="eyebrow">Discussion</div><h2>评论 {post.commentsCount}</h2>{typingUser && <span className="comment-typing">{typingUser} 正在输入…</span>}</div>{!user && <Link className="text-button" to={`/login?from=${encodeURIComponent(`/posts/${post.slug}`)}`}>登录后参与 <ArrowRight size={15} /></Link>}</div>{user && <div className="comment-composer">{replyTo && <div className="replying">回复 @{replyTo.author.username} <button onClick={() => setReplyTo(undefined)}>取消</button></div>}<textarea value={commentInput} onChange={(event) => { setCommentInput(event.target.value); if (user && event.target.value.trim() && Date.now() - typingSentAt.current > 3000) { typingSentAt.current = Date.now(); void sendCommentTyping(post.slug).catch(() => undefined) } }} placeholder="写下你的看法" rows={4} /><button className="button button-dark" onClick={() => void submitComment()} disabled={commentSaving || !commentInput.trim()}><MessageCircle size={15} /> {commentSaving ? '发送中' : '发表评论'}</button></div>}<div className="comments-list">{commentTree.length ? commentTree.map((node) => { const collapsed = collapsedThreads.has(node.comment.id); return <div className="comment-thread" key={node.comment.id}>{renderComment(node.comment, false)}{node.replies.length > 0 && <><button className="comment-collapse" onClick={() => toggleThread(node.comment.id)}>{collapsed ? <ChevronRight size={14} /> : <ChevronDown size={14} />} {collapsed ? `展开 ${node.replies.length} 条回复` : `收起 ${node.replies.length} 条回复`}</button>{!collapsed && <div className="comment-replies">{node.replies.map((reply) => renderComment(reply, true))}</div>}</>}</div> }) : <div className="empty-state comments-empty"><h2>还没有评论</h2><p>成为第一个留下观点的人。</p></div>}</div>{commentTree.length < commentsTotal && <button className="text-button comments-load-more" onClick={() => void loadComments(post.slug, commentPage + 1)} disabled={commentsLoading}>{commentsLoading ? '正在读取。' : '加载更多评论（还有 ' + (commentsTotal - commentTree.length) + ' 条）'}</button>}</section>
      </article>
      <PromptDialog open={Boolean(reportRequest)} title={reportRequest?.title ?? ''} message="请输入举报理由，管理员审核后会处理。" value={reportReason} placeholder="例如：内容涉及广告、骚扰或违规信息" onChange={setReportReason} onSubmit={submitReport} onCancel={() => { setReportRequest(undefined); setReportReason('') }} />
      <ConfirmDialog open={Boolean(deleteRequest)} title="删除评论" message="确定删除这条评论吗？它下面的回复也会一并删除。" onCancel={() => setDeleteRequest(undefined)} onConfirm={confirmRemoveComment} />
      <NoticeDialog open={Boolean(notice)} title={notice?.title ?? ''} message={notice?.message} onCancel={() => setNotice(undefined)} />
    </>
  )
}
