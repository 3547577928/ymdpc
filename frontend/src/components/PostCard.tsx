import { ArrowUpRight, Clock3 } from 'lucide-react'
import { Link } from 'react-router-dom'
import type { Post } from '../types'
import { formatDate } from '../utils'

export function PostCard({ post, featured = false }: { post: Post; featured?: boolean }) {
  return (
    <article className={`post-card ${featured ? 'post-card-featured' : ''}`}>
      <Link to={`/posts/${post.slug}`} className="post-cover-link" aria-label={`阅读 ${post.title}`}>
        <img className="post-cover" src={post.coverImage} alt="" loading="lazy" />
        <span className="cover-arrow"><ArrowUpRight size={18} /></span>
      </Link>
      <div className="post-card-body">
        <div className="post-meta">
          <span>{formatDate(post.publishedAt)}</span>
          <span className="meta-dot" />
          <span className="inline-meta"><Clock3 size={14} /> {post.readingTime} min read</span>
        </div>
        <h3><Link to={`/posts/${post.slug}`}>{post.title}</Link></h3>
        <p>{post.summary}</p>
        <div className="tag-row">
          {post.tags.map((tag) => <span className="tag" key={tag}>{tag}</span>)}
        </div>
      </div>
    </article>
  )
}
