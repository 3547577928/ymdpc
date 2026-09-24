import { CalendarDays, Edit3, UserPlus, UserRound, UserRoundCheck } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { followUser, getUserProfile, unfollowUser, type UserProfile } from '../services/api'
import { useAuth } from '../services/auth'
import { usePageMeta } from '../utils/usePageMeta'
import { PostCard } from '../components/PostCard'
import { formatDate } from '../utils'

export function ProfilePage() {
  const { username = '' } = useParams()
  const [profile, setProfile] = useState<UserProfile>()
  const { user } = useAuth()
  usePageMeta(profile ? profile.user.nickname : '个人主页', profile?.user.bio || undefined)
  const currentUserID = user?.id
  const [page, setPage] = useState(1)
  const pageSize = 12
  const [loading, setLoading] = useState(true)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  // 切换用户时回到第一页
  useEffect(() => { setPage(1) }, [username])
  useEffect(() => {
    setLoading(true)
    getUserProfile(username, { page, pageSize }).then(setProfile).catch((reason: Error) => setError(reason.message)).finally(() => setLoading(false))
  }, [username, page])

  const toggleFollow = async () => {
    if (!profile || !currentUserID || currentUserID === profile.user.id) return
    setBusy(true)
    try {
      const data = profile.followingMe ? await unfollowUser(profile.user.id) : await followUser(profile.user.id)
      setProfile({ ...profile, followingMe: data.following, followers: data.followers })
    } catch (reason) {
      setError(reason instanceof Error ? reason.message : '操作失败')
    } finally {
      setBusy(false)
    }
  }

  if (loading) return <div className="container page-state"><span className="eyebrow">Profile</span><h1>正在读取用户。</h1></div>
  if (error || !profile) return <div className="container page-state"><span className="eyebrow">Profile</span><h1>用户暂时无法加载。</h1><p>{error}</p></div>

  const isSelf = currentUserID === profile.user.id
  return <section className="profile-page container">
    <header className="profile-header"><div className="profile-avatar">{profile.user.avatar ? <img src={profile.user.avatar} alt="" /> : <UserRound size={34} />}</div><div className="profile-heading"><div className="eyebrow">Member / @{profile.user.username}</div><h1>{profile.user.nickname}</h1><p>{profile.user.bio || '还没有写个人简介。'}</p><span className="profile-joined"><CalendarDays size={14} /> 加入于 {profile.user.createdAt ? formatDate(profile.user.createdAt) : '最近'}</span></div><div className="profile-actions">{isSelf ? <Link className="button button-light" to="/settings/profile"><Edit3 size={15} /> 编辑资料</Link> : currentUserID ? <button className="button button-dark" onClick={() => void toggleFollow()} disabled={busy}>{profile.followingMe ? <UserRoundCheck size={15} /> : <UserPlus size={15} />} {profile.followingMe ? '已关注' : '关注'}</button> : <Link className="button button-dark" to={`/login?from=${encodeURIComponent(`/users/${profile.user.username}`)}`}>登录后关注</Link>}</div></header>
    <div className="profile-stats"><div><strong>{profile.postCount}</strong><span>文章</span></div><div><strong>{profile.likeCount}</strong><span>获赞</span></div><div><strong>{profile.followers}</strong><span>粉丝</span></div><div><strong>{profile.following}</strong><span>关注</span></div></div>
    <div className="profile-content"><div><div className="eyebrow">Published notes</div><h2>最近发布</h2></div>{profile.posts.length ? <div className="post-grid">{profile.posts.map((post) => <PostCard key={post.id} post={post} />)}</div> : <div className="empty-state"><h2>还没有公开文章</h2><p>等作者写下第一篇文章。</p></div>}{profile.postCount > pageSize && <div className="admin-pagination"><button className="icon-button" disabled={page === 1} onClick={() => setPage((current) => current - 1)}>上一页</button><span>{page} / {Math.max(1, Math.ceil(profile.postCount / pageSize))}</span><button className="icon-button" disabled={page >= Math.ceil(profile.postCount / pageSize)} onClick={() => setPage((current) => current + 1)}>下一页</button></div>}</div>
  </section>
}
