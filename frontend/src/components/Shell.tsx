import { Link, NavLink } from 'react-router-dom'
import { ArrowUpRight, Github, Mail, Menu, X } from 'lucide-react'
import { useState } from 'react'
import { BrandMark } from './BrandMark'

const navItems = [
  { to: '/', label: '首页' },
  { to: '/posts', label: '文章' },
  { to: '/about', label: '关于' },
]

export function Shell({ children }: { children: React.ReactNode }) {
  const [open, setOpen] = useState(false)

  return (
    <div className="site-shell">
      <header className="site-header">
        <div className="container header-inner">
          <Link to="/" className="wordmark" onClick={() => setOpen(false)}>
            <BrandMark />
            <span>Quiet Signal</span>
          </Link>

          <button className="icon-button mobile-menu" onClick={() => setOpen((value) => !value)} aria-label="打开菜单">
            {open ? <X size={19} /> : <Menu size={19} />}
          </button>

          <nav className={`primary-nav ${open ? 'is-open' : ''}`}>
            {navItems.map((item) => (
              <NavLink key={item.to} to={item.to} end={item.to === '/'} onClick={() => setOpen(false)}>
                {item.label}
              </NavLink>
            ))}
            <Link className="nav-admin" to="/admin" onClick={() => setOpen(false)}>
              管理后台 <ArrowUpRight size={14} />
            </Link>
          </nav>
        </div>
      </header>

      <main>{children}</main>

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
