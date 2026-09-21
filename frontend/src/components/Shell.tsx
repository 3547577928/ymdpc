import { Link, NavLink, useLocation, useNavigate } from 'react-router-dom'
import { Bell, FileText, Github, House, Info, LogIn, Mail, PenLine, Rss, UserCircle, UserPlus } from 'lucide-react'
import { useEffect, useState } from 'react'
import { BrandMark } from './BrandMark'
import { getCurrentUser, getNotifications, logout, type AuthUser } from '../services/api'

const navItems = [
  { to: '/', label: '首页', icon: House },
  { to: '/posts', label: '文章', icon: FileText },
  { to: '/about', label: '关于', icon: Info },
]

export function Shell({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser>()
  const [unread, setUnread] = useState(0)
  const location = useLocation()
  const navigate = useNavigate()
  const transitionKey = `${location.key}-${location.pathname}${location.search}`

  useEffect(() => {
    getCurrentUser().then(setUser).catch(() => setUser(undefined))
  }, [location.pathname, location.search])

  // 登录后加载未读通知数，用于导航栏铃铛角标
  useEffect(() => {
    if (!user) {
      setUnread(0)
      return
    }
    const refreshUnread = () => getNotifications({ page: 1, pageSize: 1 }).then((data) => setUnread(data.unread)).catch(() => undefined)
    void refreshUnread()
    const timer = window.setInterval(refreshUnread, 30_000)
    return () => window.clearInterval(timer)
  }, [user, location.pathname])

  const signOut = async () => {
    await logout().catch(() => undefined)
    setUser(undefined)
    navigate('/')
  }

  return (
    <div className="site-shell">
      <header className="site-header">
        <div className="container header-inner">
          <Link to="/" className="wordmark">
            <BrandMark />
            <span>Quiet Signal</span>
          </Link>

          <nav className="primary-nav">
            {navItems.map((item) => {
              const Icon = item.icon
              return <NavLink key={item.to} className="nav-icon-link" data-tooltip={item.label} aria-label={item.label} to={item.to} end={item.to === '/'}>
                <Icon size={17} strokeWidth={1.8} />
              </NavLink>
            })}
            {user && <NavLink className="nav-icon-link" data-tooltip="关注" aria-label="关注" to="/posts?mode=following"><Rss size={17} strokeWidth={1.8} /></NavLink>}
            {user ? <><Link className="nav-icon-link nav-write" data-tooltip="写文章" aria-label="写文章" to="/write"><PenLine size={17} strokeWidth={1.8} /></Link><Link className="nav-icon-link nav-bell" data-tooltip="通知中心" to="/notifications" aria-label="通知中心"><Bell size={17} strokeWidth={1.8} />{unread > 0 && <span className="nav-bell-badge">{unread > 99 ? '99+' : unread}</span>}</Link><details className="nav-user-menu"><summary className="nav-icon-summary" data-tooltip={`${user.nickname}菜单`} aria-label={`${user.nickname}菜单`}><UserCircle size={17} strokeWidth={1.8} /></summary><div className="nav-dropdown"><Link to={`/users/${user.username}`}>个人主页</Link><Link to="/me/posts">我的文章</Link><Link to="/me/posts?status=draft">草稿箱</Link><Link to="/me/favorites">我的收藏</Link><Link to="/settings/profile">账号设置</Link><button className="nav-signout" onClick={() => void signOut()}>退出登录</button></div></details></> : <><Link className="nav-icon-link nav-user" data-tooltip="登录" aria-label="登录" to="/login"><LogIn size={17} strokeWidth={1.8} /></Link><Link className="nav-icon-link nav-write" data-tooltip="注册" aria-label="注册" to="/register"><UserPlus size={17} strokeWidth={1.8} /></Link></>}
          </nav>
        </div>
      </header>

      <main className="route-transition" key={transitionKey}>{children}</main>

      <footer className="site-footer">
        <div className="container footer-inner">
          <div>
            <div className="footer-title">Keep making things quieter.</div>
            <p>写代码、做产品，也记录那些还没有答案的问题。</p>
          </div>
          <div className="footer-links">
            <a href="https://github.com" target="_blank" rel="noreferrer" aria-label="GitHub"><Github size={17} /></a>
            <a href="mailto:hello@quietsig.dev" aria-label="邮件"><Mail size={17} /></a>
          </div>
        </div>
      </footer>
    </div>
  )
}
