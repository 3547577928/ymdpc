import type { AdminComment, AdminLogEntry, AdminReport, Category, Comment, ForumKind, ForumReply, ForumTopic, NotificationItem, Post, PostRevision, PostStatus, PostSummary, TagUsage, UserSummary } from '../types'
import { prepareImageForUpload } from '../utils/imageUpload'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api'

type ApiPayload<T> = { code: number; message: string; data: T }

export type AuthUser = UserSummary & { role: string; status: string }
export type Tag = { id: number; name: string; slug: string }
export type PostPage = { items: PostSummary[]; total: number; page: number; pageSize: number }
export type PostInput = Pick<Post, 'title' | 'summary' | 'content' | 'coverImage' | 'tags' | 'status' | 'featured'> & { slug?: string; categoryId?: number | null; seriesId?: number | null; scheduledAt?: string | null }
export type AdjacentPost = Pick<PostSummary, 'title' | 'slug'>
export type SeriesBrief = { id: number; title: string; slug: string }
export type SeriesListItem = SeriesBrief & { description: string; postCount: number; createdAt: string }
export type SeriesDetail = SeriesBrief & { description: string; author: UserSummary; posts: AdjacentPost[] }
export type PostDetail = { post: Post; previous: AdjacentPost | null; next: AdjacentPost | null; series?: SeriesDetail | null }
export type AdminDailyMetric = { date: string; users: number; posts: number; comments: number }
export type AdminTopPost = { id: number; title: string; slug: string; views: number; likesCount: number; commentsCount: number; author: UserSummary }
export type AdminStats = { total: number; users: number; posts: number; published: number; views: number; likes: number; comments: number; follows: number; todayUsers: number; todayPosts: number; todayComments: number; pendingReports: number; activeUsers: { user: UserSummary; postCount: number; commentCount: number }[]; dailyMetrics: AdminDailyMetric[]; topPosts: AdminTopPost[] }
export type TrendingTag = { id: number; name: string; slug: string; postCount: number; score: number }
export type FeedPage = PostPage & { mode: string }
export type UserProfile = { user: UserSummary; posts: PostSummary[]; postCount: number; likeCount: number; followers: number; following: number; followingMe: boolean }
export type AdminUser = UserSummary & { status: string; postCount: number; likeCount: number; followers: number; following: number }
export type ForumTopicPage = { items: ForumTopic[]; total: number; page: number; pageSize: number }
export type ForumTopicDetail = { topic: ForumTopic; replies: ForumReply[]; repliesTotal: number }
export type CommentPage = { items: Comment[]; total: number; page: number; pageSize: number }

// 请求超时：后端无响应时及时失败，避免页面长期停留在加载态
const REQUEST_TIMEOUT_MS = 15_000
export const AUTH_EXPIRED_EVENT = 'quietsig:auth-expired'

type RequestOptions = RequestInit & { suppressAuthExpired?: boolean }

function notifyAuthExpired() {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event(AUTH_EXPIRED_EVENT))
  }
}

async function request<T>(path: string, options?: RequestOptions): Promise<T> {
  const { suppressAuthExpired = false, ...fetchOptions } = options ?? {}
  const headers = new Headers(fetchOptions.headers)
  // FormData needs the browser-generated multipart boundary; setting JSON here
  // makes Gin unable to parse the uploaded file from the request body.
  if (fetchOptions.body && !(fetchOptions.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  let response: Response
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...fetchOptions,
      credentials: 'include',
      signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
      headers,
    })
  } catch (reason) {
    // 超时抛的是 DOMException，转成中文提示，其余错误原样抛出
    if (reason instanceof DOMException && reason.name === 'TimeoutError') {
      throw new Error('请求超时，请检查网络后重试', { cause: reason })
    }
    throw reason
  }
  const payload = await response.json().catch(() => ({
    code: response.status,
    message: response.ok ? '服务器响应格式错误' : `请求失败 (${response.status})`,
  })) as ApiPayload<T>
  if (!response.ok || payload.code !== 0) {
    if (response.status === 401 && !suppressAuthExpired) notifyAuthExpired()
    throw new Error(payload.message || `请求失败 (${response.status})`)
  }
  return payload.data
}

