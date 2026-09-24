import { ArrowUpRight, Clock3, Heart, MessageCircle } from 'lucide-react'
import { memo } from 'react'
import { Link } from 'react-router-dom'
import type { PostSummary } from '../types'
import { formatDate } from '../utils'
import { CoverImage } from './CoverImage'

// memo：列表页搜索框每次按键都会重渲染父组件，post 引用不变时卡片无需跟着重渲染
export const PostCard = memo(function PostCard({ post, featured = false, showAuthor = true }: { post: PostSummary; featured?: boolean; showAuthor?: boolean }) {
  return (
    <article className={`post-card ${featured ? 'post-card-featured' : ''}`}>
      <Link to={`/posts/${post.slug}`} className="post-cover-link" aria-label={`阅读 ${post.title}`}>
        <CoverImage className="post-cover" src={post.coverImage} title={post.title} loading="lazy" />
        <span className="cover-arrow"><ArrowUpRight size={18} /></span>
      </Link>
      <div className="post-card-body">
        <div className="post-meta">
          <span>{formatDate(post.publishedAt)}</span>
          <span className="meta-dot" />
          <span className="inline-meta"><Clock3 size={14} /> {post.readingTime} min read</span>
          <span className="inline-meta"><Heart size={13} /> {post.likesCount}</span>
          <span className="inline-meta"><MessageCircle size={13} /> {post.commentsCount}</span>
        </div>
        <h3><Link to={`/posts/${post.slug}`}>{post.title}</Link></h3>
        <p>{post.summary}</p>
        {showAuthor && <Link className="post-author" to={`/users/${post.author.username}`}>@{post.author.nickname}</Link>}
        <div className="tag-row">
          {post.tags.map((tag) => <span className="tag" key={tag}>{tag}</span>)}
        </div>
      </div>
    </article>
  )
})
