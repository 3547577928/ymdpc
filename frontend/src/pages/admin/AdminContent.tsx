import { ChevronLeft, ChevronRight, Eye, EyeOff, Flag, History, MessageSquare, Pencil, Plus, Settings, Tag, Trash2, UserRound } from 'lucide-react'
import type { Dispatch, SetStateAction } from 'react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import type { AdminSettings, AdminStats, AdminUser } from '../../services/api'
import type { AdminComment, AdminLogEntry, AdminReport, PostStatus, PostSummary, TagUsage } from '../../types'
import { formatDate } from '../../utils'

const statusLabels: Record<PostStatus, string> = { draft: '草稿', scheduled: '待发布', published: '已发布', archived: '已归档' }
const userStatusLabels: Record<string, string> = { active: '正常', muted: '禁言', banned: '封禁' }
const commentStatusLabels: Record<string, string> = { published: '公开', hidden: '已隐藏' }
const reportStatusLabels: Record<string, string> = { pending: '待处理', handled: '已处理', dismissed: '已驳回' }

type AdminContentProps = {
  active: string
  stats: AdminStats
  posts: PostSummary[]
  listLoading: boolean
  page: number
  totalPages: number
  onPageChange: (page: number) => void
  onEditPost: (post: PostSummary) => void
  onPublishPost: (post: PostSummary) => void
  onModeratePost: (post: PostSummary) => void
  onDeletePost: (post: PostSummary) => void
  users: AdminUser[]
  userTotal: number
  userPage: number
  userLoading: boolean
  onUserPageChange: (page: number) => void
  onToggleUser: (member: AdminUser) => void
  comments: AdminComment[]
  commentTotal: number
  commentPage: number
  commentLoading: boolean
  commentPostFilter?: number
  onCommentPageChange: (page: number) => void
  setCommentPostFilter: Dispatch<SetStateAction<number | undefined>>
  onCommentStatus: (comment: AdminComment) => void
  onCommentsBatch?: (ids: number[], status: 'published' | 'hidden') => void
  onDeleteComment: (comment: AdminComment) => void
  tagData: { tags: TagUsage[]; categories: TagUsage[] }
  newCategory: string
  setNewCategory: Dispatch<SetStateAction<string>>
  onCreateCategory: () => void
  onDeleteCategory: (category: TagUsage) => void
  onDeleteTag: (tag: TagUsage) => void
  reports: AdminReport[]
  reportStatus: string
  reportTotal: number
  reportPage: number
  reportLoading: boolean
  onReportStatusChange: (status: string) => void
  onReportPageChange: (page: number) => void
  onReport: (report: AdminReport, status: 'handled' | 'dismissed') => void
  onReportsBatch?: (ids: number[], status: 'handled' | 'dismissed') => void
  logs: AdminLogEntry[]
  logTotal: number
  logPage: number
  logLoading: boolean
  onLogPageChange: (page: number) => void
  settings: AdminSettings
  setSettings: Dispatch<SetStateAction<AdminSettings>>
  onSaveSettings: () => void
}

function Pagination({ page, total, onChange }: { page: number; total: number; onChange: (page: number) => void }) {
  if (total <= 1) return null
  return <div className="admin-pagination">
    <button className="icon-button" disabled={page === 1} onClick={() => onChange(page - 1)} aria-label="上一页"><ChevronLeft size={17} /></button>
    <span>{page} / {total}</span>
    <button className="icon-button" disabled={page >= total} onClick={() => onChange(page + 1)} aria-label="下一页"><ChevronRight size={17} /></button>
  </div>
}

