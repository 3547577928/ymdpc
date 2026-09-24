import { ArrowLeft, Bell, CheckCheck, Heart, MessageCircle, Newspaper, Reply, UserPlus } from 'lucide-react'
import { useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useQueryClient } from '@tanstack/react-query'
import { markAllNotificationsRead, markNotificationRead } from '../services/api'
import { UNREAD_QUERY_KEY, useNotificationList, type NotificationFilter } from '../services/queries'
import { SkeletonList } from '../components/Skeleton'
import type { NotificationItem } from '../types'
import { formatDate } from '../utils'
import { usePageMeta } from '../utils/usePageMeta'

const typeMeta: Record<NotificationItem['type'], { icon: typeof Bell; text: (item: NotificationItem) => string }> = {
  comment: { icon: MessageCircle, text: (item) => `评论了你的文章《${item.resourceTitle ?? ''}》` },
  reply: { icon: Reply, text: (item) => `回复了你的评论${item.resourceTitle ? `（${item.resourceTitle}）` : ''}` },
  like: { icon: Heart, text: (item) => `赞了你的文章《${item.resourceTitle ?? ''}》` },
  follow: { icon: UserPlus, text: () => '关注了你' },
  post: { icon: Newspaper, text: (item) => `发布了新文章《${item.resourceTitle ?? ''}》` },
  forum_reply: { icon: Reply, text: (item) => `回复了你的帖子「${item.resourceTitle ?? ''}」` },
  forum_like: { icon: Heart, text: (item) => `赞了你的帖子或回复「${item.resourceTitle ?? ''}」` },
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
  usePageMeta('通知')
  const navigate = useNavigate()
  const [page, setPage] = useState(1)
  const [filter, setFilter] = useState<NotificationFilter>('all')
  const pageSize = 20
  const queryClient = useQueryClient()
  const listQuery = useNotificationList(page, pageSize, filter)
  const items = groupNotificationItems(listQuery.data?.items ?? [])
  const unread = listQuery.data?.unread ?? 0
  const total = listQuery.data?.total ?? 0
  const loading = listQuery.isLoading
  const error = listQuery.error?.message ?? ''

  // 已读状态变化后同步刷新列表与全局未读角标缓存（Shell 的铃铛读这份缓存）
  const syncReadState = (nextUnread?: number) => {
    if (nextUnread !== undefined) queryClient.setQueryData(UNREAD_QUERY_KEY, nextUnread)
    void queryClient.invalidateQueries({ queryKey: ['notifications', 'list'] })
  }

  const markAllRead = async () => {
    try {
      await markAllNotificationsRead()
      syncReadState(0)
    } catch {
      // 失败时重新拉取，保证展示与后端一致
      syncReadState()
    }
  }

  const changeFilter = (next: typeof filter) => {
    setFilter(next)
    setPage(1)
  }

  // 点击通知：标记已读并跳转到对应内容
  const open = async (item: NotificationItem) => {
    if (!item.read) {
      await Promise.all((item.groupedIds ?? [item.id]).map((id) => markNotificationRead(id).catch(() => undefined)))
      // 角标即时更新：减去刚读掉的条数，不必等 SSE 或轮询
      syncReadState(Math.max(0, unread - (item.groupedIds?.length ?? 1)))
    }
    if (item.type === 'follow') navigate(`/users/${item.actor.username}`)
    else if (item.resourceSlug && (item.type === 'forum_reply' || item.type === 'forum_like')) navigate(`/forum/${item.resourceSlug}`)
    else if (item.resourceSlug) navigate(`/posts/${item.resourceSlug}${item.type === 'post' ? '' : '#comments'}`)
    else syncReadState()
  }

  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  return <section className="container my-posts-page">
    <div className="write-topbar"><Link className="back-link" to="/"><ArrowLeft size={15} /> 返回社区</Link><span className="eyebrow">Notifications</span></div>
    <div className="notifications-heading"><h1>通知中心 {unread > 0 && <small className="unread-badge">{unread} 条未读</small>}</h1>{unread > 0 && <button className="text-button" onClick={() => void markAllRead()}><CheckCheck size={15} /> 全部已读</button>}</div>
    {error && <div className="form-error">{error}</div>}
    <div className="notification-filters">{[['all', '全部'], ['unread', '未读'], ['comment', '文章评论'], ['reply', '评论回复'], ['like', '文章点赞'], ['forum_reply', '帖子回复'], ['forum_like', '帖子点赞'], ['follow', '关注'], ['post', '作者更新']].map(([value, label]) => <button key={value} className={filter === value ? 'is-active' : ''} onClick={() => changeFilter(value as typeof filter)}>{label}</button>)}</div>
    {loading ? <SkeletonList count={6} /> : items.length ? <div className="notifications-list">
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
    </div> : <div className="empty-state comments-empty"><h2>{filter === 'unread' ? '没有未读通知' : '还没有通知'}</h2><p>关注、互动和作者更新会出现在这里。</p></div>}
    {totalPages > 1 && <div className="admin-pagination"><button className="icon-button" disabled={page === 1} onClick={() => setPage((current) => current - 1)}>上一页</button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page >= totalPages} onClick={() => setPage((current) => current + 1)}>下一页</button></div>}
  </section>
}
