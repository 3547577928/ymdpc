import { Plus } from 'lucide-react'
import type { FormEvent } from 'react'
import { useState } from 'react'
import { Link } from 'react-router-dom'
import { createCategory, createPost, deleteCategory, deleteComment, deletePost, deleteTag, downloadAdminExport, getAdminPost, handleAdminReport, handleAdminReports, login, logout, updateAdminCommentStatus, updateAdminCommentsStatus, updateAdminSettings, updateAdminUser, updatePost, updatePostModeration, updatePostStatus, type PostInput } from '../services/api'
import { useAuth } from '../services/auth'
import { usePageMeta } from '../utils/usePageMeta'
import type { AdminComment, AdminReport, PostStatus, PostSummary, TagUsage } from '../types'
import { ConfirmDialog } from '../components/Dialog'
import { AdminContent } from './admin/AdminContent'
import { AdminLogin } from './admin/AdminLogin'
import { AdminPostEditor } from './admin/AdminPostEditor'
import { AdminSidebar } from './admin/AdminSidebar'
import { useAdminData } from './admin/useAdminData'

const emptyDraft: PostInput = { title: '', slug: '', summary: '', content: '', coverImage: '', tags: [], status: 'draft', featured: false, scheduledAt: null }
const pageSize = 20
type ConfirmRequest = { title: string; message: string; action: () => Promise<void> }

