import { ArrowLeft, Flag, Heart, MessageCircle, Reply, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { ConfirmDialog, NoticeDialog, PromptDialog } from '../components/Dialog'
import { createForumReply, createReport, deleteForumReply, deleteForumTopic, getForumTopic, likeForumReply, likeForumTopic, unlikeForumReply, unlikeForumTopic } from '../services/api'
import { useAuth } from '../services/auth'
import { usePageMeta } from '../utils/usePageMeta'
import type { ForumReply, ForumTopic } from '../types'
import { formatDate } from '../utils'

const kindLabels = { discuss: '讨论', share: '分享', help: '求助', rant: '吐槽' } as const
type ReplyNode = { reply: ForumReply; children: ForumReply[] }
type ReportRequest = { targetType: 'forum_topic' | 'forum_reply'; targetId: number; title: string }

function replyTree(replies: ForumReply[]) {
  const nodes = new Map<number, ReplyNode>(replies.map((reply) => [reply.id, { reply, children: [] }]))
  const roots: ReplyNode[] = []
  for (const reply of replies) {
    if (!reply.parentId) { roots.push(nodes.get(reply.id)!); continue }
    const parent = nodes.get(reply.parentId)
    if (parent) parent.children.push(reply); else roots.push(nodes.get(reply.id)!)
  }
  return roots
}

export function ForumTopicPage() {
  const navigate = useNavigate()
  const { id = '' } = useParams()
  const topicID = Number(id)
  const [topic, setTopic] = useState<ForumTopic>()
  const [replies, setReplies] = useState<ForumReply[]>([])
  const [replyPage, setReplyPage] = useState(1)
  const [repliesTotal, setRepliesTotal] = useState(0)
  const [repliesLoading, setRepliesLoading] = useState(false)
  const { user } = useAuth()
  usePageMeta(topic ? topic.content.slice(0, 40) : '帖子详情')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [content, setContent] = useState('')
  const [replyTo, setReplyTo] = useState<ForumReply>()
  const [saving, setSaving] = useState(false)
  const [deleteReplyRequest, setDeleteReplyRequest] = useState<ForumReply>()
  const [deleteTopicRequest, setDeleteTopicRequest] = useState(false)
  const [reportRequest, setReportRequest] = useState<ReportRequest>()
  const [reportReason, setReportReason] = useState('')
  const [notice, setNotice] = useState<{ title: string; message: string }>()

  useEffect(() => {
    if (!topicID) { setError('帖子编号无效'); setLoading(false); return }
    getForumTopic(topicID, { replyPage: 1, replyPageSize: 20 }).then((detail) => {
      setTopic(detail.topic); setReplies(detail.replies); setRepliesTotal(detail.repliesTotal); setReplyPage(1)
    }).catch((reason: Error) => setError(reason.message)).finally(() => setLoading(false))
  }, [topicID])

  // 顶层回复分页加载更多，子回复由后端随父回复一并返回
  const loadMoreReplies = async () => {
    setRepliesLoading(true)
    try {
      const detail = await getForumTopic(topicID, { replyPage: replyPage + 1, replyPageSize: 20 })
      setReplies((current) => [...current, ...detail.replies]); setRepliesTotal(detail.repliesTotal); setReplyPage(replyPage + 1)
    } catch (reason) { setError(reason instanceof Error ? reason.message : '读取回复失败') } finally { setRepliesLoading(false) }
  }

  const toggleTopicLike = async () => {
    if (!topic || !user) return
    try { const result = topic.liked ? await unlikeForumTopic(topic.id) : await likeForumTopic(topic.id); setTopic({ ...topic, ...result }) } catch (reason) { setError(reason instanceof Error ? reason.message : '点赞失败') }
  }
  const toggleReplyLike = async (reply: ForumReply) => {
    if (!user) return
    try { const result = reply.liked ? await unlikeForumReply(reply.id) : await likeForumReply(reply.id); setReplies((current) => current.map((item) => item.id === reply.id ? { ...item, ...result } : item)) } catch (reason) { setError(reason instanceof Error ? reason.message : '点赞失败') }
  }
  const submitReply = async () => {
    if (!topic || !user || !content.trim()) return
    setSaving(true)
    try {
      const parentId = replyTo ? replyTo.parentId ?? replyTo.id : undefined
      const created = await createForumReply(topic.id, { content: content.trim(), parentId, replyToUserId: replyTo?.author.id })
      setReplies((current) => [...current, created]); if (!created.parentId) setRepliesTotal((total) => total + 1); setTopic({ ...topic, repliesCount: topic.repliesCount + 1 }); setContent(''); setReplyTo(undefined)
    } catch (reason) { setError(reason instanceof Error ? reason.message : '发表回复失败') } finally { setSaving(false) }
  }
  const confirmDeleteReply = async () => {
    if (!topic || !deleteReplyRequest) return
    const target = deleteReplyRequest; setDeleteReplyRequest(undefined)
    try {
      await deleteForumReply(target.id)
      const removed = replies.filter((item) => item.id === target.id || item.parentId === target.id)
      setReplies((current) => current.filter((item) => item.id !== target.id && item.parentId !== target.id))
      if (!target.parentId) setRepliesTotal((total) => Math.max(0, total - 1))
      setTopic({ ...topic, repliesCount: Math.max(0, topic.repliesCount - removed.length) })
    } catch (reason) { setError(reason instanceof Error ? reason.message : '删除回复失败') }
  }
  const confirmDeleteTopic = async () => {
    if (!topic) return
    try { await deleteForumTopic(topic.id); navigate('/forum') } catch (reason) { setError(reason instanceof Error ? reason.message : '删除帖子失败') }
  }
  const submitReport = async () => {
    if (!reportRequest || !reportReason.trim()) return
    try { await createReport({ targetType: reportRequest.targetType, targetId: reportRequest.targetId, reason: reportReason.trim() }); setReportRequest(undefined); setReportReason(''); setNotice({ title: '举报已提交', message: '管理员会核对帖子上下文并处理。' }) } catch (reason) { setError(reason instanceof Error ? reason.message : '举报失败') }
  }

  if (loading) return <div className="container page-state"><span className="eyebrow">Forum</span><h1>正在读取帖子。</h1></div>
  if (!topic) return <div className="container page-state"><span className="eyebrow">Forum</span><h1>帖子暂时无法加载。</h1><p>{error}</p></div>
  const tree = replyTree(replies)
  const renderReply = (reply: ForumReply, child = false) => <div className={`forum-reply ${child ? 'is-child' : ''}`} key={reply.id}><Link className="forum-avatar" to={`/users/${reply.author.username}`}>{reply.author.avatar ? <img src={reply.author.avatar} alt="" /> : reply.author.nickname.slice(0, 1)}</Link><div className="forum-reply-body"><div className="forum-reply-meta"><Link to={`/users/${reply.author.username}`}>{reply.author.nickname}</Link><span>{formatDate(reply.createdAt)}</span></div><p>{reply.content}</p><div className="forum-reply-actions"><button onClick={() => setReplyTo(reply)}><Reply size={13} />回复</button><button className={reply.liked ? 'is-liked' : ''} onClick={() => void toggleReplyLike(reply)}><Heart size={13} fill={reply.liked ? 'currentColor' : 'none'} />{reply.likesCount}</button>{user && user.id !== reply.author.id && <button onClick={() => { setReportReason(''); setReportRequest({ targetType: 'forum_reply', targetId: reply.id, title: '举报回复' }) }}><Flag size={13} />举报</button>}{user && (user.id === reply.author.id || user.role === 'admin') && <button onClick={() => setDeleteReplyRequest(reply)}><Trash2 size={13} />删除</button>}</div></div></div>

  return <section className="container forum-topic-page"><Link className="back-link" to="/forum"><ArrowLeft size={15} />返回论坛</Link>{error && <div className="form-error forum-error">{error}</div>}<article className="forum-topic-detail"><header className="forum-topic-author"><Link className="forum-avatar" to={`/users/${topic.author.username}`}>{topic.author.avatar ? <img src={topic.author.avatar} alt="" /> : topic.author.nickname.slice(0, 1)}</Link><div><Link to={`/users/${topic.author.username}`}>{topic.author.nickname}</Link><small>@{topic.author.username} · {formatDate(topic.createdAt)}</small></div><span className={`forum-kind is-${topic.kind}`}>{kindLabels[topic.kind]}</span></header><p className="forum-topic-content">{topic.content}</p>{topic.images.length > 0 && <div className={`forum-detail-images count-${Math.min(topic.images.length, 3)}`}>{topic.images.map((image, index) => <a href={image} target="_blank" rel="noreferrer" key={image}><img src={image} alt={`帖子图片 ${index + 1}`} /></a>)}</div>}<footer className="forum-topic-actions"><button className={topic.liked ? 'is-liked' : ''} onClick={() => void toggleTopicLike()}><Heart size={15} fill={topic.liked ? 'currentColor' : 'none'} />{topic.likesCount} 赞</button><a href="#forum-replies"><MessageCircle size={15} />{topic.repliesCount} 回复</a>{user && user.id !== topic.author.id && <button onClick={() => { setReportReason(''); setReportRequest({ targetType: 'forum_topic', targetId: topic.id, title: '举报帖子' }) }}><Flag size={14} />举报</button>}{user && (user.id === topic.author.id || user.role === 'admin') && <button onClick={() => setDeleteTopicRequest(true)}><Trash2 size={14} />删除帖子</button>}</footer></article><section className="forum-replies-section" id="forum-replies"><div className="comments-heading"><div><span className="eyebrow">Replies</span><h2>回复 {topic.repliesCount}</h2></div>{!user && <Link className="text-button" to={`/login?from=${encodeURIComponent(`/forum/${topic.id}`)}`}>登录后回复</Link>}</div>{user && <div className="forum-reply-composer">{replyTo && <div className="replying">回复 @{replyTo.author.username}<button onClick={() => setReplyTo(undefined)}>取消</button></div>}<textarea value={content} maxLength={1000} onChange={(event) => setContent(event.target.value)} rows={4} placeholder="回复这个帖子" /><div><span>{content.length}/1000</span><button className="button button-dark" onClick={() => void submitReply()} disabled={saving || !content.trim()}><MessageCircle size={15} />{saving ? '发送中' : '发表回复'}</button></div></div>}<div className="forum-reply-list">{tree.length ? tree.map((node) => <div className="forum-reply-thread" key={node.reply.id}>{renderReply(node.reply)}{node.children.map((child) => renderReply(child, true))}</div>) : <div className="empty-state comments-empty"><h2>还没有回复</h2><p>成为第一个参与讨论的人。</p></div>}</div>{tree.length < repliesTotal && <button className="text-button comments-load-more" onClick={() => void loadMoreReplies()} disabled={repliesLoading}>{repliesLoading ? '正在读取。' : '加载更多回复（还有 ' + (repliesTotal - tree.length) + ' 条）'}</button>}</section><ConfirmDialog open={deleteTopicRequest} title="删除帖子" message="确定删除这个帖子吗？所有回复也会一并删除。" onCancel={() => setDeleteTopicRequest(false)} onConfirm={confirmDeleteTopic} /><ConfirmDialog open={Boolean(deleteReplyRequest)} title="删除回复" message="确定删除这条回复吗？它下面的回复也会一并删除。" onCancel={() => setDeleteReplyRequest(undefined)} onConfirm={confirmDeleteReply} /><PromptDialog open={Boolean(reportRequest)} title={reportRequest?.title ?? ''} message="请输入举报理由，管理员审核后会处理。" value={reportReason} placeholder="例如：广告、骚扰或违规内容" onChange={setReportReason} onSubmit={submitReport} onCancel={() => { setReportRequest(undefined); setReportReason('') }} /><NoticeDialog open={Boolean(notice)} title={notice?.title ?? ''} message={notice?.message} onCancel={() => setNotice(undefined)} /></section>
}
