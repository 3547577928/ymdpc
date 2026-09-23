import type { AdminComment, AdminLogEntry, AdminReport, Category, Comment, NotificationItem, Post, PostStatus, PostSummary, TagUsage, UserSummary } from '../types'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api'

type ApiPayload<T> = { code: number; message: string; data: T }

export type AuthUser = UserSummary & { role: string; status: string }
export type Tag = { id: number; name: string; slug: string }
export type PostPage = { items: PostSummary[]; total: number; page: number; pageSize: number }
export type PostInput = Pick<Post, 'title' | 'summary' | 'content' | 'coverImage' | 'tags' | 'status' | 'featured'> & { slug?: string; categoryId?: number | null }
export type AdjacentPost = Pick<PostSummary, 'title' | 'slug'>
export type PostDetail = { post: Post; previous: AdjacentPost | null; next: AdjacentPost | null }
export type AdminStats = { total: number; users: number; posts: number; published: number; views: number; likes: number; comments: number; follows: number; todayUsers: number; todayPosts: number; todayComments: number; activeUsers: { user: UserSummary; postCount: number; commentCount: number }[] }
export type FeedPage = PostPage & { mode: string }
export type UserProfile = { user: UserSummary; posts: PostSummary[]; postCount: number; likeCount: number; followers: number; following: number; followingMe: boolean }
export type AdminUser = UserSummary & { status: string; postCount: number; likeCount: number; followers: number; following: number }

// 请求超时：后端无响应时及时失败，避免页面长期停留在加载态
const REQUEST_TIMEOUT_MS = 15_000

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const headers = new Headers(options?.headers)
  if (options?.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  let response: Response
  try {
    response = await fetch(`${API_BASE}${path}`, {
      ...options,
      credentials: 'include',
      signal: AbortSignal.timeout(REQUEST_TIMEOUT_MS),
      headers,
    })
  } catch (reason) {
    // 超时抛的是 DOMException，转成中文提示，其余错误原样抛出
    if (reason instanceof DOMException && reason.name === 'TimeoutError') {
      throw new Error('请求超时，请检查网络后重试')
    }
    throw reason
  }
  const payload = await response.json().catch(() => ({
    code: response.status,
    message: response.ok ? '服务器响应格式错误' : `请求失败 (${response.status})`,
  })) as ApiPayload<T>
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || `请求失败 (${response.status})`)
  }
  return payload.data
}

export function getPosts(params: { q?: string; tag?: string; page?: number; pageSize?: number } = {}, admin = false) {
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

export function login(username: string, password: string) {
  return request<AuthUser>('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) })
}

export function register(username: string, password: string, nickname: string) {
  return request<AuthUser>('/auth/register', { method: 'POST', body: JSON.stringify({ username, password, nickname }) })
}

export function logout() {
  return request<void>('/auth/logout', { method: 'POST' })
}

export function getCurrentUser() {
  return request<AuthUser>('/auth/me')
}

export function updateProfile(input: { nickname: string; avatar: string; bio: string }) {
  return request<AuthUser>('/me/profile', { method: 'PATCH', body: JSON.stringify(input) })
}

export function updatePassword(input: { currentPassword: string; newPassword: string }) {
  return request<void>('/me/password', { method: 'PATCH', body: JSON.stringify(input) })
}

export function getFeed(params: { mode?: string; q?: string; tag?: string; page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<FeedPage>(`/feed${query.size ? `?${query}` : ''}`)
}

export function getUserProfile(username: string) {
  return request<UserProfile>(`/users/${encodeURIComponent(username)}`)
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

export function getComments(slug: string) {
  return request<Comment[]>(`/posts/${encodeURIComponent(slug)}/comments`)
}

export function createComment(slug: string, input: { content: string; parentId?: number; replyToUserId?: number }) {
  return request<Comment>(`/posts/${encodeURIComponent(slug)}/comments`, { method: 'POST', body: JSON.stringify(input) })
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

export function createReport(input: { targetType: 'post' | 'comment'; targetId: number; reason: string }) {
  return request<void>('/reports', { method: 'POST', body: JSON.stringify(input) })
}

export function getNotifications(params: { page?: number; pageSize?: number } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<{ items: NotificationItem[]; total: number; unread: number; page: number; pageSize: number }>(`/notifications${query.size ? `?${query}` : ''}`)
}

export function markNotificationRead(id: number) {
  return request<void>(`/notifications/${id}/read`, { method: 'PATCH' })
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

export function getCategories() {
  return request<Category[]>('/categories')
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

export function updatePostStatus(id: number, status: PostStatus) {
  return request<void>(`/admin/posts/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) })
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

export function getAdminReports(params: { page?: number; pageSize?: number; status?: string } = {}) {
  const query = new URLSearchParams()
  Object.entries(params).forEach(([key, value]) => { if (value) query.set(key, String(value)) })
  return request<{ items: AdminReport[]; total: number; page: number; pageSize: number }>(`/admin/reports${query.size ? `?${query}` : ''}`)
}

export function handleAdminReport(id: number, status: 'handled' | 'dismissed') {
  return request<void>(`/admin/reports/${id}`, { method: 'PATCH', body: JSON.stringify({ status }) })
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
  return request<Category>('/admin/categories', { method: 'POST', body: JSON.stringify({ name }) })
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