export function getPosts(params: { q?: string; tag?: string; category?: string; page?: number; pageSize?: number } = {}, admin = false) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<PostPage>(`${admin ? '/admin' : ''}/posts${query.size ? `?${query}` : ''}`)
}

export function getPostBySlug(slug: string) {
  return request<PostDetail>(`/posts/${encodeURIComponent(slug)}`)
}

export function recordPostView(slug: string) {
  return request<{ views: number }>(`/posts/${encodeURIComponent(slug)}/views`, { method: 'POST' })
}

export function getTags() {
  return request<Tag[]>('/tags')
}

export function getSeries(slug: string) {
  return request<SeriesDetail>(`/series/${encodeURIComponent(slug)}`)
}

export function getAuthorSeries(username: string) {
  return request<SeriesListItem[]>(`/users/${encodeURIComponent(username)}/series`)
}

export function createSeries(input: { title: string; description: string }) {
  return request<SeriesListItem>('/series', { method: 'POST', body: JSON.stringify(input) })
}

export type TagInfo = { id: number; name: string; slug: string; postCount: number; subscribed: boolean }

export function getTag(slug: string) {
  return request<TagInfo>(`/tags/${encodeURIComponent(slug)}`)
}

export function subscribeTag(id: number) {
  return request<{ subscribed: boolean }>(`/tags/${id}/subscribe`, { method: 'POST' })
}

export function unsubscribeTag(id: number) {
  return request<{ subscribed: boolean }>(`/tags/${id}/subscribe`, { method: 'DELETE' })
}

export type SearchResult = PostSummary & { titleMarked: string; snippet: string }
export type SearchPage = { items: SearchResult[]; total: number; page: number; pageSize: number; fts: boolean; topics: ForumTopic[]; users: UserSummary[] }

