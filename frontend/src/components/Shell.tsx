import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Bell, FileText, Github, House, Info, LogIn, Mail, PenLine, Rss, UserCircle, UserPlus, type LucideIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { BrandMark } from './BrandMark'
import { AUTH_EXPIRED_EVENT, getCurrentUser, getNotifications, logout, type AuthUser } from '../services/api'

type NavItem = { to: string; label: string; icon: LucideIcon }

const navItems: NavItem[] = [
  { to: '/', label: '首页', icon: House },
  { to: '/posts', label: '文章', icon: FileText },
  { to: '/about', label: '关于', icon: Info },
]

// 关注 feed 和文章列表共用 /posts，只靠 mode 参数区分，单独列出便于计算选中态
const followingNavItem: NavItem = { to: '/posts?mode=following', label: '关注', icon: Rss }

// 已登录用户的导航区：写文章入口、通知铃铛（带未读角标）与个人下拉菜单
function NavUserMenu({ user, unread, onSignOut }: { user: AuthUser; unread: number; onSignOut: () => void }) {
  return (
    <>
      <Link className="nav-icon-link nav-write" data-tooltip="写文章" aria-label="写文章" to="/write"><PenLine size={17} strokeWidth={1.8} /></Link>
      <Link className="nav-icon-link nav-bell" data-tooltip="通知中心" to="/notifications" aria-label="通知中心">
        <Bell size={17} strokeWidth={1.8} />
        {unread > 0 && <span className="nav-bell-badge">{unread > 99 ? '99+' : unread}</span>}
      </Link>
      <details className="nav-user-menu">
        <summary className="nav-icon-summary" data-tooltip={`${user.nickname}菜单`} aria-label={`${user.nickname}菜单`}><UserCircle size={17} strokeWidth={1.8} /></summary>
        <div className="nav-dropdown">
          <Link to={`/users/${user.username}`}>个人主页</Link>
          <Link to="/me/posts">我的文章</Link>
          <Link to="/me/posts?status=draft">草稿箱</Link>
          <Link to="/me/favorites">我的收藏</Link>
          <Link to="/settings/profile">账号设置</Link>
          <button className="nav-signout" onClick={onSignOut}>退出登录</button>
        </div>
      </details>
    </>
  )
}

// 未登录时的导航区：登录与注册入口
function NavGuestLinks() {
  return (
    <>
      <Link className="nav-icon-link nav-user" data-tooltip="登录" aria-label="登录" to="/login"><LogIn size={17} strokeWidth={1.8} /></Link>
      <Link className="nav-icon-link nav-write" data-tooltip="注册" aria-label="注册" to="/register"><UserPlus size={17} strokeWidth={1.8} /></Link>
    </>
  )
}

export function Shell({ children }: { children: React.ReactNode }) {
  const [user, setUser] = useState<AuthUser>()
  const [unread, setUnread] = useState(0)
  const location = useLocation()
  const navigate = useNavigate()
  const transitionKey = `${location.key}-${location.pathname}${location.search}`

  // /posts 同时承载文章列表和关注 feed，选中项只能结合 mode 参数判断，否则两个导航图标会同时高亮
  const activeNavTo = location.pathname === '/posts' && new URLSearchParams(location.search).get('mode') === 'following' ? followingNavItem.to : location.pathname
  const navIconLink = ({ to, label, icon: Icon }: NavItem) => {
    const [path] = to.split('?')
    // 首页要求完全匹配，“文章”这类入口在其子路径（如 /posts/:slug）下也保持选中
    const active = to === activeNavTo || (!to.includes('?') && path !== '/' && location.pathname.startsWith(`${path}/`))
    return <Link key={to} className={active ? 'nav-icon-link active' : 'nav-icon-link'} data-tooltip={label} aria-label={label} aria-current={active ? 'page' : undefined} to={to}><Icon size={17} strokeWidth={1.8} /></Link>
  }

  useEffect(() => {
    getCurrentUser().then(setUser).catch(() => setUser(undefined))
  }, [location.pathname, location.search])

  // 受保护请求返回 401 时统一清理导航状态，并回到对应的登录入口。
  useEffect(() => {
    const handleAuthExpired = () => {
      setUser(undefined)
      setUnread(0)
      if (location.pathname.startsWith('/admin')) {
        navigate('/admin', { replace: true })
      } else if (location.pathname !== '/login' && location.pathname !== '/register') {
        navigate('/login', { replace: true })
      }
    }
    window.addEventListener(AUTH_EXPIRED_EVENT, handleAuthExpired)
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handleAuthExpired)
  }, [location.pathname, navigate])

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
            {navItems.map(navIconLink)}
            {user && navIconLink(followingNavItem)}
            {user
              ? <NavUserMenu user={user} unread={unread} onSignOut={() => void signOut()} />
              : <NavGuestLinks />}
          </nav>
        </div>
      </header>

      <main className="route-transition" key={transitionKey}>{children}</main>

      <footer className="site-footer">
        <div className="container footer-inner">
          <div>
            <div className="footer-title">Keep making things quieter.</div>
            <p>一个安静的写作社区：写下界面的细节、系统的取舍，以及还没有答案的问题。</p>
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
