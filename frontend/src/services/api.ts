import type { Post } from '../types'

const API_BASE = import.meta.env.VITE_API_BASE ?? '/api'

type ApiPayload<T> = { code: number; message: string; data: T }

export type AuthUser = { id: number; username: string; nickname: string }
export type Tag = { id: number; name: string; slug: string }
export type PostPage = { items: Post[]; total: number; page: number; pageSize: number }
export type PostInput = Pick<Post, 'title' | 'summary' | 'content' | 'coverImage' | 'tags' | 'status' | 'featured'> & { slug?: string }

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...(options?.headers ?? {}) },
    ...options,
  })
  const payload = await response.json().catch(() => ({ message: '服务器响应格式错误' })) as ApiPayload<T>
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
  return request<Post>(`/posts/${encodeURIComponent(slug)}`)
}

export function getTags() {
  return request<Tag[]>('/tags')
}

export function login(username: string, password: string) {
  return request<AuthUser>('/auth/login', { method: 'POST', body: JSON.stringify({ username, password }) })
}

export function logout() {
  return request<void>('/auth/logout', { method: 'POST' })
}

export function getCurrentUser() {
  return request<AuthUser>('/auth/me')
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

export function updatePostStatus(id: number, status: Post['status']) {
  return request<void>(`/admin/posts/${id}/status`, { method: 'PATCH', body: JSON.stringify({ status }) })
}