export function searchPosts(q: string, params: { page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  query.set('q', q)
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('pageSize', String(params.pageSize))
  return request<SearchPage>(`/search?${query}`)
}

export function login(username: string, password: string) {
  return request<AuthUser>('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }), suppressAuthExpired: true })
}

// 忘记密码：验证码校验通过后重置密码并直接签发会话
export function resetPassword(email: string, code: string, newPassword: string) {
  return request<AuthUser>('/auth/password/reset', { method: 'POST', body: JSON.stringify({ email, code, newPassword }), suppressAuthExpired: true })
}

export function requestEmailCode(email: string) {
  return request<void>('/auth/email-code/request', { method: 'POST', body: JSON.stringify({ email }), suppressAuthExpired: true })
}

export function verifyEmailCode(email: string, code: string) {
  return request<AuthUser>('/auth/email-code/verify', { method: 'POST', body: JSON.stringify({ email, code }), suppressAuthExpired: true })
}

export function register(username: string, password: string, nickname: string) {
  return request<AuthUser>('/auth/register', { method: 'POST', body: JSON.stringify({ username, password, nickname }), suppressAuthExpired: true })
}

export function logout() {
  return request<void>('/auth/logout', { method: 'POST', suppressAuthExpired: true })
}

export function getCurrentUser() {
  return request<AuthUser>('/auth/me', { suppressAuthExpired: true })
}

export function updateProfile(input: { nickname: string; avatar: string; bio: string; email: string }) {
  return request<AuthUser>('/me/profile', { method: 'PATCH', body: JSON.stringify(input) })
}

export function updatePassword(input: { currentPassword: string; newPassword: string }) {
  return request<void>('/me/password', { method: 'PATCH', body: JSON.stringify(input) })
}

export function getFeed(params: { mode?: string; q?: string; tag?: string; category?: string; page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<FeedPage>(`/feed${query.size ? `?${query}` : ''}`)
}

export function getForumTopics(params: { mode?: 'latest' | 'hot'; kind?: ForumKind | 'all'; q?: string; page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value && value !== 'all') query.set(key, String(value)) })
  return request<ForumTopicPage>(`/forum${query.size ? `?${query}` : ''}`)
}

export function createForumTopic(input: { content: string; kind: ForumKind; images: string[] }) {
  return request<ForumTopic>('/forum', { method: 'POST', body: JSON.stringify(input) })
}

export function getForumTopic(id: number, params: { replyPage?: number; replyPageSize?: number } = {}) {
  const query = new URLSearchParams()
  if (params.replyPage) query.set('replyPage', String(params.replyPage))
  if (params.replyPageSize) query.set('replyPageSize', String(params.replyPageSize))
  return request<ForumTopicDetail>(`/forum/${id}${query.size ? `?${query}` : ''}`)
}

// 编辑话题：images 不传表示保留原图
export function updateForumTopic(id: number, input: { content: string; kind: ForumKind; images?: string[] }) {
  return request<ForumTopic>(`/forum/${id}`, { method: 'PATCH', body: JSON.stringify(input) })
}

export function updateForumReply(id: number, content: string) {
  return request<ForumReply>(`/forum/replies/${id}`, { method: 'PATCH', body: JSON.stringify({ content }) })
}

export function deleteForumTopic(id: number) {
  return request<void>(`/forum/${id}`, { method: 'DELETE' })
}

export function likeForumTopic(id: number) {
  return request<{ liked: boolean; likesCount: number }>(`/forum/${id}/like`, { method: 'POST' })
}

export function unlikeForumTopic(id: number) {
  return request<{ liked: boolean; likesCount: number }>(`/forum/${id}/like`, { method: 'DELETE' })
}

export function createForumReply(id: number, input: { content: string; parentId?: number; replyToUserId?: number }) {
  return request<ForumReply>(`/forum/${id}/replies`, { method: 'POST', body: JSON.stringify(input) })
}

export function deleteForumReply(id: number) {
  return request<void>(`/forum/replies/${id}`, { method: 'DELETE' })
}

export function likeForumReply(id: number) {
  return request<{ liked: boolean; likesCount: number }>(`/forum/replies/${id}/like`, { method: 'POST' })
}

export function unlikeForumReply(id: number) {
  return request<{ liked: boolean; likesCount: number }>(`/forum/replies/${id}/like`, { method: 'DELETE' })
}

export type AuthorDailyMetric = { date: string; posts: number; likes: number; comments: number }
export type MyStats = { posts: number; published: number; drafts: number; views: number; likes: number; comments: number; followers: number; dailyMetrics: AuthorDailyMetric[]; topPosts: { id: number; title: string; slug: string; views: number; likesCount: number; commentsCount: number }[] }

export function getMyStats() {
  return request<MyStats>('/me/stats')
}

export function getUserProfile(username: string, params: { page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  if (params.page) query.set('page', String(params.page))
  if (params.pageSize) query.set('pageSize', String(params.pageSize))
  return request<UserProfile>(`/users/${encodeURIComponent(username)}${query.size ? `?${query}` : ''}`)
}

export function followUser(id: number) {
  return request<{ following: boolean; followers: number }>(`/users/${id}/follow`, { method: 'POST' })
}

export function unfollowUser(id: number) {
  return request<{ following: boolean; followers: number }>(`/users/${id}/follow`, { method: 'DELETE' })
}

export function createUserPost(input: PostInput) {
  return request<Post>('/posts', { method: 'POST', body: JSON.stringify(input) })
}

export function updateUserPost(id: number, input: PostInput) {
  return request<Post>(`/posts/id/${id}`, { method: 'PUT', body: JSON.stringify(input) })
}

export function deleteUserPost(id: number) {
  return request<void>(`/posts/id/${id}`, { method: 'DELETE' })
}

// 评论按顶层分页返回（回复随父评论带出），total 为顶层评论数
export function getComments(slug: string, params: { page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  query.set('page', String(params.page ?? 1))
  query.set('pageSize', String(params.pageSize ?? 20))
  return request<CommentPage>(`/posts/${encodeURIComponent(slug)}/comments?${query}`)
}

// 「正在输入」提醒：客户端节流到每 3 秒最多一次
export function sendCommentTyping(slug: string) {
  return request<void>(`/posts/${encodeURIComponent(slug)}/typing`, { method: 'POST' })
}

export function createComment(slug: string, input: { content: string; parentId?: number; replyToUserId?: number }) {
  return request<Comment>(`/posts/${encodeURIComponent(slug)}/comments`, { method: 'POST', body: JSON.stringify(input) })
}

// 编辑评论：postSlug 用于服务端向该文章的读者广播编辑事件
export function updateComment(id: number, content: string, postSlug?: string) {
  return request<Comment>(`/comments/${id}${postSlug ? `?postSlug=${encodeURIComponent(postSlug)}` : ''}`, { method: 'PATCH', body: JSON.stringify({ content }) })
}

export function deleteComment(id: number) {
  return request<void>(`/comments/${id}`, { method: 'DELETE' })
}

export function likeComment(id: number) {
  return request<{ liked: boolean; likesCount: number }>(`/comments/${id}/like`, { method: 'POST' })
}

export function unlikeComment(id: number) {
  return request<{ liked: boolean; likesCount: number }>(`/comments/${id}/like`, { method: 'DELETE' })
}

export function favoritePost(slug: string) {
  return request<{ favorited: boolean; favoriteCount: number }>(`/posts/${encodeURIComponent(slug)}/favorite`, { method: 'POST' })
}

export function unfavoritePost(slug: string) {
  return request<{ favorited: boolean; favoriteCount: number }>(`/posts/${encodeURIComponent(slug)}/favorite`, { method: 'DELETE' })
}

export function createReport(input: { targetType: 'post' | 'comment' | 'forum_topic' | 'forum_reply'; targetId: number; reason: string }) {
  return request<void>('/reports', { method: 'POST', body: JSON.stringify(input) })
}

export function getNotifications(params: { page?: number; pageSize?: number; type?: NotificationItem['type']; unread?: boolean } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<{ items: NotificationItem[]; total: number; unread: number; page: number; pageSize: number }>(`/notifications${query.size ? `?${query}` : ''}`)
}

export function markNotificationRead(id: number) {
  return request<void>(`/notifications/${id}/read`, { method: 'PATCH' })
}

export function markAllNotificationsRead() {
  return request<void>('/notifications/read-all', { method: 'POST' })
}

export function pinComment(id: number, pinned: boolean) {
  return request<{ pinned: boolean }>(`/comments/${id}/pin`, { method: 'PATCH', body: JSON.stringify({ pinned }) })
}

export function getMyPosts(params: { status?: string; q?: string; page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<PostPage>(`/me/posts${query.size ? `?${query}` : ''}`)
}

export function getMyFavorites(params: { page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<PostPage>(`/me/favorites${query.size ? `?${query}` : ''}`)
}

export function getPostById(id: number) {
  return request<Post>(`/posts/id/${id}`)
}

export function getPostRevisions(id: number) {
  return request<PostRevision[]>(`/posts/id/${id}/revisions`)
}

export function restorePostRevision(id: number, revisionId: number) {
  return request<Post>(`/posts/id/${id}/revisions/${revisionId}/restore`, { method: 'POST' })
}

export function getCategories() {
  return request<Category[]>('/categories')
}

export function getTrendingTags(params: { days?: number; limit?: number } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<TrendingTag[]>(`/trending/tags${query.size ? `?${query}` : ''}`)
}

export async function uploadImage(file: File, options: { avatar?: boolean } = {}) {
  const body = new FormData()
  // 头像最大展示位 96px，256px 覆盖 2x 高分屏；原来统一压到 800px 纯属浪费带宽
  body.append('file', await prepareImageForUpload(file, options.avatar ? 256 : 2400))
  return request<{ url: string }>('/uploads/images', { method: 'POST', body })
}

export function likePost(slug: string) {
  return request<{ liked: boolean; likesCount: number }>(`/posts/${encodeURIComponent(slug)}/like`, { method: 'POST' })
}

export function unlikePost(slug: string) {
  return request<{ liked: boolean; likesCount: number }>(`/posts/${encodeURIComponent(slug)}/like`, { method: 'DELETE' })
}

export function getAdminPost(id: number) {
  return request<Post>(`/admin/posts/${id}`)
}

export function getAdminStats() {
  return request<AdminStats>('/admin/stats')
}

export function getAdminUsers(params: { page?: number; pageSize?: number; q?: string } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<{ items: AdminUser[]; total: number; page: number; pageSize: number }>(`/admin/users${query.size ? `?${query}` : ''}`)
}

export function updateAdminUser(id: number, input: { status?: string; role?: string }) {
  return request<AdminUser>(`/admin/users/${id}`, { method: 'PATCH', body: JSON.stringify(input) })
}

export function createPost(input: PostInput) {
  return request<Post>('/admin/posts', { method: 'POST', body: JSON.stringify(input) })
}

export function updatePost(id: number, input: PostInput) {
  return request<Post>(`/admin/posts/${id}`, { method: 'PUT', body: JSON.stringify(input) })
}

export function deletePost(id: number) {
  return request<void>(`/admin/posts/${id}`, { method: 'DELETE' })
}

export function updatePostStatus(id: number, status: PostStatus, scheduledAt?: string | null) {
  return request<void>(`/admin/posts/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status, scheduledAt }) })
}

export function updatePostModeration(id: number, status: 'normal' | 'hidden') {
  return request<void>(`/admin/posts/${id}/moderation`, { method: 'PATCH', body: JSON.stringify({ status }) })
}

export function getAdminComments(params: { page?: number; pageSize?: number; status?: string; postId?: number } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<{ items: AdminComment[]; total: number; page: number; pageSize: number }>(`/admin/comments${query.size ? `?${query}` : ''}`)
}

export function updateAdminCommentStatus(id: number, status: 'published' | 'hidden') {
  return request<void>(`/admin/comments/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) })
}

export function updateAdminCommentsStatus(ids: number[], status: 'published' | 'hidden') {
  return request<void>('/admin/comments/batch', { method: 'POST', body: JSON.stringify({ ids, status }) })
}

export function getAdminReports(params: { page?: number; pageSize?: number; status?: string } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<{ items: AdminReport[]; total: number; page: number; pageSize: number }>(`/admin/reports${query.size ? `?${query}` : ''}`)
}

export function getAdminReport(id: number) {
  return request<AdminReport>(`/admin/reports/${id}`)
}

export function handleAdminReport(id: number, status: 'handled' | 'dismissed', resolution = '') {
  return request<void>(`/admin/reports/${id}`, { method: 'PATCH', body: JSON.stringify({ status, resolution }) })
}

export function handleAdminReports(ids: number[], status: 'handled' | 'dismissed', resolution = '') {
  return request<void>('/admin/reports/batch', { method: 'POST', body: JSON.stringify({ ids, status, resolution }) })
}

export async function downloadAdminExport(type: 'posts' | 'users' | 'reports') {
  const response = await fetch(`${API_BASE}/admin/export?type=${type}`, { credentials: 'include', signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS) })
  if (!response.ok) throw new Error(`导出失败 (${response.status})`)
  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `quietsig-${type}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

export function getAdminLogs(params: { page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<{ items: AdminLogEntry[]; total: number; page: number; pageSize: number }>(`/admin/logs${query.size ? `?${query}` : ''}`)
}

export function getAdminTags() {
  return request<{ tags: TagUsage[]; categories: TagUsage[] }>('/admin/tags')
}

export function createCategory(name: string) {
  return request<Category>('/categories', { method: 'POST', body: JSON.stringify({ name }) })
}

export function deleteCategory(id: number) {
  return request<void>(`/admin/categories/${id}`, { method: 'DELETE' })
}

export function deleteTag(id: number) {
  return request<void>(`/admin/tags/${id}`, { method: 'DELETE' })
}

export type AdminSettings = { openRegistration: boolean; commentsEnabled: boolean }

export function getAdminSettings() {
  return request<AdminSettings>('/admin/settings')
}

export function updateAdminSettings(input: Partial<AdminSettings>) {
  return request<AdminSettings>('/admin/settings', { method: 'PATCH', body: JSON.stringify(input) })
}
