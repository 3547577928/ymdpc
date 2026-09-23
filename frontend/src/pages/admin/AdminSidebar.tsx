import { FileText, Flag, History, LayoutDashboard, LogOut, MessageSquare, Settings, Tag, UserRound } from 'lucide-react'

const items = [
  ['概览', LayoutDashboard],
  ['文章', FileText],
  ['评论', MessageSquare],
  ['用户', UserRound],
  ['标签', Tag],
  ['举报', Flag],
  ['日志', History],
  ['设置', Settings],
] as const

type AdminSidebarProps = {
  active: string
  onChange: (active: string) => void
  onLogout: () => void
}

export function AdminSidebar({ active, onChange, onLogout }: AdminSidebarProps) {
  return <aside className="admin-sidebar">
    <div className="admin-logo">QS <span>CONTROL</span></div>
    <div className="admin-nav">
      {items.map(([label, Icon]) => <button key={label} className={active === label ? 'is-active' : ''} onClick={() => onChange(label)}><Icon size={16} /> {label}</button>)}
    </div>
    <button className="admin-logout" onClick={onLogout}><LogOut size={16} /> 退出登录</button>
  </aside>
}
