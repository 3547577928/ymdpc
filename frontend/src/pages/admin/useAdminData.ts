import { useCallback, useEffect, useState } from 'react'
import { getAdminComments, getAdminLogs, getAdminReports, getAdminSettings, getAdminStats, getAdminTags, getAdminUsers, getPosts, type AdminSettings, type AdminStats, type AdminUser, type AuthUser } from '../../services/api'
import type { AdminComment, AdminLogEntry, AdminReport, PostSummary, TagUsage } from '../../types'

const pageSize = 20

const emptyStats: AdminStats = { total: 0, users: 0, posts: 0, published: 0, views: 0, likes: 0, comments: 0, follows: 0, todayUsers: 0, todayPosts: 0, todayComments: 0, activeUsers: [] }
const emptySettings: AdminSettings = { openRegistration: true, commentsEnabled: true }

type SetError = (message: string) => void

function errorMessage(reason: unknown, fallback: string) {
  return reason instanceof Error ? reason.message : fallback
}

export function useAdminData(options: {
  user?: AuthUser
  active: string
  page: number
  userPage: number
  commentPage: number
  commentPostFilter?: number
  reportPage: number
  reportStatus: string
  logPage: number
  showEditor: boolean
  setError: SetError
}) {
  const { user, active, page, userPage, commentPage, commentPostFilter, reportPage, reportStatus, logPage, showEditor, setError } = options
  const [posts, setPosts] = useState<PostSummary[]>([])
  const [stats, setStats] = useState<AdminStats>(emptyStats)
  const [users, setUsers] = useState<AdminUser[]>([])
  const [userTotal, setUserTotal] = useState(0)
  const [userLoading, setUserLoading] = useState(false)
  const [comments, setComments] = useState<AdminComment[]>([])
  const [commentTotal, setCommentTotal] = useState(0)
  const [commentLoading, setCommentLoading] = useState(false)
  const [tagData, setTagData] = useState<{ tags: TagUsage[]; categories: TagUsage[] }>({ tags: [], categories: [] })
  const [newCategory, setNewCategory] = useState('')
  const [reports, setReports] = useState<AdminReport[]>([])
  const [reportTotal, setReportTotal] = useState(0)
  const [reportLoading, setReportLoading] = useState(false)
  const [logs, setLogs] = useState<AdminLogEntry[]>([])
  const [logTotal, setLogTotal] = useState(0)
  const [logLoading, setLogLoading] = useState(false)
  const [settings, setSettings] = useState<AdminSettings>(emptySettings)
  const [listLoading, setListLoading] = useState(false)

  const loadAdmin = useCallback(async () => {
    setListLoading(true)
    setError('')
    try {
      const [postPage, nextStats] = await Promise.all([getPosts({ page, pageSize }, true), getAdminStats()])
      setPosts(postPage.items)
      setStats(nextStats)
    } catch (reason) {
      setError(errorMessage(reason, '读取后台数据失败'))
    } finally {
      setListLoading(false)
    }
  }, [page, setError])

  const loadUsers = useCallback(async () => {
    setUserLoading(true)
    try {
      const data = await getAdminUsers({ page: userPage, pageSize })
      setUsers(data.items)
      setUserTotal(data.total)
    } catch (reason) {
      setError(errorMessage(reason, '读取用户失败'))
    } finally {
      setUserLoading(false)
    }
  }, [setError, userPage])

  const loadComments = useCallback(async () => {
    setCommentLoading(true)
    try {
      const data = await getAdminComments({ page: commentPage, pageSize, postId: commentPostFilter })
      setComments(data.items)
      setCommentTotal(data.total)
    } catch (reason) {
      setError(errorMessage(reason, '读取评论失败'))
    } finally {
      setCommentLoading(false)
    }
  }, [commentPage, commentPostFilter, setError])

  const loadTags = useCallback(async () => {
    try {
      setTagData(await getAdminTags())
    } catch (reason) {
      setError(errorMessage(reason, '读取标签失败'))
    }
  }, [setError])

  const loadReports = useCallback(async () => {
    setReportLoading(true)
    try {
      const data = await getAdminReports({ page: reportPage, pageSize, status: reportStatus })
      setReports(data.items)
      setReportTotal(data.total)
    } catch (reason) {
      setError(errorMessage(reason, '读取举报失败'))
    } finally {
      setReportLoading(false)
    }
  }, [reportPage, reportStatus, setError])

  const loadLogs = useCallback(async () => {
    setLogLoading(true)
    try {
      const data = await getAdminLogs({ page: logPage, pageSize })
      setLogs(data.items)
      setLogTotal(data.total)
    } catch (reason) {
      setError(errorMessage(reason, '读取日志失败'))
    } finally {
      setLogLoading(false)
    }
  }, [logPage, setError])

  const loadSettings = useCallback(async () => {
    try {
      setSettings(await getAdminSettings())
    } catch (reason) {
      setError(errorMessage(reason, '读取设置失败'))
    }
  }, [setError])

  useEffect(() => {
    if (user) void loadAdmin()
  }, [loadAdmin, user])

  useEffect(() => {
    if (!user) return
    if (active === '用户') void loadUsers()
    if (active === '评论') void loadComments()
    if (active === '标签') void loadTags()
    if (active === '举报') void loadReports()
    if (active === '日志') void loadLogs()
    if (active === '设置') void loadSettings()
  }, [active, loadComments, loadLogs, loadReports, loadSettings, loadTags, loadUsers, user])

  useEffect(() => {
    if (user && showEditor && tagData.categories.length === 0) void loadTags()
  }, [loadTags, showEditor, tagData.categories.length, user])

  return {
    posts, stats, users, userTotal, userLoading, comments, commentTotal, commentLoading,
    tagData, setTagData, newCategory, setNewCategory, reports, reportTotal, reportLoading,
    logs, logTotal, logLoading, settings, setSettings, listLoading,
    loadAdmin, loadUsers, loadComments, loadTags, loadReports, loadLogs, loadSettings,
  }
}

export type AdminData = ReturnType<typeof useAdminData>
