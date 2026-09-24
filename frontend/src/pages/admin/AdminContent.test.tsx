import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import type { AdminSettings, AdminStats, AdminUser } from '../../services/api'
import type { PostSummary, UserSummary } from '../../types'
import { AdminContent } from './AdminContent'

const user: UserSummary = { id: 1, username: 'alice', nickname: 'Alice', avatar: '', bio: '' }
const stats: AdminStats = {
  total: 1,
  users: 3,
  posts: 1,
  published: 1,
  views: 128,
  likes: 12,
  comments: 4,
  follows: 2,
  todayUsers: 1,
  todayPosts: 1,
  todayComments: 1,
  pendingReports: 0,
  activeUsers: [],
  dailyMetrics: [],
  topPosts: [],
}
const post: PostSummary = {
  id: 9,
  author: user,
  title: '测试文章',
  slug: 'test-post',
  summary: '摘要',
  coverImage: '',
  tags: [],
  status: 'published',
  moderationStatus: 'normal',
  category: null,
  featured: false,
  views: 12,
  likesCount: 2,
  favoriteCount: 0,
  commentsCount: 1,
  liked: false,
  publishedAt: '2026-09-23T00:00:00Z',
  readingTime: 1,
}
const settings: AdminSettings = { openRegistration: true, commentsEnabled: true }

function renderContent(overrides: Partial<React.ComponentProps<typeof AdminContent>> = {}) {
  const base: React.ComponentProps<typeof AdminContent> = {
    active: '文章',
    stats,
    posts: [post],
    listLoading: false,
    page: 1,
    totalPages: 1,
    onPageChange: vi.fn(),
    onEditPost: vi.fn(),
    onPublishPost: vi.fn(),
    onModeratePost: vi.fn(),
    onDeletePost: vi.fn(),
    users: [],
    userTotal: 0,
    userPage: 1,
    userLoading: false,
    onUserPageChange: vi.fn(),
    onToggleUser: vi.fn(),
    comments: [],
    commentTotal: 0,
    commentPage: 1,
    commentLoading: false,
    onCommentPageChange: vi.fn(),
    commentPostFilter: undefined,
    setCommentPostFilter: vi.fn(),
    onCommentStatus: vi.fn(),
    onDeleteComment: vi.fn(),
    tagData: { tags: [], categories: [] },
    newCategory: '',
    setNewCategory: vi.fn(),
    onCreateCategory: vi.fn(),
    onDeleteCategory: vi.fn(),
    onDeleteTag: vi.fn(),
    reports: [],
    reportStatus: 'pending',
    reportTotal: 0,
    reportPage: 1,
    reportLoading: false,
    onReportStatusChange: vi.fn(),
    onReportPageChange: vi.fn(),
    onReport: vi.fn(),
    onExport: vi.fn(),
    logs: [],
    logTotal: 0,
    logPage: 1,
    logLoading: false,
    onLogPageChange: vi.fn(),
    settings,
    setSettings: vi.fn(),
    onSaveSettings: vi.fn(),
  }
  return render(<AdminContent {...base} {...overrides} />)
}

describe('AdminContent', () => {
  it('renders article actions and forwards selected post', () => {
    const onEditPost = vi.fn()
    const onPublishPost = vi.fn()
    renderContent({ onEditPost, onPublishPost })

    fireEvent.click(screen.getByRole('button', { name: '测试文章' }))
    fireEvent.click(screen.getByRole('button', { name: '已发布' }))

    expect(onEditPost).toHaveBeenCalledWith(post)
    expect(onPublishPost).toHaveBeenCalledWith(post)
  })

  it('renders users and forwards status changes', () => {
    const onToggleUser = vi.fn()
    const member: AdminUser = { ...user, status: 'active', postCount: 2, likeCount: 5, followers: 3, following: 1 }
    renderContent({ active: '用户', users: [member], userTotal: 1, onToggleUser })

    expect(screen.getByText('@alice')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: '正常' }))

    expect(onToggleUser).toHaveBeenCalledWith(member)
  })
})
