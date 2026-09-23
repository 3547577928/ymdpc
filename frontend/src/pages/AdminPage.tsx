import { ChevronLeft, ChevronRight, Eye, EyeOff, FileText, Flag, History, LayoutDashboard, LogOut, MessageSquare, Pencil, Plus, Settings, Tag, Trash2, UploadCloud, UserRound, X } from 'lucide-react'
import { FormEvent, useCallback, useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { createCategory, createPost, deleteCategory, deleteComment, deletePost, deleteTag, getAdminComments, getAdminLogs, getAdminPost, getAdminReports, getAdminSettings, getAdminStats, getAdminTags, getAdminUsers, getCurrentUser, getPosts, handleAdminReport, login, logout, updateAdminCommentStatus, updateAdminSettings, updateAdminUser, updatePost, updatePostModeration, updatePostStatus, type AdminSettings, type AdminStats, type AdminUser, type AuthUser, type PostInput } from '../services/api'
import type { AdminComment, AdminLogEntry, AdminReport, PostStatus, PostSummary, TagUsage } from '../types'
import { formatDate } from '../utils'

const emptyDraft: PostInput = { title: '', slug: '', summary: '', content: '', coverImage: '', tags: [], status: 'draft', featured: false }
const pageSize = 20
const statusLabels: Record<PostStatus, string> = { draft: '草稿', published: '已发布', archived: '已归档' }
const userStatusLabels: Record<string, string> = { active: '正常', muted: '禁言', banned: '封禁' }
const commentStatusLabels: Record<string, string> = { published: '公开', hidden: '已隐藏' }
const reportStatusLabels: Record<string, string> = { pending: '待处理', handled: '已处理', dismissed: '已驳回' }
const emptySettings: AdminSettings = { openRegistration: true, commentsEnabled: true }

export function AdminPage() {
  const [user, setUser] = useState<AuthUser>()
  const [active, setActive] = useState('文章')
  const [posts, setPosts] = useState<PostSummary[]>([])
  const [stats, setStats] = useState<AdminStats>({ total: 0, users: 0, posts: 0, published: 0, views: 0, likes: 0, comments: 0, follows: 0, todayUsers: 0, todayPosts: 0, todayComments: 0, activeUsers: [] })
  const [users, setUsers] = useState<AdminUser[]>([])
  const [userTotal, setUserTotal] = useState(0)
  const [userPage, setUserPage] = useState(1)
  const [userLoading, setUserLoading] = useState(false)
  const [comments, setComments] = useState<AdminComment[]>([])
  const [commentTotal, setCommentTotal] = useState(0)
  const [commentPage, setCommentPage] = useState(1)
  const [commentLoading, setCommentLoading] = useState(false)
  const [commentPostFilter, setCommentPostFilter] = useState<number>()
  const [tagData, setTagData] = useState<{ tags: TagUsage[]; categories: TagUsage[] }>({ tags: [], categories: [] })
  const [newCategory, setNewCategory] = useState('')
  const [reports, setReports] = useState<AdminReport[]>([])
  const [reportStatus, setReportStatus] = useState('pending')
  const [reportTotal, setReportTotal] = useState(0)
  const [reportPage, setReportPage] = useState(1)
  const [reportLoading, setReportLoading] = useState(false)
  const [logs, setLogs] = useState<AdminLogEntry[]>([])
  const [logTotal, setLogTotal] = useState(0)
  const [logPage, setLogPage] = useState(1)
  const [logLoading, setLogLoading] = useState(false)
  const [settings, setSettings] = useState<AdminSettings>(emptySettings)
  const [page, setPage] = useState(1)
  const [showEditor, setShowEditor] = useState(false)
  const [editingID, setEditingID] = useState<number>()
  const [draft, setDraft] = useState<PostInput>(emptyDraft)
  const [loginForm, setLoginForm] = useState({ username: 'admin', password: '' })
  const [loading, setLoading] = useState(true)
  const [listLoading, setListLoading] = useState(false)
  const [editorLoading, setEditorLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    getCurrentUser().then(setUser).catch(() => undefined).finally(() => setLoading(false))
  }, [])

  const loadAdmin = useCallback(async () => {
    setListLoading(true)
    setError('')
    try {
      const [postPage, nextStats] = await Promise.all([getPosts({ page, pageSize }, true), getAdminStats()])
      setPosts(postPage.items)
      setStats(nextStats)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取后台数据失败')
    } finally {
      setListLoading(false)
    }
  }, [page])

  const loadUsers = useCallback(async () => {
    setUserLoading(true)
    try {
      const data = await getAdminUsers({ page: userPage, pageSize: 20 })
      setUsers(data.items)
      setUserTotal(data.total)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取用户失败')
    } finally {
      setUserLoading(false)
    }
  }, [userPage])

  const loadComments = useCallback(async () => {
    setCommentLoading(true)
    try {
      const data = await getAdminComments({ page: commentPage, pageSize: 20, postId: commentPostFilter })
      setComments(data.items)
      setCommentTotal(data.total)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取评论失败')
    } finally {
      setCommentLoading(false)
    }
  }, [commentPage, commentPostFilter])

  const loadTags = useCallback(async () => {
    try {
      setTagData(await getAdminTags())
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取标签失败')
    }
  }, [])

  const loadReports = useCallback(async () => {
    setReportLoading(true)
    try {
      const data = await getAdminReports({ page: reportPage, pageSize: 20, status: reportStatus })
      setReports(data.items)
      setReportTotal(data.total)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取举报失败')
    } finally {
      setReportLoading(false)
    }
  }, [reportPage, reportStatus])

  const loadLogs = useCallback(async () => {
    setLogLoading(true)
    try {
      const data = await getAdminLogs({ page: logPage, pageSize: 20 })
      setLogs(data.items)
      setLogTotal(data.total)
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取日志失败')
    } finally {
      setLogLoading(false)
    }
  }, [logPage])

  const loadSettings = useCallback(async () => {
    try {
      setSettings(await getAdminSettings())
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '读取设置失败')
    }
  }, [])

  useEffect(() => {
    if (user) void loadAdmin()
  }, [user, loadAdmin])

  useEffect(() => {
    if (user && active === '用户') void loadUsers()
    if (user && active === '评论') void loadComments()
    if (user && active === '标签') void loadTags()
    if (user && active === '举报') void loadReports()
    if (user && active === '日志') void loadLogs()
    if (user && active === '设置') void loadSettings()
  }, [user, active, loadUsers, loadComments, loadTags, loadReports, loadLogs, loadSettings])

  // 编辑器需要分类选项，但标签数据默认只在“标签”页加载，这里按需补一次
  useEffect(() => {
    if (user && showEditor && tagData.categories.length === 0) void loadTags()
  }, [user, showEditor, tagData.categories.length, loadTags])

  const handleLogin = async (event: FormEvent) => {
    event.preventDefault()
    setError('')
    try {
      setUser(await login(loginForm.username, loginForm.password))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '登录失败')
    }
  }

  const openCreate = () => {
    setEditingID(undefined)
    setDraft(emptyDraft)
    setShowEditor(true)
  }

  const openEdit = async (post: PostSummary) => {
    setEditingID(post.id)
    setShowEditor(true)
    setEditorLoading(true)
    setError('')
    try {
      const detail = await getAdminPost(post.id)
      // categoryId 必须回传：后端更新时直接赋值 input.CategoryID，缺失会被当成清空分类
      setDraft({ title: detail.title, slug: detail.slug, summary: detail.summary, content: detail.content, coverImage: detail.coverImage, tags: detail.tags, status: detail.status, featured: detail.featured, categoryId: detail.category?.id ?? null })
    } catch (reason) {
      setShowEditor(false)
      setError(reason instanceof Error ? reason.message : '读取文章失败')
    } finally {
      setEditorLoading(false)
    }
  }

  const submitDraft = async (status: PostStatus) => {
    if (!draft.title.trim()) {
      setError('文章标题不能为空')
      return
    }
    setSaving(true)
    setError('')
    try {
      if (editingID) await updatePost(editingID, { ...draft, status })
      else await createPost({ ...draft, status })
      setShowEditor(false)
      setEditingID(undefined)
      setDraft(emptyDraft)
      await loadAdmin()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '保存文章失败')
    } finally {
      setSaving(false)
    }
  }

  const handleSubmit = (event: FormEvent) => {
    event.preventDefault()
    void submitDraft(editingID ? draft.status : 'published')
  }

  const handlePublish = async (post: PostSummary) => {
    try {
      await updatePostStatus(post.id, post.status === 'published' ? 'draft' : 'published')
      await loadAdmin()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '更新状态失败')
    }
  }

  const handleDelete = async (post: PostSummary) => {
    if (!window.confirm(`确定删除「${post.title}」吗？`)) return
    try {
      await deletePost(post.id)
      if (posts.length === 1 && page > 1) setPage((current) => current - 1)
      else await loadAdmin()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '删除文章失败')
    }
  }

  // 下架或恢复文章：下架后公开页面不可见，管理后台仍保留内容
  const handleModeration = async (post: PostSummary) => {
    try {
      await updatePostModeration(post.id, post.moderationStatus === 'hidden' ? 'normal' : 'hidden')
      await loadAdmin()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '更新审核状态失败')
    }
  }

  // 隐藏或恢复评论
  const handleCommentStatus = async (comment: AdminComment) => {
    try {
      await updateAdminCommentStatus(comment.id, comment.status === 'hidden' ? 'published' : 'hidden')
      await loadComments()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '更新评论状态失败')
    }
  }

  const handleRemoveComment = async (comment: AdminComment) => {
    if (!window.confirm('确定删除这条评论吗？')) return
    try {
      await deleteComment(comment.id)
      if (comments.length === 1 && commentPage > 1) setCommentPage((current) => current - 1)
      else await loadComments()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '删除评论失败')
    }
  }

  // 创建分类
  const handleCreateCategory = async () => {
    if (!newCategory.trim()) return
    try {
      await createCategory(newCategory.trim())
      setNewCategory('')
      await loadTags()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '创建分类失败')
    }
  }

  const handleDeleteCategory = async (category: TagUsage) => {
    if (!window.confirm(`确定删除分类「${category.name}」吗？相关文章将变为未分类。`)) return
    try {
      await deleteCategory(category.id)
      await loadTags()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '删除分类失败')
    }
  }

  const handleDeleteTag = async (tag: TagUsage) => {
    if (!window.confirm(`确定删除标签「${tag.name}」吗？`)) return
    try {
      await deleteTag(tag.id)
      await loadTags()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '删除标签失败')
    }
  }

  // 处理或驳回举报
  const handleReport = async (report: AdminReport, status: 'handled' | 'dismissed') => {
    try {
      await handleAdminReport(report.id, status)
      await loadReports()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '处理举报失败')
    }
  }

  // 保存系统设置
  const handleSaveSettings = async () => {
    try {
      setSettings(await updateAdminSettings(settings))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '保存设置失败')
    }
  }

  if (loading) return <div className="container page-state"><span className="eyebrow">Admin</span><h1>正在验证登录状态。</h1></div>
  if (!user) return <section className="admin-login-page"><form className="admin-login-card" onSubmit={handleLogin}><div className="admin-logo">QS <span>CONTROL</span></div><div className="eyebrow">Private workspace</div><h1>登录管理后台</h1><p>管理文章、草稿和公开内容。</p><label>用户名<input autoComplete="username" value={loginForm.username} onChange={(event) => setLoginForm({ ...loginForm, username: event.target.value })} /></label><label>密码<input type="password" autoComplete="current-password" value={loginForm.password} onChange={(event) => setLoginForm({ ...loginForm, password: event.target.value })} /></label>{error && <div className="form-error">{error}</div>}<button className="button button-dark" type="submit">登录后台</button></form></section>

  const totalPages = Math.max(1, Math.ceil(stats.total / pageSize))

  return (
    <section className="admin-page">
      <div className="admin-sidebar"><div className="admin-logo">QS <span>CONTROL</span></div><div className="admin-nav">{[['概览', LayoutDashboard], ['文章', FileText], ['评论', MessageSquare], ['用户', UserRound], ['标签', Tag], ['举报', Flag], ['日志', History], ['设置', Settings]].map(([label, Icon]) => <button key={String(label)} className={active === label ? 'is-active' : ''} onClick={() => setActive(String(label))}><Icon size={16} /> {String(label)}</button>)}</div><button className="admin-logout" onClick={async () => { await logout(); setUser(undefined) }}><LogOut size={16} /> 退出登录</button></div>
      <div className="admin-main"><header className="admin-topbar"><div><div className="eyebrow">Admin / {active}</div><h1>{active}</h1></div>{active === '文章' && <button className="button button-dark" onClick={openCreate}><Plus size={16} /> 写新文章</button>}</header>
        {error && <div className="form-error admin-error">{error}</div>}
        {active === '文章' ? <><div className="admin-stats"><div><span>文章总数</span><strong>{String(stats.posts || stats.total).padStart(2, '0')}</strong><small>全部内容</small></div><div><span>已发布</span><strong>{String(stats.published).padStart(2, '0')}</strong><small>公开内容</small></div><div><span>总阅读</span><strong>{stats.views.toLocaleString()}</strong><small>累计访问</small></div><div><span>总点赞</span><strong>{stats.likes.toLocaleString()}</strong><small>文章获赞</small></div><div><span>总评论</span><strong>{stats.comments.toLocaleString()}</strong><small>公开评论</small></div><div><span>总粉丝关系</span><strong>{stats.follows.toLocaleString()}</strong><small>用户关注</small></div></div><div className={`admin-table ${listLoading ? 'is-loading' : ''}`}><div className="table-head"><span>文章标题</span><span>状态</span><span>更新时间</span><span>操作</span></div>{posts.map((post) => <div className="table-row" key={post.id}><button className="table-title table-title-button" onClick={() => void openEdit(post)}><FileText size={16} /><span>{post.title}</span></button><button className="status-dot status-button" onClick={() => void handlePublish(post)}><i className={`is-${post.status}`} /> {statusLabels[post.status]}</button><span>{formatDate(post.updatedAt ?? post.createdAt ?? post.publishedAt)}</span><div className="table-actions">{post.moderationStatus === 'hidden' && <span className="takedown-flag">已下架</span>}<button className="icon-button" onClick={() => void handleModeration(post)} aria-label={post.moderationStatus === 'hidden' ? `恢复 ${post.title}` : `下架 ${post.title}`}>{post.moderationStatus === 'hidden' ? <Eye size={15} /> : <EyeOff size={15} />}</button><button className="icon-button" onClick={() => void openEdit(post)} aria-label={`编辑 ${post.title}`}><Pencil size={15} /></button><button className="icon-button" onClick={() => void handleDelete(post)} aria-label={`删除 ${post.title}`}><Trash2 size={15} /></button></div></div>)}</div>{totalPages > 1 && <div className="admin-pagination"><button className="icon-button" disabled={page === 1} onClick={() => setPage((current) => current - 1)} aria-label="上一页"><ChevronLeft size={17} /></button><span>{page} / {totalPages}</span><button className="icon-button" disabled={page === totalPages} onClick={() => setPage((current) => current + 1)} aria-label="下一页"><ChevronRight size={17} /></button></div>}</> : active === '用户' ? <><div className="admin-stats"><div><span>注册用户</span><strong>{stats.users.toLocaleString()}</strong><small>全部账号</small></div><div><span>文章作者</span><strong>{users.filter((item) => item.postCount > 0).length}</strong><small>当前页</small></div><div><span>社区点赞</span><strong>{stats.likes.toLocaleString()}</strong><small>累计点赞</small></div></div><div className={`admin-table user-table ${userLoading ? 'is-loading' : ''}`}><div className="table-head"><span>用户</span><span>文章 / 获赞</span><span>粉丝 / 关注</span><span>状态</span></div>{users.map((member) => <div className="table-row" key={member.id}><div className="table-title"><UserRound size={16} /><span><strong>{member.nickname}</strong><small>@{member.username}</small></span></div><span>{member.postCount} 篇 / {member.likeCount} 赞</span><span>{member.followers} 粉丝 / {member.following} 关注</span><button className="status-dot status-button" onClick={() => void updateAdminUser(member.id, { status: member.status === 'active' ? 'muted' : 'active' }).then(loadUsers).catch((reason: Error) => setError(reason.message))}><i className={`is-${member.status}`} /> {userStatusLabels[member.status] ?? member.status}</button></div>)}</div>{Math.max(1, Math.ceil(userTotal / 20)) > 1 && <div className="admin-pagination"><button className="icon-button" disabled={userPage === 1} onClick={() => setUserPage((current) => current - 1)} aria-label="上一页"><ChevronLeft size={17} /></button><span>{userPage} / {Math.ceil(userTotal / 20)}</span><button className="icon-button" disabled={userPage >= Math.ceil(userTotal / 20)} onClick={() => setUserPage((current) => current + 1)} aria-label="下一页"><ChevronRight size={17} /></button></div>}</> : active === '评论' ? <><div className="admin-stats"><div><span>评论总数</span><strong>{commentTotal.toLocaleString()}</strong><small>全部评论</small></div><div><span>公开评论</span><strong>{stats.comments.toLocaleString()}</strong><small>文章页展示</small></div></div><div className={`admin-table ${commentLoading ? 'is-loading' : ''}`}><div className="table-head"><span>评论内容</span><span>文章 / 作者</span><span>状态</span><span>操作</span></div>{commentPostFilter && <div className="context-filter-bar">正在查看单篇文章的评论 <button onClick={() => setCommentPostFilter(undefined)}>查看全部</button></div>}{comments.map((comment) => <div className="table-row" key={comment.id}><div className="table-title"><MessageSquare size={16} /><span><strong>{comment.content.length > 60 ? `${comment.content.slice(0, 60)}…` : comment.content}</strong><small>{formatDate(comment.createdAt)}</small></span></div><button className="status-button context-link" onClick={() => setCommentPostFilter(comment.postId)} title="查看这篇文章的全部评论（上下文）"><strong>{comment.postTitle}</strong><small>@{comment.author.nickname}</small></button><button className="status-dot status-button" onClick={() => void handleCommentStatus(comment)}><i className={comment.status === 'hidden' ? 'is-archived' : ''} /> {commentStatusLabels[comment.status] ?? comment.status}</button><div className="table-actions"><button className="icon-button" onClick={() => void handleRemoveComment(comment)} aria-label="删除评论"><Trash2 size={15} /></button></div></div>)}</div>{Math.max(1, Math.ceil(commentTotal / 20)) > 1 && <div className="admin-pagination"><button className="icon-button" disabled={commentPage === 1} onClick={() => setCommentPage((current) => current - 1)} aria-label="上一页"><ChevronLeft size={17} /></button><span>{commentPage} / {Math.ceil(commentTotal / 20)}</span><button className="icon-button" disabled={commentPage >= Math.ceil(commentTotal / 20)} onClick={() => setCommentPage((current) => current + 1)} aria-label="下一页"><ChevronRight size={17} /></button></div>}</> : active === '概览' ? <><div className="admin-stats"><div><span>注册用户</span><strong>{stats.users.toLocaleString()}</strong><small>今日 +{stats.todayUsers}</small></div><div><span>文章总数</span><strong>{stats.posts.toLocaleString()}</strong><small>今日 +{stats.todayPosts}</small></div><div><span>公开评论</span><strong>{stats.comments.toLocaleString()}</strong><small>今日 +{stats.todayComments}</small></div><div><span>总阅读</span><strong>{stats.views.toLocaleString()}</strong><small>累计访问</small></div><div><span>总点赞</span><strong>{stats.likes.toLocaleString()}</strong><small>文章获赞</small></div><div><span>关注关系</span><strong>{stats.follows.toLocaleString()}</strong><small>用户关注</small></div></div><div className="admin-table"><div className="table-head"><span>活跃用户</span><span>近 7 天发文</span><span>近 7 天评论</span><span>操作</span></div>{stats.activeUsers.map((item) => <div className="table-row" key={item.user.id}><div className="table-title"><UserRound size={16} /><span><strong>{item.user.nickname}</strong><small>@{item.user.username}</small></span></div><span>{item.postCount} 篇</span><span>{item.commentCount} 条</span><span><Link to={`/users/${item.user.username}`}>查看主页</Link></span></div>)}</div></> : active === '标签' ? <><div className="admin-table"><div className="table-head"><span>分类</span><span>文章数</span><span>操作</span></div><div className="table-row"><div className="table-title"><Tag size={16} /><input className="inline-input" value={newCategory} onChange={(event) => setNewCategory(event.target.value)} placeholder="新分类名称" onKeyDown={(event) => { if (event.key === 'Enter') { event.preventDefault(); void handleCreateCategory() } }} /></div><span /><div className="table-actions"><button className="icon-button" onClick={() => void handleCreateCategory()} aria-label="创建分类"><Plus size={15} /></button></div></div>{tagData.categories.map((category) => <div className="table-row" key={category.id}><div className="table-title"><Tag size={16} /><span>{category.name}</span></div><span>{category.usage} 篇</span><div className="table-actions"><button className="icon-button" onClick={() => void handleDeleteCategory(category)} aria-label={`删除分类 ${category.name}`}><Trash2 size={15} /></button></div></div>)}</div><div className="admin-table"><div className="table-head"><span>标签</span><span>文章数</span><span>操作</span></div>{tagData.tags.map((tag) => <div className="table-row" key={tag.id}><div className="table-title"><Tag size={16} /><span>{tag.name}</span></div><span>{tag.usage} 篇</span><div className="table-actions"><button className="icon-button" onClick={() => void handleDeleteTag(tag)} aria-label={`删除标签 ${tag.name}`}><Trash2 size={15} /></button></div></div>)}</div></> : active === '举报' ? <><div className="feed-tabs report-tabs">{['pending', 'handled', 'dismissed'].map((status) => <button key={status} className={reportStatus === status ? 'is-active' : ''} onClick={() => { setReportStatus(status); setReportPage(1) }}>{reportStatusLabels[status]}</button>)}</div><div className={`admin-table ${reportLoading ? 'is-loading' : ''}`}><div className="table-head"><span>举报理由</span><span>举报目标</span><span>状态</span><span>操作</span></div>{reports.length ? reports.map((report) => <div className="table-row" key={report.id}><div className="table-title"><Flag size={16} /><span><strong>{report.reason}</strong><small>@{report.reporter.username} · {formatDate(report.createdAt)}</small></span></div><span>{report.targetSummary}</span><span className="status-dot"><i className={report.status === 'pending' ? 'is-draft' : 'is-archived'} /> {reportStatusLabels[report.status]}</span><div className="table-actions">{report.status === 'pending' && <><button className="icon-button" onClick={() => void handleReport(report, 'handled')} aria-label="标记已处理">已处理</button><button className="icon-button" onClick={() => void handleReport(report, 'dismissed')} aria-label="驳回举报">驳回</button></>}</div></div>) : <div className="table-row"><span>当前没有{reportStatusLabels[reportStatus]}的举报。</span><span /><span /><span /></div>}</div>{Math.max(1, Math.ceil(reportTotal / 20)) > 1 && <div className="admin-pagination"><button className="icon-button" disabled={reportPage === 1} onClick={() => setReportPage((current) => current - 1)} aria-label="上一页"><ChevronLeft size={17} /></button><span>{reportPage} / {Math.max(1, Math.ceil(reportTotal / 20))}</span><button className="icon-button" disabled={reportPage >= Math.ceil(reportTotal / 20)} onClick={() => setReportPage((current) => current + 1)} aria-label="下一页"><ChevronRight size={17} /></button></div>}</> : active === '日志' ? <><div className={`admin-table ${logLoading ? 'is-loading' : ''}`}><div className="table-head"><span>操作</span><span>目标</span><span>详情</span><span>时间</span></div>{logs.length ? logs.map((entry) => <div className="table-row" key={entry.id}><div className="table-title"><History size={16} /><span><strong>{entry.action}</strong><small>@{entry.admin.username}</small></span></div><span>{entry.targetType} #{entry.targetId}</span><span>{entry.detail}</span><span>{formatDate(entry.createdAt)}</span></div>) : <div className="table-row"><span>还没有管理操作记录。</span><span /><span /><span /></div>}</div>{Math.max(1, Math.ceil(logTotal / 20)) > 1 && <div className="admin-pagination"><button className="icon-button" disabled={logPage === 1} onClick={() => setLogPage((current) => current - 1)} aria-label="上一页"><ChevronLeft size={17} /></button><span>{logPage} / {Math.max(1, Math.ceil(logTotal / 20))}</span><button className="icon-button" disabled={logPage >= Math.ceil(logTotal / 20)} onClick={() => setLogPage((current) => current + 1)} aria-label="下一页"><ChevronRight size={17} /></button></div>}</> : active === '设置' ? <div className="admin-settings"><label className="checkbox-field"><input type="checkbox" checked={settings.openRegistration} onChange={(event) => setSettings({ ...settings, openRegistration: event.target.checked })} />开放公开注册（关闭后新用户无法注册）</label><label className="checkbox-field"><input type="checkbox" checked={settings.commentsEnabled} onChange={(event) => setSettings({ ...settings, commentsEnabled: event.target.checked })} />启用评论（关闭后全站不能发表评论）</label><button className="button button-dark" onClick={() => void handleSaveSettings()}>保存设置</button></div> : <div className="admin-placeholder"><Settings size={22} /><h2>{active}模块</h2><p>该模块尚未开放。</p></div>}
      </div>
      {showEditor && <div className="modal-backdrop" onClick={() => setShowEditor(false)}><form className="editor-modal" onSubmit={handleSubmit} onClick={(event) => event.stopPropagation()} aria-busy={editorLoading}><div className="modal-head"><div><div className="eyebrow">{editingID ? 'Edit post' : 'New post'}</div><h2>{editingID ? '编辑文章' : '写一篇新文章'}</h2></div><button type="button" className="icon-button" onClick={() => setShowEditor(false)} aria-label="关闭"><X size={17} /></button></div>{editorLoading ? <div className="editor-loading">正在读取文章。</div> : <>{error && <div className="form-error">{error}</div>}<div className="editor-grid"><label>标题<input required value={draft.title} onChange={(event) => setDraft({ ...draft, title: event.target.value })} placeholder="输入文章标题" /></label><label>链接标识<input value={draft.slug} onChange={(event) => setDraft({ ...draft, slug: event.target.value })} placeholder="留空则根据标题生成" /></label></div><label>摘要<textarea value={draft.summary} onChange={(event) => setDraft({ ...draft, summary: event.target.value })} placeholder="用一句话说明这篇文章" /></label><div className="editor-grid"><label>标签<input value={draft.tags.join(', ')} onChange={(event) => setDraft({ ...draft, tags: event.target.value.split(',').map((tag) => tag.trim()).filter(Boolean) })} placeholder="React, Go" /></label>{editingID && <label>状态<select value={draft.status} onChange={(event) => setDraft({ ...draft, status: event.target.value as PostStatus })}><option value="draft">草稿</option><option value="published">已发布</option><option value="archived">已归档</option></select></label>}<label>分类<select value={draft.categoryId ?? ''} onChange={(event) => setDraft({ ...draft, categoryId: event.target.value ? Number(event.target.value) : null })}><option value="">不设分类</option>{tagData.categories.map((category) => <option key={category.id} value={category.id}>{category.name}</option>)}</select></label></div><label>封面地址<input value={draft.coverImage} onChange={(event) => setDraft({ ...draft, coverImage: event.target.value })} placeholder="https://..." /></label><label className="checkbox-field"><input type="checkbox" checked={draft.featured} onChange={(event) => setDraft({ ...draft, featured: event.target.checked })} />设为精选文章</label><label>正文<textarea required className="editor-textarea" value={draft.content} onChange={(event) => setDraft({ ...draft, content: event.target.value })} placeholder="支持 Markdown" /></label><div className="editor-actions">{!editingID && <button type="button" className="button button-light" onClick={() => void submitDraft('draft')} disabled={saving}>保存草稿</button>}<button type="submit" className="button button-dark" disabled={saving}><UploadCloud size={16} /> {saving ? '保存中' : editingID ? '保存修改' : '发布文章'}</button></div></>}</form></div>}
    </section>
  )
}