function PostContent({ props }: { props: AdminContentProps }) {
  const { stats, posts, listLoading, page, totalPages, onPageChange, onEditPost, onPublishPost, onModeratePost, onDeletePost } = props
  return <>
    <div className="admin-stats">
      <div><span>文章总数</span><strong>{String(stats.posts || stats.total).padStart(2, '0')}</strong><small>全部内容</small></div>
      <div><span>已发布</span><strong>{String(stats.published).padStart(2, '0')}</strong><small>公开内容</small></div>
      <div><span>总阅读</span><strong>{stats.views.toLocaleString()}</strong><small>累计访问</small></div>
      <div><span>总点赞</span><strong>{stats.likes.toLocaleString()}</strong><small>文章获赞</small></div>
      <div><span>总评论</span><strong>{stats.comments.toLocaleString()}</strong><small>公开评论</small></div>
      <div><span>总粉丝关系</span><strong>{stats.follows.toLocaleString()}</strong><small>用户关注</small></div>
    </div>
    <div className={`admin-table ${listLoading ? 'is-loading' : ''}`}>
      <div className="table-head"><span>文章标题</span><span>状态</span><span>更新时间</span><span>操作</span></div>
      {posts.map((post) => <div className="table-row" key={post.id}>
        <button className="table-title table-title-button" onClick={() => onEditPost(post)}><span>{post.title}</span></button>
        <button className="status-dot status-button" onClick={() => onPublishPost(post)}><i className={`is-${post.status}`} /> {statusLabels[post.status]}</button>
        <span>{formatDate(post.updatedAt ?? post.createdAt ?? post.publishedAt)}</span>
        <div className="table-actions">
          {post.moderationStatus === 'hidden' && <span className="takedown-flag">已下架</span>}
          <button className="icon-button" onClick={() => onModeratePost(post)} aria-label={post.moderationStatus === 'hidden' ? `恢复 ${post.title}` : `下架 ${post.title}`}>{post.moderationStatus === 'hidden' ? <Eye size={15} /> : <EyeOff size={15} />}</button>
          <button className="icon-button" onClick={() => onEditPost(post)} aria-label={`编辑 ${post.title}`}><Pencil size={15} /></button>
          <button className="icon-button" onClick={() => onDeletePost(post)} aria-label={`删除 ${post.title}`}><Trash2 size={15} /></button>
        </div>
      </div>)}
    </div>
    <Pagination page={page} total={totalPages} onChange={onPageChange} />
  </>
}

function UserContent({ props }: { props: AdminContentProps }) {
  const { stats, users, userTotal, userPage, userLoading, onUserPageChange, onToggleUser } = props
  const totalPages = Math.max(1, Math.ceil(userTotal / 20))
  return <>
    <div className="admin-stats">
      <div><span>注册用户</span><strong>{stats.users.toLocaleString()}</strong><small>全部账号</small></div>
      <div><span>文章作者</span><strong>{users.filter((item) => item.postCount > 0).length}</strong><small>当前页</small></div>
      <div><span>社区点赞</span><strong>{stats.likes.toLocaleString()}</strong><small>累计点赞</small></div>
    </div>
    <div className={`admin-table user-table ${userLoading ? 'is-loading' : ''}`}>
      <div className="table-head"><span>用户</span><span>文章 / 获赞</span><span>粉丝 / 关注</span><span>状态</span></div>
      {users.map((member) => <div className="table-row" key={member.id}>
        <div className="table-title"><UserRound size={16} /><span><strong>{member.nickname}</strong><small>@{member.username}</small></span></div>
        <span>{member.postCount} 篇 / {member.likeCount} 赞</span>
        <span>{member.followers} 粉丝 / {member.following} 关注</span>
        <button className="status-dot status-button" onClick={() => onToggleUser(member)}><i className={`is-${member.status}`} /> {userStatusLabels[member.status] ?? member.status}</button>
      </div>)}
    </div>
    <Pagination page={userPage} total={totalPages} onChange={onUserPageChange} />
  </>
}

function CommentContent({ props }: { props: AdminContentProps }) {
  const { stats, comments, commentTotal, commentPage, commentLoading, commentPostFilter, setCommentPostFilter, onCommentStatus, onCommentsBatch, onDeleteComment } = props
  const [selected, setSelected] = useState<number[]>([])
  const toggle = (id: number) => setSelected((current) => current.includes(id) ? current.filter((item) => item !== id) : [...current, id])
  const selectAll = () => setSelected(selected.length === comments.length ? [] : comments.map((comment) => comment.id))
  return <>
    <div className="admin-stats"><div><span>评论总数</span><strong>{commentTotal.toLocaleString()}</strong><small>全部评论</small></div><div><span>公开评论</span><strong>{stats.comments.toLocaleString()}</strong><small>文章页展示</small></div></div>
    {selected.length > 0 && onCommentsBatch && <div className="batch-toolbar"><span>已选择 {selected.length} 条</span><button className="button button-light" onClick={() => { onCommentsBatch(selected, 'published'); setSelected([]) }}>批量恢复</button><button className="button button-light" onClick={() => { onCommentsBatch(selected, 'hidden'); setSelected([]) }}>批量隐藏</button></div>}
    <div className={`admin-table has-selection ${commentLoading ? 'is-loading' : ''}`}>
      <div className="table-head"><button className="table-select-all" onClick={selectAll} aria-label="全选评论"><input type="checkbox" checked={comments.length > 0 && selected.length === comments.length} readOnly /></button><span>评论内容</span><span>文章 / 作者</span><span>状态</span><span>操作</span></div>
      {commentPostFilter && <div className="context-filter-bar">正在查看单篇文章的评论 <button onClick={() => setCommentPostFilter(undefined)}>查看全部</button></div>}
      {comments.map((comment) => <div className="table-row" key={comment.id}>
        <label className="table-select"><input type="checkbox" checked={selected.includes(comment.id)} onChange={() => toggle(comment.id)} aria-label={`选择评论 ${comment.id}`} /></label><div className="table-title"><MessageSquare size={16} /><span><strong>{comment.content.length > 60 ? `${comment.content.slice(0, 60)}…` : comment.content}</strong><small>{comment.pinned ? '已置顶 · ' : ''}{formatDate(comment.createdAt)}</small></span></div>
        <button className="status-button context-link" onClick={() => setCommentPostFilter(comment.postId)}><strong>{comment.postTitle}</strong><small>@{comment.author.nickname}</small></button>
        <button className="status-dot status-button" onClick={() => onCommentStatus(comment)}><i className={comment.status === 'hidden' ? 'is-archived' : ''} /> {commentStatusLabels[comment.status] ?? comment.status}</button>
        <div className="table-actions"><button className="icon-button" onClick={() => onDeleteComment(comment)} aria-label="删除评论"><Trash2 size={15} /></button></div>
      </div>)}
    </div>
    <Pagination page={commentPage} total={Math.max(1, Math.ceil(commentTotal / 20))} onChange={props.onCommentPageChange} />
  </>
}

