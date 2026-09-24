export type PostStatus = 'draft' | 'scheduled' | 'published' | 'archived'

export type Category = { id: number; name: string; slug: string }

export type PostSummary = {
  id: number
  author: UserSummary
  title: string
  slug: string
  summary: string
  coverImage: string
  tags: string[]
  status: PostStatus
  moderationStatus?: string
  category?: Category | null
  seriesId?: number | null
  featured: boolean
  views: number
  likesCount: number
  favoriteCount: number
  commentsCount: number
  liked: boolean
  favorited?: boolean
  publishedAt: string
  scheduledAt?: string | null
  readingTime: number
  createdAt?: string
  updatedAt?: string
}

export type Post = PostSummary & {
  content: string
}

export type PostRevision = {
  id: number
  title: string
  slug: string
  summary: string
  content: string
  coverImage: string
  categoryId: number | null
  featured: boolean
  tags: string[]
  createdAt: string
}

export type UserSummary = {
  id: number
  username: string
  email?: string
  nickname: string
  avatar: string
  bio: string
  role?: string
  createdAt?: string
}

export type ForumKind = 'discuss' | 'share' | 'help' | 'rant'

export type ForumTopic = {
  id: number
  content: string
  kind: ForumKind
  images: string[]
  likesCount: number
  repliesCount: number
  liked: boolean
  createdAt: string
  updatedAt: string
  editedAt?: string | null
  author: UserSummary
}

export type ForumReply = {
  id: number
  topicId: number
  parentId: number | null
  replyToUserId: number | null
  content: string
  likesCount: number
  liked: boolean
  createdAt: string
  editedAt?: string | null
  author: UserSummary
}

export type Comment = {
  id: number
  postId: number
  parentId: number | null
  replyToUserId: number | null
  content: string
  pinned: boolean
  likesCount: number
  liked: boolean
  createdAt: string
  editedAt?: string | null
  author: UserSummary
}

export type AdminComment = {
  id: number
  postId: number
  postTitle: string
  postSlug: string
  content: string
  status: string
  pinned: boolean
  createdAt: string
  author: UserSummary
}

export type NotificationItem = {
  id: number
  type: 'comment' | 'reply' | 'like' | 'follow' | 'post' | 'forum_reply' | 'forum_like'
  resourceId: number
  resourceSlug?: string
  resourceTitle?: string
  read: boolean
  createdAt: string
  actor: UserSummary
  groupCount?: number
  groupedIds?: number[]
}

export type AdminReport = {
  id: number
  targetType: 'post' | 'comment' | 'forum_topic' | 'forum_reply'
  targetId: number
  targetSummary: string
  targetTitle?: string
  targetSlug?: string
  targetStatus?: string
  reason: string
  status: 'pending' | 'handled' | 'dismissed'
  resolution?: string
  handledAt?: string
  createdAt: string
  reporter: UserSummary
  handler?: UserSummary
}

export type AdminLogEntry = {
  id: number
  action: string
  targetType: string
  targetId: number
  detail: string
  createdAt: string
  admin: UserSummary
}

export type TagUsage = { id: number; name: string; slug: string; usage: number }
