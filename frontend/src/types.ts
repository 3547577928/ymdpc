export type PostStatus = 'draft' | 'published'

export type Post = {
  id: number
  title: string
  slug: string
  summary: string
  content: string
  coverImage: string
  tags: string[]
  status: PostStatus
  featured: boolean
  views: number
  publishedAt: string
  readingTime: number
  createdAt?: string
  updatedAt?: string
}
