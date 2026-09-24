import { Link, useLocation, useNavigate } from 'react-router-dom'
import { Bell, FileText, Github, House, Info, LogIn, Mail, MessagesSquare, Moon, PenLine, Rss, Search, Sun, UserCircle, UserPlus, type LucideIcon } from 'lucide-react'
import { applyTheme, getTheme, type Theme } from '../utils/theme'
import { useEffect, useState } from 'react'
import { BrandMark } from './BrandMark'
import { AUTH_EXPIRED_EVENT, logout, type AuthUser } from '../services/api'
import { useAuth } from '../services/auth'
import { useQueryClient } from '@tanstack/react-query'
import { UNREAD_QUERY_KEY, useUnreadCount } from '../services/queries'

type NavItem = { to: string; label: string; icon: LucideIcon }

const navItems: NavItem[] = [
  { to: '/', label: '首页', icon: House },
  { to: '/posts', label: '文章', icon: FileText },
  { to: '/forum', label: '论坛', icon: MessagesSquare },
  { to: '/search', label: '搜索', icon: Search },
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
          <Link to="/me/stats">数据看板</Link>
          <Link to="/me/posts?status=draft">草稿箱</Link>
          <Link to="/me/favorites">我的收藏</Link>
          <Link to="/settings/profile">账号设置</Link>
          <button className="nav-signout" onClick={onSignOut}>退出登录</button>
        </div>
      </details>
    </>
  )
}

// 未登录时的导航区：登录与注册入口；登录链接带上当前页，登录后回跳
function NavGuestLinks() {
  const location = useLocation()
  return (
    <>
      <Link className="nav-icon-link nav-user" data-tooltip="登录" aria-label="登录" to={`/login?from=${encodeURIComponent(location.pathname + location.search)}`}><LogIn size={17} strokeWidth={1.8} /></Link>
      <Link className="nav-icon-link nav-write" data-tooltip="注册" aria-label="注册" to="/register"><UserPlus size={17} strokeWidth={1.8} /></Link>
    </>
  )
}

export function Shell({ children }: { children: React.ReactNode }) {
  // 登录用户由 AuthProvider 全局共享，不再每个页面各自请求 /auth/me
  const { user, setUser } = useAuth()
  // 未读角标：React Query 缓存 + SSE 实时推送，30s 轮询兜底
  const { data: unread = 0 } = useUnreadCount(Boolean(user))
  const queryClient = useQueryClient()
  const location = useLocation()
  const [theme, setTheme] = useState<Theme>(getTheme)
  useEffect(() => applyTheme(theme), [theme])
  const navigate = useNavigate()
  // 只按 pathname 过渡：key 包含 location.key/search 时，搜索防抖、翻页等查询参数变化
  // 会整页重挂载（每页 4+ 个请求重发、滚动丢失），页面组件自己响应查询参数变化即可
  const transitionKey = location.pathname

  // /posts 同时承载文章列表和关注 feed，选中项只能结合 mode 参数判断，否则两个导航图标会同时高亮
  const activeNavTo = location.pathname === '/posts' && new URLSearchParams(location.search).get('mode') === 'following' ? followingNavItem.to : location.pathname
  const navIconLink = ({ to, label, icon: Icon }: NavItem) => {
    const [path] = to.split('?')
    // 首页要求完全匹配，“文章”这类入口在其子路径（如 /posts/:slug）下也保持选中
    const active = to === activeNavTo || (!to.includes('?') && path !== '/' && location.pathname.startsWith(`${path}/`))
    return <Link key={to} className={active ? 'nav-icon-link active' : 'nav-icon-link'} data-tooltip={label} aria-label={label} aria-current={active ? 'page' : undefined} to={to}><Icon size={17} strokeWidth={1.8} /></Link>
  }

  // 受保护请求返回 401 时统一清理导航状态，并回到对应的登录入口。
  useEffect(() => {
    const handleAuthExpired = () => {
      setUser(undefined)
      queryClient.setQueryData(UNREAD_QUERY_KEY, 0)
      if (location.pathname.startsWith('/admin')) {
        navigate('/admin', { replace: true })
      } else if (location.pathname !== '/login' && location.pathname !== '/register') {
        navigate(`/login?from=${encodeURIComponent(location.pathname + location.search)}`, { replace: true })
      }
    }
    window.addEventListener(AUTH_EXPIRED_EVENT, handleAuthExpired)
    return () => window.removeEventListener(AUTH_EXPIRED_EVENT, handleAuthExpired)
  }, [location.pathname, location.search, navigate, queryClient, setUser])

  // SSE 实时未读推送：新通知（评论/点赞/关注/粉丝新文）秒级亮角标；
  // 断线时 EventSource 自动重连，React Query 的 30s 轮询作为最终兜底
  useEffect(() => {
    if (!user) return
    const source = new EventSource('/api/notifications/stream')
    source.onmessage = (event) => {
      const count = Number(event.data)
      if (!Number.isNaN(count)) queryClient.setQueryData(UNREAD_QUERY_KEY, count)
    }
    return () => source.close()
  }, [user, queryClient])

  const signOut = async () => {
    await logout().catch(() => undefined)
    setUser(undefined)
    navigate('/')
  }

  return (
    <div className="site-shell">
      <a className="skip-link" href="#main-content">跳到主要内容</a>
      <header className="site-header">
        <div className="container header-inner">
          <Link to="/" className="wordmark">
            <BrandMark />
            <span>Quiet Signal</span>
          </Link>

          <nav className="primary-nav">
            {navItems.map(navIconLink)}
            {user && navIconLink(followingNavItem)}
            <button className="nav-icon-link theme-toggle" data-tooltip={theme === 'dark' ? '浅色模式' : '深色模式'} aria-label={theme === 'dark' ? '切换到浅色模式' : '切换到深色模式'} onClick={() => setTheme(theme === 'dark' ? 'light' : 'dark')}>
              {theme === 'dark' ? <Sun size={17} strokeWidth={1.8} /> : <Moon size={17} strokeWidth={1.8} />}
            </button>
            {user
              ? <NavUserMenu user={user} unread={unread} onSignOut={() => void signOut()} />
              : <NavGuestLinks />}
          </nav>
        </div>
      </header>

      <main id="main-content" className="route-transition" key={transitionKey}>{children}</main>

      <footer className="site-footer">
        <div className="container footer-inner">
          <div>
            <div className="footer-title">Keep making things quieter.</div>
            <p>一个安静的写作社区：写下界面的细节、系统的取舍，以及还没有答案的问题。</p>
          </div>
          <div className="footer-links">
            <a href="/api/feed.xml" target="_blank" rel="noreferrer" aria-label="RSS 订阅"><Rss size={17} /></a>
            <a href="https://github.com" target="_blank" rel="noreferrer" aria-label="GitHub"><Github size={17} /></a>
            <a href="mailto:hello@quietsig.dev" aria-label="邮件"><Mail size={17} /></a>
          </div>
        </div>
      </footer>
    </div>
  )
}