export function AdminPage() {
  usePageMeta('管理后台')
  // 登录态由 AuthProvider 全局共享
  const { user, loading, setUser } = useAuth()
  const [active, setActive] = useState('文章')
  const [page, setPage] = useState(1)
  const [userPage, setUserPage] = useState(1)
  const [commentPage, setCommentPage] = useState(1)
  const [commentPostFilter, setCommentPostFilter] = useState<number>()
  const [reportStatus, setReportStatus] = useState('pending')
  const [reportPage, setReportPage] = useState(1)
  const [logPage, setLogPage] = useState(1)
  const [showEditor, setShowEditor] = useState(false)
  const [editingID, setEditingID] = useState<number>()
  const [draft, setDraft] = useState<PostInput>(emptyDraft)
  const [loginForm, setLoginForm] = useState({ username: '', password: '' })
  const [editorLoading, setEditorLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [confirmRequest, setConfirmRequest] = useState<ConfirmRequest>()

  // 非管理员不触发任何管理端请求：仅角色确认为 admin 时才把 user 传给数据层
  const adminUser = user?.role === 'admin' ? user : undefined
  const data = useAdminData({ user: adminUser, active, page, userPage, commentPage, commentPostFilter, reportPage, reportStatus, logPage, showEditor, setError })

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
      setDraft({ title: detail.title, slug: detail.slug, summary: detail.summary, content: detail.content, coverImage: detail.coverImage, tags: detail.tags, status: detail.status, featured: detail.featured, categoryId: detail.category?.id ?? null, scheduledAt: detail.scheduledAt ?? null })
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
      await data.loadAdmin()
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
      await data.loadAdmin()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '更新状态失败')
    }
  }

  const handleDelete = (post: PostSummary) => {
    setConfirmRequest({ title: '删除文章', message: `确定删除「${post.title}」吗？`, action: async () => {
      try {
        await deletePost(post.id)
        if (data.posts.length === 1 && page > 1) setPage((current) => current - 1)
        else await data.loadAdmin()
      } catch (reason) {
        setError(reason instanceof Error ? reason.message : '删除文章失败')
      }
    } })
  }

  const handleModeration = async (post: PostSummary) => {
    try {
      await updatePostModeration(post.id, post.moderationStatus === 'hidden' ? 'normal' : 'hidden')
      await data.loadAdmin()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '更新审核状态失败')
    }
  }

  const handleCommentStatus = async (comment: AdminComment) => {
    try {
      await updateAdminCommentStatus(comment.id, comment.status === 'hidden' ? 'published' : 'hidden')
      await data.loadComments()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '更新评论状态失败')
    }
  }

  const handleCommentsBatch = async (ids: number[], status: 'published' | 'hidden') => {
    try {
      await updateAdminCommentsStatus(ids, status)
      await data.loadComments()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '批量更新评论失败')
    }
  }

  const handleRemoveComment = (comment: AdminComment) => {
    setConfirmRequest({ title: '删除评论', message: '确定删除这条评论吗？', action: async () => {
      try {
        await deleteComment(comment.id)
        if (data.comments.length === 1 && commentPage > 1) setCommentPage((current) => current - 1)
        else await data.loadComments()
      } catch (reason) {
        setError(reason instanceof Error ? reason.message : '删除评论失败')
      }
    } })
  }

  const handleCreateCategory = async () => {
    if (!data.newCategory.trim()) return
    try {
      await createCategory(data.newCategory.trim())
      data.setNewCategory('')
      await data.loadTags()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '创建分类失败')
    }
  }

  const handleDeleteCategory = (category: TagUsage) => {
    setConfirmRequest({ title: '删除分类', message: `确定删除分类「${category.name}」吗？相关文章将变为未分类。`, action: async () => {
      try {
        await deleteCategory(category.id)
        await data.loadTags()
      } catch (reason) {
        setError(reason instanceof Error ? reason.message : '删除分类失败')
      }
    } })
  }

  const handleDeleteTag = (tag: TagUsage) => {
    setConfirmRequest({ title: '删除标签', message: `确定删除标签「${tag.name}」吗？`, action: async () => {
      try {
        await deleteTag(tag.id)
        await data.loadTags()
      } catch (reason) {
        setError(reason instanceof Error ? reason.message : '删除标签失败')
      }
    } })
  }

  const handleReport = async (report: AdminReport, status: 'handled' | 'dismissed', resolution = '') => {
    try {
      await handleAdminReport(report.id, status, resolution)
      await data.loadReports()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '处理举报失败')
    }
  }

  const handleReportsBatch = async (ids: number[], status: 'handled' | 'dismissed', resolution = '') => {
    try {
      await handleAdminReports(ids, status, resolution)
      await data.loadReports()
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '批量处理举报失败')
    }
  }

  const handleSaveSettings = async () => {
    try {
      data.setSettings(await updateAdminSettings(data.settings))
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '保存设置失败')
    }
  }

  if (loading) return <div className="container page-state"><span className="eyebrow">Admin</span><h1>正在验证登录状态。</h1></div>
  if (!user) return <AdminLogin username={loginForm.username} password={loginForm.password} error={error} onUsernameChange={(username) => setLoginForm({ ...loginForm, username })} onPasswordChange={(password) => setLoginForm({ ...loginForm, password })} onSubmit={handleLogin} />
  // 普通登录用户直接访问 /admin 时，不能看到后台界面，只能看到无权限提示
  if (!adminUser) {
    return <div className="container page-state"><span className="eyebrow">Admin</span><h1>需要管理员权限。</h1><p>当前账号（{user.nickname}）没有后台访问权限。</p><p><Link className="text-button" to="/">返回首页</Link></p></div>
  }

  return <section className="admin-page">
    <AdminSidebar active={active} onChange={setActive} onLogout={async () => { await logout().catch(() => undefined); setUser(undefined) }} />
    <div className="admin-main">
      <header className="admin-topbar"><div><div className="eyebrow">Admin / {active}</div><h1>{active}</h1></div>{active === '文章' && <button className="button button-dark" onClick={openCreate}><Plus size={16} /> 写新文章</button>}</header>
      {error && <div className="form-error admin-error">{error}</div>}
      <AdminContent active={active} stats={data.stats} posts={data.posts} listLoading={data.listLoading} page={page} totalPages={Math.max(1, Math.ceil(data.stats.total / pageSize))} onPageChange={setPage} onEditPost={(post) => void openEdit(post)} onPublishPost={(post) => void handlePublish(post)} onModeratePost={(post) => void handleModeration(post)} onDeletePost={handleDelete} users={data.users} userTotal={data.userTotal} userPage={userPage} userLoading={data.userLoading} onUserPageChange={setUserPage} onToggleUser={(member) => void updateAdminUser(member.id, { status: member.status === 'active' ? 'muted' : 'active' }).then(data.loadUsers).catch((reason: Error) => setError(reason.message))} comments={data.comments} commentTotal={data.commentTotal} commentPage={commentPage} commentLoading={data.commentLoading} commentPostFilter={commentPostFilter} setCommentPostFilter={setCommentPostFilter} onCommentPageChange={setCommentPage} onCommentStatus={(comment) => void handleCommentStatus(comment)} onCommentsBatch={(ids, status) => void handleCommentsBatch(ids, status)} onDeleteComment={handleRemoveComment} tagData={data.tagData} newCategory={data.newCategory} setNewCategory={data.setNewCategory} onCreateCategory={() => void handleCreateCategory()} onDeleteCategory={handleDeleteCategory} onDeleteTag={handleDeleteTag} reports={data.reports} reportStatus={reportStatus} reportTotal={data.reportTotal} reportPage={reportPage} reportLoading={data.reportLoading} onReportStatusChange={(status) => { setReportStatus(status); setReportPage(1) }} onReportPageChange={setReportPage} onReport={(report, status, resolution) => void handleReport(report, status, resolution)} onReportsBatch={(ids, status, resolution) => void handleReportsBatch(ids, status, resolution)} onExport={(type) => void downloadAdminExport(type).catch((reason: Error) => setError(reason.message))} logs={data.logs} logTotal={data.logTotal} logPage={logPage} logLoading={data.logLoading} onLogPageChange={setLogPage} settings={data.settings} setSettings={data.setSettings} onSaveSettings={() => void handleSaveSettings()} />
    </div>
    {showEditor && <AdminPostEditor editingID={editingID} editorLoading={editorLoading} saving={saving} error={error} draft={draft} categories={data.tagData.categories} setDraft={setDraft} onClose={() => setShowEditor(false)} onSubmit={handleSubmit} onSaveDraft={() => void submitDraft('draft')} />}
    <ConfirmDialog open={Boolean(confirmRequest)} title={confirmRequest?.title ?? ''} message={confirmRequest?.message} onCancel={() => setConfirmRequest(undefined)} onConfirm={async () => { const action = confirmRequest?.action; setConfirmRequest(undefined); if (action) await action() }} />
  </section>
}