function OverviewContent({ stats }: { stats: AdminStats }) {
  return <>
    <div className="admin-stats">
      <div><span>注册用户</span><strong>{stats.users.toLocaleString()}</strong><small>今日 +{stats.todayUsers}</small></div>
      <div><span>文章总数</span><strong>{stats.posts.toLocaleString()}</strong><small>今日 +{stats.todayPosts}</small></div>
      <div><span>公开评论</span><strong>{stats.comments.toLocaleString()}</strong><small>今日 +{stats.todayComments}</small></div>
      <div><span>总阅读</span><strong>{stats.views.toLocaleString()}</strong><small>累计访问</small></div>
      <div><span>总点赞</span><strong>{stats.likes.toLocaleString()}</strong><small>文章获赞</small></div>
      <div><span>关注关系</span><strong>{stats.follows.toLocaleString()}</strong><small>用户关注</small></div>
    </div>
    <div className="admin-table"><div className="table-head"><span>活跃用户</span><span>近 7 天发文</span><span>近 7 天评论</span><span>操作</span></div>{stats.activeUsers.map((item) => <div className="table-row" key={item.user.id}><div className="table-title"><UserRound size={16} /><span><strong>{item.user.nickname}</strong><small>@{item.user.username}</small></span></div><span>{item.postCount} 篇</span><span>{item.commentCount} 条</span><span><Link to={`/users/${item.user.username}`}>查看主页</Link></span></div>)}</div>
  </>
}

function TagContent({ props }: { props: AdminContentProps }) {
  const { tagData, newCategory, setNewCategory, onCreateCategory, onDeleteCategory, onDeleteTag } = props
  return <>
    <div className="admin-table"><div className="table-head"><span>分类</span><span>文章数</span><span>操作</span></div><div className="table-row"><div className="table-title"><Tag size={16} /><input className="inline-input" value={newCategory} onChange={(event) => setNewCategory(event.target.value)} placeholder="新分类名称" onKeyDown={(event) => { if (event.key === 'Enter') { event.preventDefault(); onCreateCategory() } }} /></div><span /><div className="table-actions"><button className="icon-button" onClick={onCreateCategory} aria-label="创建分类"><Plus size={15} /></button></div></div>{tagData.categories.map((category) => <div className="table-row" key={category.id}><div className="table-title"><Tag size={16} /><span>{category.name}</span></div><span>{category.usage} 篇</span><div className="table-actions"><button className="icon-button" onClick={() => onDeleteCategory(category)} aria-label={`删除分类 ${category.name}`}><Trash2 size={15} /></button></div></div>)}</div>
    <div className="admin-table"><div className="table-head"><span>标签</span><span>文章数</span><span>操作</span></div>{tagData.tags.map((tag) => <div className="table-row" key={tag.id}><div className="table-title"><Tag size={16} /><span>{tag.name}</span></div><span>{tag.usage} 篇</span><div className="table-actions"><button className="icon-button" onClick={() => onDeleteTag(tag)} aria-label={`删除标签 ${tag.name}`}><Trash2 size={15} /></button></div></div>)}</div>
  </>
}

