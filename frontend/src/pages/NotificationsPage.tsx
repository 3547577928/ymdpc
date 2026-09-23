import { ArrowLeft, Bell, CheckCheck, Heart, MessageCircle, Newspaper, Reply, UserPlus } from 'lucide-react'
import { useCallback, useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { getNotifications, markAllNotificationsRead, markNotificationRead } from '../services/api'
import type { NotificationItem } from '../types'
import { formatDate } from '../utils'

const typeMeta: Record<NotificationItem['type'], { icon: typeof Bell; text: (item: NotificationItem) => string }> = {
  comment: { icon: MessageCircle, text: (item) => `评论了你的文章《${item.resourceTitle ?? ''}》` },
  reply: { icon: Reply, text: (item) => `回复了你的评论${item.resourceTitle ? `（${item.resourceTitle}）` : ''}` },
  like: { icon: Heart, text: (item) => `赞了你的文章《${item.resourceTitle ?? ''}》` },
  follow: { icon: UserPlus, text: () => '关注了你' },
  post: { icon: Newspaper, text: (item) => `发布了新文章《${item.resourceTitle ?? ''}》` },
}

function groupNotificationItems(items: NotificationItem[]) {
  const groups: NotificationItem[] = []
  const byKey = new Map<string, NotificationItem>()
  for (const item of items) {
    const key = `${item.type}:${item.resourceId}:${item.actor.id}`
    const existing = byKey.get(key)
    if (!existing) {
      const group = { ...item, groupCount: 1, groupedIds: [item.id] }
      byKey.set(key, group)
      groups.push(group)
    } else {
      existing.groupCount = (existing.groupCount ?? 1) + 1
      existing.groupedIds = [...(existing.groupedIds ?? [existing.id]), item.id]
      existing.read = existing.read && item.read
    }
  }
  return groups
}

// 通知中心：展示评论、回复、点赞、关注四类通知，点击跳转并标记已读
export function NotificationsPage() {
  const navigate = useNavigate()
  const [items, setItems] = useState<NotificationItem[]>([])
  const [unread, setUnread] = useState(0)
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const pageSize = 20

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getNotifications({ page, pageSize })
      setItems(groupNotificationItems(data.items))
      setUnread(data.unread)
      setTotal(data.total)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取通知失败')
    } finally {
      setLoading(false)
    }
  }, [page])

  const markAllRead = async () => {
    try {
      await markAllNotificationsRead()
      setItems((current) => current.map((item) => ({ ...item, read: true })))
      setUnread(0)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '更新通知失败')
    }
  }

  useEffect(() => {
    void load()
  }, [load])

  // 点击通知：标记已读并跳转到对应内容
  const open = async (item: NotificationItem) => {
    if (!item.read) {
      await Promise.all((item.groupedIds ?? [item.id]).map((id) => markNotificationRead(id).catch(() => undefined)))
    }
    if (item.type === 'follow') navigate(`/users/${item.actor.username}`)
    else if (item.resourceSlug) navigate(`/posts/${item.resourceSlug}${item.type === 'post' ? '' : '#comments'}`)
    else void load()
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return <section className="container my-posts-page">
    <div className="write-topbar"><Link className="back-link" to="/"><ArrowLeft size={15} /> 返回社区</Link><span className="eyebrow">Notifications</span></div>
    <div className="notifications-heading"><h1>通知中心 {unread > 0 && <small className="unread-badge">{unread} 条未读</small>}</h1>{unread > 0 && <button className="text-button" onClick={() => void markAllRead()}><CheckCheck size={15} /> 全部已读</button>}</div>
    {error && <div className="form-error">{error}</div>}
    {loading ? <div className="page-state"><h1>正在读取。</h1></div> : items.length ? <div className="notifications-list">
      {items.map((item) => {
        const meta = typeMeta[item.type]
        const Icon = meta.icon
        return (
          <button className={`notification-item ${item.read ? '' : 'is-unread'}`} key={item.id} onClick={() => void open(item)}>
            <span className="notification-icon"><Icon size={16} /></span>
            <span className="notification-body">
              <strong>{item.actor.nickname}</strong> {meta.text(item)}{(item.groupCount ?? 1) > 1 && <span> 等 {item.groupCount} 次</span>}
              <small>{formatDate(item.createdAt)}</small>
            </span>
            {!item.read && <span className="notification-dot" aria-label="未读" />}
          </button>
        )
      })}
    </div> : <div className="empty-state comments-empty"><h2>还没有通知</h2><p>关注、互动和作者更新会出现在这里。</p></div>}
    {totalPages > 1 && <div className="admin-pagination"><button className="icon-button" disabled={page === 1} onClick={() => setPage((current) => current - 1)}>上一页</button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page >= totalPages} onClick={() => setPage((current) => current + 1)}>下一页</button></div>}
  </section>
}
