import { QueryClient, keepPreviousData, useQuery } from '@tanstack/react-query'
import { getCategories, getFeed, getForumTopics, getMyFavorites, getMyPosts, getMyStats, getNotifications, getTag, getTags, getTrendingTags, searchPosts } from './api'
import type { ForumKind, NotificationItem } from '../types'

// 全局查询客户端：列表/详情查询 30s 内视为新鲜，翻页保留上一页数据避免闪烁；
// 窗口隐藏时自动暂停轮询类查询（react-query 默认行为）
export const queryClient = new QueryClient({
  defaultOptions: {
    queries: { staleTime: 30_000, retry: 1, refetchOnWindowFocus: false, placeholderData: keepPreviousData },
  },
})

export const UNREAD_QUERY_KEY = ['notifications', 'unread'] as const

// 标签/分类变化极少，缓存 10 分钟——此前每次进入文章页都重新拉取
export function useTags() {
  return useQuery({ queryKey: ['tags'], queryFn: getTags, staleTime: 10 * 60_000 })
}

export function useCategories() {
  return useQuery({ queryKey: ['categories'], queryFn: getCategories, staleTime: 10 * 60_000 })
}

export function useTrendingTags(limit: number) {
  return useQuery({ queryKey: ['trending-tags', limit], queryFn: () => getTrendingTags({ limit }), staleTime: 5 * 60_000 })
}

export type FeedParams = { mode?: string; q?: string; tag?: string; category?: string; page?: number; pageSize?: number }

export function useFeed(params: FeedParams, staleTime = 0) {
  return useQuery({ queryKey: ['feed', params], queryFn: () => getFeed(params), staleTime })
}

export type ForumTopicsParams = { mode?: 'latest' | 'hot'; kind?: ForumKind | 'all'; q?: string; page?: number; pageSize?: number }

export function useForumTopics(params: ForumTopicsParams) {
  return useQuery({ queryKey: ['forum-topics', params], queryFn: () => getForumTopics(params) })
}

export function useMyPosts(status: string, page: number, pageSize: number) {
  return useQuery({ queryKey: ['my-posts', { status, page, pageSize }], queryFn: () => getMyPosts({ status, page, pageSize }) })
}

export function useMyFavorites(page: number, pageSize: number) {
  return useQuery({ queryKey: ['my-favorites', { page, pageSize }], queryFn: () => getMyFavorites({ page, pageSize }) })
}

export function useMyStats(enabled: boolean) {
  return useQuery({ queryKey: ['my-stats'], queryFn: getMyStats, enabled })
}

export function useTag(slug: string) {
  return useQuery({ queryKey: ['tag', slug], queryFn: () => getTag(slug) })
}

export function useSearch(q: string, page: number, pageSize = 12) {
  return useQuery({ queryKey: ['search', { q, page, pageSize }], queryFn: () => searchPosts(q, { page, pageSize }), enabled: q.trim().length > 0 })
}

export type NotificationFilter = 'all' | 'unread' | NotificationItem['type']

export function useNotificationList(page: number, pageSize: number, filter: NotificationFilter) {
  return useQuery({
    queryKey: ['notifications', 'list', { page, pageSize, filter }],
    queryFn: () => getNotifications({ page, pageSize, type: filter === 'all' || filter === 'unread' ? undefined : filter, unread: filter === 'unread' }),
  })
}

// 未读角标：SSE 推送直接写入该缓存（见 Shell），30s 轮询作为兜底；
// 未登录不查询
export function useUnreadCount(enabled: boolean) {
  return useQuery({
    queryKey: UNREAD_QUERY_KEY,
    queryFn: async () => (await getNotifications({ page: 1, pageSize: 1 })).unread,
    enabled,
    refetchInterval: 30_000,
  })
}