function ReportContent({ props }: { props: AdminContentProps }) {
  const { reports, reportStatus, reportTotal, reportPage, reportLoading, onReportStatusChange, onReportPageChange, onReport, onReportsBatch } = props
  const [selected, setSelected] = useState<number[]>([])
  const toggle = (id: number) => setSelected((current) => current.includes(id) ? current.filter((item) => item !== id) : [...current, id])
  const selectAll = () => setSelected(selected.length === reports.length ? [] : reports.map((report) => report.id))
  return <>
    <div className="feed-tabs report-tabs">{['pending', 'handled', 'dismissed'].map((status) => <button key={status} className={reportStatus === status ? 'is-active' : ''} onClick={() => onReportStatusChange(status)}>{reportStatusLabels[status]}</button>)}</div>
    {selected.length > 0 && onReportsBatch && <div className="batch-toolbar"><span>已选择 {selected.length} 条</span><button className="button button-light" onClick={() => { onReportsBatch(selected, 'handled'); setSelected([]) }}>批量处理</button><button className="button button-light" onClick={() => { onReportsBatch(selected, 'dismissed'); setSelected([]) }}>批量驳回</button></div>}
    <div className={`admin-table has-selection ${reportLoading ? 'is-loading' : ''}`}><div className="table-head"><button className="table-select-all" onClick={selectAll} aria-label="全选举报"><input type="checkbox" checked={reports.length > 0 && selected.length === reports.length} readOnly /></button><span>举报理由</span><span>举报目标</span><span>状态</span><span>操作</span></div>{reports.length ? reports.map((report) => <div className="table-row" key={report.id}><label className="table-select"><input type="checkbox" checked={selected.includes(report.id)} onChange={() => toggle(report.id)} aria-label={`选择举报 ${report.id}`} /></label><div className="table-title"><Flag size={16} /><span><strong>{report.reason}</strong><small>@{report.reporter.username} · {formatDate(report.createdAt)}</small></span></div><span>{report.targetSummary}</span><span className="status-dot"><i className={report.status === 'pending' ? 'is-draft' : 'is-archived'} /> {reportStatusLabels[report.status]}</span><div className="table-actions">{report.status === 'pending' && <><button className="icon-button" onClick={() => onReport(report, 'handled')}>已处理</button><button className="icon-button" onClick={() => onReport(report, 'dismissed')}>驳回</button></>}</div></div>) : <div className="table-row"><span>当前没有{reportStatusLabels[reportStatus]}的举报。</span><span /><span /><span /><span /></div>}</div>
    <Pagination page={reportPage} total={Math.max(1, Math.ceil(reportTotal / 20))} onChange={onReportPageChange} />
  </>
}

function LogContent({ props }: { props: AdminContentProps }) {
  const { logs, logTotal, logPage, logLoading, onLogPageChange } = props
  return <>
    <div className={`admin-table ${logLoading ? 'is-loading' : ''}`}><div className="table-head"><span>操作</span><span>目标</span><span>详情</span><span>时间</span></div>{logs.length ? logs.map((entry) => <div className="table-row" key={entry.id}><div className="table-title"><History size={16} /><span><strong>{entry.action}</strong><small>@{entry.admin.username}</small></span></div><span>{entry.targetType} #{entry.targetId}</span><span>{entry.detail}</span><span>{formatDate(entry.createdAt)}</span></div>) : <div className="table-row"><span>还没有管理操作记录。</span><span /><span /><span /></div>}</div>
    <Pagination page={logPage} total={Math.max(1, Math.ceil(logTotal / 20))} onChange={onLogPageChange} />
  </>
}

function SettingsContent({ settings, setSettings, onSaveSettings }: Pick<AdminContentProps, 'settings' | 'setSettings' | 'onSaveSettings'>) {
  return <div className="admin-settings"><label className="checkbox-field"><input type="checkbox" checked={settings.openRegistration} onChange={(event) => setSettings({ ...settings, openRegistration: event.target.checked })} />开放公开注册（关闭后新用户无法注册）</label><label className="checkbox-field"><input type="checkbox" checked={settings.commentsEnabled} onChange={(event) => setSettings({ ...settings, commentsEnabled: event.target.checked })} />启用评论（关闭后全站不能发表评论）</label><button className="button button-dark" onClick={onSaveSettings}>保存设置</button></div>
}

export function AdminContent(props: AdminContentProps) {
  switch (props.active) {
    case '文章': return <PostContent props={props} />
    case '用户': return <UserContent props={props} />
    case '评论': return <CommentContent props={props} />
    case '概览': return <OverviewContent stats={props.stats} />
    case '标签': return <TagContent props={props} />
    case '举报': return <ReportContent props={props} />
    case '日志': return <LogContent props={props} />
    case '设置': return <SettingsContent settings={props.settings} setSettings={props.setSettings} onSaveSettings={props.onSaveSettings} />
    default: return <div className="admin-placeholder"><Settings size={22} /><h2>{props.active}模块</h2><p>该模块尚未开放。</p></div>
  }
}
